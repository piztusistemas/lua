package ficha

// Ler e escribir o ficheiro .md: front matter YAML entre dúas liñas «---» e
// despois o corpo. O corpo consérvase EXACTO (é o que permite que un .matex
// importado volva saír byte a byte): Escribir pon sempre unha liña en branco
// tras o «---» de peche, e Ler quita exactamente esa.

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrSenFrontMatter: o ficheiro non empeza por «---», así que non é unha
// ficha (pode ser unha nota calquera da bóveda).
var ErrSenFrontMatter = errors.New("non é unha ficha: falta o bloque «---» do principio")

// Ler interpreta o contido dun ficheiro .md de ficha.
func Ler(datos []byte) (*Ficha, error) {
	t := strings.TrimPrefix(string(datos), "\ufeff")
	t = strings.ReplaceAll(t, "\r\n", "\n")
	if !strings.HasPrefix(t, "---\n") {
		return nil, ErrSenFrontMatter
	}
	resto := t[len("---\n"):]
	var yamlTxt, corpo string
	switch {
	case strings.HasPrefix(resto, "---\n") || resto == "---":
		// front matter baleiro
		corpo = strings.TrimPrefix(strings.TrimPrefix(resto, "---"), "\n")
	default:
		i := strings.Index(resto, "\n---\n")
		if i < 0 {
			if strings.HasSuffix(resto, "\n---") {
				i = len(resto) - len("\n---")
			} else {
				return nil, errors.New("non é unha ficha: falta o «---» que pecha o front matter")
			}
		}
		yamlTxt = resto[:i+1]
		corpo = resto[min(len(resto), i+len("\n---\n")):]
	}
	corpo = strings.TrimPrefix(corpo, "\n")

	f := &Ficha{}
	if err := yaml.Unmarshal([]byte(yamlTxt), f); err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	if f.Luayang == 0 {
		return nil, errors.New("non é unha ficha de LuaYang: falta a clave «luayang»")
	}
	if f.Luayang > Version {
		return nil, fmt.Errorf("a ficha é dunha versión máis nova de LuaYang (%d; esta le ata a %d)", f.Luayang, Version)
	}
	if len(f.Extra) == 0 {
		f.Extra = nil
	}
	f.Corpo = corpo
	return f, nil
}

// Escribir devolve o contido do ficheiro .md da ficha.
func Escribir(f *Ficha) ([]byte, error) {
	copia := *f
	if copia.Luayang == 0 {
		copia.Luayang = Version
	}
	var b bytes.Buffer
	b.WriteString("---\n")
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&copia); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	b.WriteString("---\n\n")
	b.WriteString(f.Corpo)
	return b.Bytes(), nil
}
