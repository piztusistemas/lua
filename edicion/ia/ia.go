// Package ia é o cliente de IA común (vén de ia_conversa.go de Lua): un
// turno de conversa con texto, imaxes e chamadas a ferramentas, nos tres
// formatos de fío (Gemini, Anthropic e «compatible con OpenAI»), máis o
// bucle dun axente que executa ferramentas no backend.
package ia

import (
	"context"
	"edicion/idioma"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Provedores.
const (
	ProvedorGemini    = "gemini"
	ProvedorAnthropic = "anthropic"
	ProvedorOpenAI    = "openai" // tamén calquera compatible con OpenAI
)

var clientePorDefecto = &http.Client{Timeout: 240 * time.Second}

// Imaxe para mandarlle á IA.
type Imaxe struct {
	Mime string
	B64  string
}

// ImaxeDeBytes: atallo para unha imaxe en memoria.
func ImaxeDeBytes(mime string, datos []byte) Imaxe {
	return Imaxe{Mime: mime, B64: base64.StdEncoding.EncodeToString(datos)}
}

// Config: a configuración de IA do usuario.
type Config struct {
	Provedor  string `json:"provedor"`
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Modelo    string `json:"modelo"`
	MaxTokens int    `json:"maxTokens"`
	// TimeoutSegundos: <30 = 240 s.
	TimeoutSegundos int `json:"timeoutSegundos"`
}

// Configurada di se hai o mínimo para chamar á IA.
func (c Config) Configurada() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.Modelo) != ""
}

// BaseURLPorDefecto: o enderezo oficial de cada provedor.
func BaseURLPorDefecto(provedor string) string {
	switch provedor {
	case ProvedorGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ProvedorAnthropic:
		return "https://api.anthropic.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func (c Config) resolver() (string, string, *http.Client, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", "", nil, idioma.Errorf("go.ia.sen_clave", "falta a clave da API de IA (configúraa en Opcións)")
	}
	if strings.TrimSpace(c.Modelo) == "" {
		return "", "", nil, idioma.Errorf("go.ia.sen_modelo", "falta o modelo de IA (configúrao en Opcións)")
	}
	p := c.Provedor
	if p == "" {
		p = ProvedorGemini
	}
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		base = BaseURLPorDefecto(p)
	}
	t := time.Duration(c.TimeoutSegundos) * time.Second
	if c.TimeoutSegundos < 30 {
		t = 240 * time.Second
	}
	return p, base, &http.Client{Timeout: t}, nil
}

// Turno pide á IA un turno da conversa.
func Turno(c Config, sistema string, msgs []IAMensaxe, ferramentas []IAFerramenta) (IATurno, error) {
	p, base, cliente, err := c.resolver()
	if err != nil {
		return IATurno{}, err
	}
	var t IATurno
	switch p {
	case ProvedorAnthropic:
		max := c.MaxTokens
		if max <= 0 {
			max = 16000
		}
		t, err = turnoAnthropic(base, c.APIKey, c.Modelo, sistema, msgs, ferramentas, max, cliente)
	case ProvedorGemini:
		t, err = turnoGemini(base, c.APIKey, c.Modelo, sistema, msgs, ferramentas, c.MaxTokens, cliente)
	default:
		t, err = turnoOpenAI(base, c.APIKey, c.Modelo, sistema, msgs, ferramentas, c.MaxTokens, cliente)
	}
	t.Provedor = p
	return t, err
}

// Preguntar: unha soa pregunta (con imaxes opcionais), sen ferramentas.
func Preguntar(c Config, sistema, texto string, imaxes ...Imaxe) (string, error) {
	t, err := Turno(c, sistema, []IAMensaxe{{Rol: "user", Texto: texto, ImaxesIA: imaxes}}, nil)
	if err != nil {
		return "", err
	}
	if t.Truncado {
		return t.Texto, fmt.Errorf("a resposta da IA cortouse polo límite de tokens")
	}
	return t.Texto, nil
}

// Ferramenta dun axente do backend.
type Ferramenta struct {
	Def      IAFerramenta
	Executar func(args json.RawMessage) (texto string, imaxes []Imaxe, err error)
}

// Axente: bucle pedir -> executar ferramentas -> volver pedir, ata que a IA
// responde sen chamadas ou se esgotan os pasos. Devolve o texto final.
func Axente(ctx context.Context, c Config, sistema, pedido string, ferramentas []Ferramenta, maxPasos int, aviso func(string)) (string, error) {
	defs := make([]IAFerramenta, len(ferramentas))
	porNome := map[string]Ferramenta{}
	for i, f := range ferramentas {
		defs[i] = f.Def
		porNome[f.Def.Nome] = f
	}
	msgs := []IAMensaxe{{Rol: "user", Texto: pedido}}
	for paso := 0; paso < maxPasos; paso++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		t, err := Turno(c, sistema, msgs, defs)
		if err != nil {
			return "", err
		}
		msgs = append(msgs, IAMensaxe{Rol: "assistant", Texto: t.Texto, Chamadas: t.Chamadas, Bruto: t.Bruto, Provedor: t.Provedor})
		if len(t.Chamadas) == 0 {
			return t.Texto, nil
		}
		var res []IAResultado
		for _, ch := range t.Chamadas {
			f, ok := porNome[ch.Nome]
			r := IAResultado{ChamadaID: ch.ID, Nome: ch.Nome}
			if !ok {
				r.Contido, r.Erro = "ferramenta descoñecida: "+ch.Nome, true
			} else {
				if aviso != nil {
					aviso(ch.Nome)
				}
				texto, imaxes, err := f.Executar(json.RawMessage(argsOuBaleiro(ch.Args)))
				if err != nil {
					r.Contido, r.Erro = err.Error(), true
				} else {
					r.Contido, r.ImaxesIA = texto, imaxes
				}
			}
			res = append(res, r)
		}
		msgs = append(msgs, IAMensaxe{Rol: "tool", Resultados: res})
	}
	return "", fmt.Errorf("o axente non rematou en %d pasos", maxPasos)
}

// ExtraerJSON: o primeiro obxecto ou lista JSON dunha resposta (a IA ás
// veces envólveo en ```json … ``` ou lle pon texto arredor).
func ExtraerJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		resto := s[i+3:]
		if nl := strings.Index(resto, "\n"); nl >= 0 {
			resto = resto[nl+1:]
		}
		if j := strings.LastIndex(resto, "```"); j >= 0 {
			return strings.TrimSpace(resto[:j])
		}
	}
	ini := strings.IndexAny(s, "{[")
	if ini < 0 {
		return s
	}
	pecha := map[byte]byte{'{': '}', '[': ']'}[s[ini]]
	if fin := strings.LastIndexByte(s, pecha); fin > ini {
		return s[ini : fin+1]
	}
	return s[ini:]
}
