package main

// Buxo non garda a clave da IA: pídelle os turnos de conversa á API da
// aplicación que o lanzou (Lua; máis adiante Yang). O bucle do axente
// (turno → ferramentas → turno...) vive aquí; as ferramentas son as boveda_*
// da propia API, que traballan sobre a bóveda rexistrada con ese id.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"edicion/api"
)

// Tipos do protocolo (mesmos nomes JSON que os de Lua, ia_conversa.go).
type IAChamada struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	Args string `json:"args"`
}

type IAResultado struct {
	ChamadaID string   `json:"chamadaId"`
	Nome      string   `json:"nome"`
	Contido   string   `json:"contido"`
	Erro      bool     `json:"erro"`
	Imaxes    []string `json:"imaxes"`
}

type IAMensaxe struct {
	Rol        string        `json:"rol"`
	Texto      string        `json:"texto"`
	Chamadas   []IAChamada   `json:"chamadas"`
	Resultados []IAResultado `json:"resultados"`
	Bruto      string        `json:"bruto"`
	Provedor   string        `json:"provedor"`
	Adxuntos   []string      `json:"adxuntos"`
	// Imaxes: data URI que a IA ve (as páxinas dun PDF; ver vision.go).
	Imaxes []string `json:"imaxes,omitempty"`
}

type IAFerramenta struct {
	Nome       string `json:"nome"`
	Descricion string `json:"descricion"`
	Parametros string `json:"parametros"`
	Tipo       string `json:"tipo"`
	Onde       string `json:"onde"`
}

type IATurno struct {
	Texto    string      `json:"texto"`
	Chamadas []IAChamada `json:"chamadas"`
	Bruto    string      `json:"bruto"`
	Provedor string      `json:"provedor"`
	Truncado bool        `json:"truncado"`
}

// EstadoIA é o que di a API sobre a IA configurada.
type EstadoIA struct {
	Configurada bool   `json:"configurada"`
	Modelo      string `json:"modelo"`
	Erro        string `json:"erro,omitempty"`
}

// ClienteIA fala coa API da aplicación anfitrioa.
type ClienteIA struct {
	C       *api.Cliente
	Boveda  string // id da bóveda na API
	ferrams []IAFerramenta
}

// IADeEntorno constrúe o cliente coas variables PIZTU_API_URL/PIZTU_API_TOKEN e
// BUXO_BOVEDA_ID. Devolve nil se non hai API (Buxo lanzado á man).
func IADeEntorno(idBoveda string) *ClienteIA {
	c, err := api.ClienteDeEntorno()
	if err != nil {
		return nil
	}
	if idBoveda == "" {
		idBoveda = "buxo"
	}
	return &ClienteIA{C: c, Boveda: idBoveda}
}

func (ia *ClienteIA) Estado(ctx context.Context) EstadoIA {
	var e EstadoIA
	if err := ia.C.Obter(ctx, "/ia/estado", nil, &e); err != nil {
		return EstadoIA{Erro: err.Error()}
	}
	return e
}

// timeoutTurnoIA é canto se agarda por un turno. Ten que superar o timeout da
// IA en Lua (240 s por defecto, configurable en Opcións): co minuto por defecto
// de api.Cliente os turnos con fontes longas cortábanse antes de que Lua
// respondese. Cancelar a construción segue cortando ao momento (ctx).
var timeoutTurnoIA = 30 * time.Minute

func (ia *ClienteIA) Turno(ctx context.Context, sistema string, msgs []IAMensaxe, ferrams []IAFerramenta) (IATurno, error) {
	var t IATurno
	c := *ia.C
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: timeoutTurnoIA}
	}
	err := c.Facer(ctx, "POST", "/ia/turno", nil, map[string]any{
		"sistema": sistema, "mensaxes": msgs, "ferramentas": ferrams,
	}, &t)
	return t, err
}

// Ferramentas devolve as ferramentas boveda_* da API (cacheadas).
func (ia *ClienteIA) Ferramentas(ctx context.Context) ([]IAFerramenta, error) {
	if ia.ferrams != nil {
		return ia.ferrams, nil
	}
	var f []IAFerramenta
	if err := ia.C.Obter(ctx, "/bovedas/ferramentas", nil, &f); err != nil {
		return nil, err
	}
	ia.ferrams = f
	return f, nil
}

