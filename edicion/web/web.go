// Package web contén o frontend compartido do módulo de edición (JS e CSS) e
// o circuíto para levalo ás aplicacións.
//
// Fonte única de verdade: os ficheiros viven aquí, en edicion/web/estatico/.
// As aplicacións NON os editan a man; reciben unha copia xerada en
// frontend/src/comun/ xunto cun MANIFEST.sha256:
//
//	cd edicion
//	go run ./cmd/sincronizarweb ../yang/yang/frontend/src/comun \
//	                                ../lua/lua/frontend/src/comun
//
// Cada aplicación ten ademais un test que chama a Verificar e falla se a copia
// deriva, de xeito que a duplicación non poida volver en silencio.
//
// Por que copia e non importar directamente fóra de frontend/: así cada
// aplicación segue sendo autónoma para `npm test`, `npm run dev` e
// `vite build` (non precisa o cartafol irmán edicion/) e non fai falla
// configurar Vite para servir fóra do seu root (§6.1 do plan).
package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:estatico
var estatico embed.FS

// prefixo é o cartafol dos recursos dentro do FS embebido; os nomes que
// devolven Ficheiros/Contido xa van sen el (relativos ao root do frontend).
const prefixo = "estatico/"

// NomeManifest é o ficheiro que acompaña á copia xerada e rexistra o sha256 de
// cada ficheiro, para que calquera deriva se vexa a simple vista.
const NomeManifest = "MANIFEST.sha256"

// ComandoSincronizar é a orde exacta que reproduce a copia, para incluíla nas
// mensaxes de erro (o erro ten que dicir como se arranxa).
const ComandoSincronizar = "cd edicion && go run ./cmd/sincronizarweb ../yang/yang/frontend/src/comun ../lua/lua/frontend/src/comun"

// Ficheiros devolve os nomes relativos (sempre con "/") dos ficheiros do
// frontend común, ordenados alfabeticamente.
func Ficheiros() ([]string, error) {
	var nomes []string
	err := fs.WalkDir(estatico, "estatico", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		nomes = append(nomes, strings.TrimPrefix(filepath.ToSlash(ruta), prefixo))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(nomes)
	return nomes, nil
}

// Contido devolve o contido canónico dun ficheiro do frontend común.
func Contido(nome string) ([]byte, error) {
	return estatico.ReadFile(prefixo + path.Clean(filepath.ToSlash(nome)))
}

// Hash devolve o sha256 (hex) do contido canónico dun ficheiro.
func Hash(nome string) (string, error) {
	contido, err := Contido(nome)
	if err != nil {
		return "", err
	}
	return hashDe(contido), nil
}

func hashDe(contido []byte) string {
	suma := sha256.Sum256(contido)
	return hex.EncodeToString(suma[:])
}

// Manifest devolve o texto do MANIFEST.sha256 para o contido canónico actual:
// unha liña por ficheiro, "<sha256>  <nome>", coma sha256sum.
func Manifest() (string, error) {
	nomes, err := Ficheiros()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, nome := range nomes {
		h, err := Hash(nome)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s  %s\n", h, nome)
	}
	return b.String(), nil
}

// Sincronizar escribe a copia xerada do frontend común en `destino` (un
// cartafol, que se crea se non existe) e mais o MANIFEST.sha256. Tamén retira
// os ficheiros que figuren no manifesto anterior e xa non existan na fonte,
// para que non queden copias mortas.
func Sincronizar(destino string) error {
	if err := os.MkdirAll(destino, 0o755); err != nil {
		return err
	}

	// Limpeza do que sobra respecto do manifesto anterior (ou da copia
	// anterior, se non había manifesto).
	if previos, err := ficheirosEnDisco(destino); err == nil {
		nomes, err := Ficheiros()
		if err != nil {
			return err
		}
		esperados := make(map[string]bool, len(nomes))
		for _, nome := range nomes {
			esperados[nome] = true
		}
		for _, previo := range previos {
			if esperados[previo] {
				continue
			}
			if err := os.Remove(filepath.Join(destino, filepath.FromSlash(previo))); err != nil {
				return err
			}
			limparCartafolesBaleiros(destino, previo)
		}
	}

	nomes, err := Ficheiros()
	if err != nil {
		return err
	}
	for _, nome := range nomes {
		contido, err := Contido(nome)
		if err != nil {
			return err
		}
		ruta := filepath.Join(destino, filepath.FromSlash(nome))
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(ruta, contido, 0o644); err != nil {
			return err
		}
	}
	manifest, err := Manifest()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destino, NomeManifest), []byte(manifest), 0o644)
}

// Verificar comproba que a copia xerada en `destino` coincide exactamente coa
// fonte (nin falta ningún ficheiro, nin sobra, nin ningún cambiou de
// contido). Devolve nil se todo está en orde.
func Verificar(destino string) error {
	nomes, err := Ficheiros()
	if err != nil {
		return err
	}
	esperados := make(map[string]bool, len(nomes))
	for _, nome := range nomes {
		esperados[nome] = true
		canonico, err := Contido(nome)
		if err != nil {
			return err
		}
		ruta := filepath.Join(destino, filepath.FromSlash(nome))
		copia, err := os.ReadFile(ruta)
		if err != nil {
			return fmt.Errorf("falta a copia xerada %s (%w); para rexenerala: %s", ruta, err, ComandoSincronizar)
		}
		if hashDe(copia) != hashDe(canonico) {
			return fmt.Errorf("a copia xerada %s non coincide coa fonte edicion/web/%s%s; "+
				"non se edita aquí: para rexenerala: %s",
				ruta, prefixo, nome, ComandoSincronizar)
		}
	}

	enDisco, err := ficheirosEnDisco(destino)
	if err != nil {
		return err
	}
	for _, nome := range enDisco {
		if !esperados[nome] {
			return fmt.Errorf("a copia xerada %s non existe na fonte edicion/web/%s%s: "+
				"ou se engade ao módulo, ou se retira; para rexenerar a copia: %s",
				filepath.Join(destino, filepath.FromSlash(nome)), prefixo, nome, ComandoSincronizar)
		}
	}
	return nil
}

// ficheirosEnDisco lista os ficheiros dunha copia xerada (recursivo, con "/"),
// excluíndo o MANIFEST.sha256 e os ficheiros ocultos.
func ficheirosEnDisco(destino string) ([]string, error) {
	var nomes []string
	err := filepath.WalkDir(destino, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(destino, ruta)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == NomeManifest {
			return nil
		}
		if base := path.Base(rel); strings.HasPrefix(base, ".") {
			return nil
		}
		nomes = append(nomes, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(nomes)
	return nomes, nil
}

// limparCartafolesBaleiros retira os cartafoles que quedaron sen ficheiros ao
// eliminar `nome` (só dentro da copia xerada; nunca o propio destino).
func limparCartafolesBaleiros(destino, nome string) {
	dir := path.Dir(nome)
	for dir != "." && dir != "/" {
		ruta := filepath.Join(destino, filepath.FromSlash(dir))
		entradas, err := os.ReadDir(ruta)
		if err != nil || len(entradas) > 0 {
			return
		}
		if err := os.Remove(ruta); err != nil {
			return
		}
		dir = path.Dir(dir)
	}
}