// Executar chama a unha ferramenta boveda_* pola API e devolve o resultado
// serializado (o que se lle devolve á IA).
func (ia *ClienteIA) Executar(ctx context.Context, nome, args string) (string, error) {
	op, ok := strings.CutPrefix(nome, "boveda_")
	if !ok {
		return "", fmt.Errorf("ferramenta descoñecida: %s", nome)
	}
	var m map[string]any
	if strings.TrimSpace(args) != "" {
		if err := json.Unmarshal([]byte(args), &m); err != nil {
			return "", fmt.Errorf("argumentos non válidos: %w", err)
		}
	}
	if m == nil {
		m = map[string]any{}
	}
	ruta := "/bovedas/" + url.PathEscape(ia.Boveda) + "/" + op
	var res any
	var err error
	if op == "escribir" {
		err = ia.C.Facer(ctx, "PUT", ruta, nil, m, &res)
	} else {
		q := url.Values{}
		for k, v := range m {
			if s, ok := v.(string); ok {
				q.Set(k, s)
			} else {
				q.Set(k, fmt.Sprint(v))
			}
		}
		err = ia.C.Obter(ctx, ruta, q, &res)
	}
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

const maxResultadoFerramenta = 60000

// Axente executa un bucle de IA: pide un turno, executa as ferramentas que
// pide, devólvelle os resultados... ata que responde sen ferramentas ou se
// esgotan maxPasos. aviso recibe unha liña por cada ferramenta chamada.
// Devolve o texto final.
func (ia *ClienteIA) Axente(ctx context.Context, sistema, pedido string, maxPasos int, aviso func(string), aoPaso func()) (string, error) {
	ferrams, err := ia.Ferramentas(ctx)
	if err != nil {
		return "", fmt.Errorf("non se puideron obter as ferramentas da bóveda: %w", err)
	}
	msgs := []IAMensaxe{{Rol: "user", Texto: pedido}}
	for paso := 0; paso < maxPasos; paso++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		t, err := ia.turnoComPulso(ctx, sistema, msgs, ferrams, aviso)
		if err != nil {
			return "", err
		}
		if len(t.Chamadas) == 0 {
			if t.Truncado {
				return t.Texto, fmt.Errorf("a resposta da IA cortouse polo límite de tokens (auméntao en Opcións)")
			}
			return t.Texto, nil
		}
		msgs = append(msgs, IAMensaxe{Rol: "assistant", Texto: t.Texto, Chamadas: t.Chamadas, Bruto: t.Bruto, Provedor: t.Provedor})
		var resultados []IAResultado
		for _, ch := range t.Chamadas {
			if aviso != nil {
				aviso(resumoChamada(ch))
			}
			if aoPaso != nil {
				aoPaso()
			}
			cont, err := ia.Executar(ctx, ch.Nome, ch.Args)
			r := IAResultado{ChamadaID: ch.ID, Nome: ch.Nome, Contido: cont}
			if err != nil {
				r.Contido, r.Erro = err.Error(), true
			} else if len(r.Contido) > maxResultadoFerramenta {
				r.Contido = r.Contido[:maxResultadoFerramenta] + "\n[... recortado ...]"
			}
			resultados = append(resultados, r)
		}
		msgs = append(msgs, IAMensaxe{Rol: "tool", Resultados: resultados})
	}
	return "", fmt.Errorf("a IA non rematou en %d pasos", maxPasos)
}

func resumoChamada(c IAChamada) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(c.Args), &m)
	for _, k := range []string{"ruta", "consulta", "carpeta"} {
		if v, ok := m[k].(string); ok && v != "" {
			return c.Nome + " " + v
		}
	}
	return c.Nome
}

// pulsoIA é cada canto se avisa de que a IA segue traballando (un turno cunha
// fonte longa pode tardar minutos e sen aviso parece que Buxo se colgou).
var pulsoIA = 10 * time.Second

// turnoComPulso pide un turno e, mentres agarda, avisa cada pulsoIA.
func (ia *ClienteIA) turnoComPulso(ctx context.Context, sistema string, msgs []IAMensaxe, ferrams []IAFerramenta, aviso func(string)) (IATurno, error) {
	if aviso == nil {
		return ia.Turno(ctx, sistema, msgs, ferrams)
	}
	aviso("agardando pola IA…")
	feito := make(chan struct{})
	defer close(feito)
	inicio, cada := time.Now(), pulsoIA
	go func() {
		tk := time.NewTicker(cada)
		defer tk.Stop()
		for {
			select {
			case <-feito:
				return
			case <-tk.C:
				aviso(fmt.Sprintf("a IA segue traballando… (%d s)", int(time.Since(inicio).Seconds())))
			}
		}
	}()
	return ia.Turno(ctx, sistema, msgs, ferrams)
}
