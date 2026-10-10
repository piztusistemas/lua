package main

// Ferramentas do asistente para os ADXUNTOS (adxuntos.go) e para o deseño das
// plantillas con imaxes: ler documentos, ver imaxes e páxinas, sacar as imaxes
// dun documento, levar unha imaxe ao cartafol dunha plantilla, e ver como
// queda unha plantilla (imaxe da páxina renderizada).

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxImaxesPorPeticion = 12   // imaxes en toda a conversa que se lle mandan á IA
	maxTextoInline       = 8000 // caracteres dun documento que van directamente na mensaxe
)

// resultadoComImaxes é o que devolve unha ferramenta que quere amosarlle
// imaxes á IA (ids de adxuntos).
type resultadoComImaxes struct {
	Texto  string
	Imaxes []string
}

// resolverAdxuntosMensaxes prepara a conversa para o provedor: as imaxes que
// o usuario adxuntou (e as que devolveron as ferramentas) pasan a imaxes
// reais, e cada mensaxe co seus adxuntos leva unha nota cos ids e, dos
// documentos curtos, o seu texto. Non modifica o historial orixinal.
func resolverAdxuntosMensaxes(msgs []IAMensaxe) []IAMensaxe {
	out := make([]IAMensaxe, len(msgs))
	copy(out, msgs)
	restantes := maxImaxesPorPeticion
	// Das máis novas ás máis vellas: se hai demasiadas imaxes, quedan as recentes.
	for i := len(out) - 1; i >= 0; i-- {
		m := out[i]
		switch m.Rol {
		case "user":
			if len(m.Adxuntos) == 0 {
				continue
			}
			nota, imaxes := describirAdxuntosDaMensaxe(m.Adxuntos, &restantes)
			m.Texto = strings.TrimSpace(m.Texto + "\n\n" + nota)
			m.imaxes = imaxes
			out[i] = m
		case "tool":
			rs := make([]IAResultado, len(m.Resultados))
			copy(rs, m.Resultados)
			for j, r := range rs {
				if len(r.Imaxes) == 0 {
					continue
				}
				for _, id := range r.Imaxes {
					if restantes <= 0 {
						rs[j].Contido += fmt.Sprintf("\n[imaxe %s non incluída: hai demasiadas imaxes na conversa; volve velaa con adxunto_ver se a necesitas]", id)
						continue
					}
					im, _, err := imaxeDoAdxuntoParaIA(id)
					if err != nil {
						rs[j].Contido += fmt.Sprintf("\n[imaxe %s non dispoñible: %v]", id, err)
						continue
					}
					restantes--
					rs[j].imaxesIA = append(rs[j].imaxesIA, im)
				}
			}
			m.Resultados = rs
			out[i] = m
		}
	}
	return out
}

func describirAdxuntosDaMensaxe(ids []string, restantes *int) (string, []imaxeIA) {
	var b strings.Builder
	var imaxes []imaxeIA
	b.WriteString("[Adxuntos desta mensaxe: ficheiros que o usuario xuntou; os ids valen para as ferramentas adxunto_*]\n")
	for _, id := range ids {
		ax, datos, err := adxuntoPorID(id)
		if err != nil {
			fmt.Fprintf(&b, "- %s — non dispoñible (%v)\n", id, err)
			continue
		}
		fmt.Fprintf(&b, "- %s\n", descricionAdxunto(ax))
		if ax.Tipo == tipoAdxImaxe {
			if *restantes <= 0 {
				b.WriteString("  (imaxe non incluída: hai demasiadas imaxes na conversa; velaa con adxunto_ver)\n")
				continue
			}
			im, _, err := imaxeDoAdxuntoParaIA(id)
			if err != nil {
				fmt.Fprintf(&b, "  (non se puido preparar a imaxe: %v)\n", err)
				continue
			}
			*restantes--
			imaxes = append(imaxes, im)
			b.WriteString("  (a imaxe vai incluída nesta mensaxe: podes velaa)\n")
			continue
		}
		t, err := textoDoAdxunto(ax, datos)
		switch {
		case err != nil:
			fmt.Fprintf(&b, "  (non se puido ler o texto: %v)\n", err)
		case len([]rune(t)) <= maxTextoInline:
			fmt.Fprintf(&b, "  Contido de «%s»:\n<<<\n%s\n>>>\n", ax.Nome, t)
		default:
			fmt.Fprintf(&b, "  (documento longo, %d caracteres: le por partes con adxunto_ler)\n", len([]rune(t)))
		}
	}
	return strings.TrimSpace(b.String()), imaxes
}

// engadirImaxeTemporal mete unha imaxe xerada polo programa (páxina renderizada,
// vista previa) no almacén, para poder amosársela á IA.
func engadirImaxeTemporal(nome string, datos []byte, orixe string) (Adxunto, error) {
	return gardarAdxunto(nome, datos, orixe)
}

var nomeFicheiroRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// nomeSeguroImaxe deixa un nome de ficheiro sen espazos nin símbolos: LaTeX
// e as ligazóns de Markdown levan mal os espazos e os guións baixos raros.
func nomeSeguroImaxe(nome, extensionDeseada string) string {
	base := filepath.Base(strings.ReplaceAll(strings.TrimSpace(nome), "\\", "/"))
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.Trim(nomeFicheiroRe.ReplaceAllString(base, "-"), "-.")
	if base == "" {
		base = "imaxe"
	}
	return base + "." + extensionDeseada
}

func init() {
	ferramentasBackend = append(ferramentasBackend,
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxuntos_listar", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Lista os ficheiros que o usuario xuntou nesta sesión (id, nome, tipo, tamaño, páxinas). Os adxuntos son temporais: non forman parte do curso ata que os levas a unha plantilla ou ao curso.",
			},
			exec: func(a *App, _ map[string]any) (any, error) {
				l := a.AdxuntosListar()
				return map[string]any{"adxuntos": l, "total": len(l)}, nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxunto_ler", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Le o TEXTO dun adxunto: PDF, Word (.docx), OpenDocument (.odt/.ods/.odp), presentacións, follas de cálculo, RTF, texto, HTML... (as imaxes non teñen texto: usa adxunto_ver). Para documentos longos devolve unha parte; usa 'inicio' para seguir.",
				Parametros: esqObx(map[string]any{
					"id":            esqTxt("Id do adxunto (de adxuntos_listar)."),
					"inicio":        esqNum("Carácter desde o que ler (por defecto 0)."),
					"maxCaracteres": esqNum("Cantos caracteres devolver (por defecto 12000, máximo 40000)."),
				}, "id"),
			},
			exec: func(a *App, args map[string]any) (any, error) {
				ax, datos, err := adxuntoPorID(argTxt(args, "id"))
				if err != nil {
					return nil, err
				}
				t, err := textoDoAdxunto(ax, datos)
				if err != nil {
					return nil, err
				}
				if strings.TrimSpace(t) == "" {
					return nil, fmt.Errorf("«%s» non ten texto (se son imaxes, velas con adxunto_ver ou adxunto_extraer_imaxes)", ax.Nome)
				}
				return paxinarTexto(t, argInt(args, "inicio"), argInt(args, "maxCaracteres")), nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxunto_ver", Tipo: tipoLectura, Onde: "backend",
				Descricion: "AMOSA á IA unha imaxe adxunta, ou unha PÁXINA renderizada dun PDF (ou dun documento Word/OpenDocument, se hai LibreOffice). Serve para ler texto en imaxes, ver un deseño, un logotipo ou un modelo de documento. Precisa un modelo con visión.",
				Parametros: esqObx(map[string]any{
					"id":     esqTxt("Id do adxunto."),
					"paxina": esqNum("Páxina (desde 1) se é un documento; por defecto a 1."),
				}, "id"),
			},
			exec: func(a *App, args map[string]any) (any, error) {
				ax, datos, err := adxuntoPorID(argTxt(args, "id"))
				if err != nil {
					return nil, err
				}
				if ax.Tipo == tipoAdxImaxe {
					return resultadoComImaxes{Texto: descricionAdxunto(ax), Imaxes: []string{ax.ID}}, nil
				}
				pax := max(argInt(args, "paxina"), 1)
				png, err := renderizarPaxina(ax, datos, pax)
				if err != nil {
					return nil, err
				}
				nome := fmt.Sprintf("%s-paxina-%d.png", strings.TrimSuffix(ax.Nome, filepath.Ext(ax.Nome)), pax)
				nova, err := engadirImaxeTemporal(nome, png, ax.ID)
				if err != nil {
					return nil, err
				}
				txt := fmt.Sprintf("Páxina %d de «%s»", pax, ax.Nome)
				if ax.Paxinas > 0 {
					txt += fmt.Sprintf(" (%d en total)", ax.Paxinas)
				}
				return resultadoComImaxes{Texto: txt + ". Gardouse coma adxunto " + nova.ID + ".", Imaxes: []string{nova.ID}}, nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxunto_extraer_imaxes", Tipo: tipoEscritura, Onde: "backend",
				Descricion: "Saca as imaxes incrustadas dun PDF ou dun documento Word/OpenDocument/presentación (o logotipo dunha cabeceira, fotos...) e gárdaas coma novos adxuntos temporais. Despois podes velas con adxunto_ver e levalas a unha plantilla.",
				Parametros: esqObx(map[string]any{
					"id":  esqTxt("Id do documento."),
					"max": esqNum("Máximo de imaxes (por defecto 12)."),
				}, "id"),
			},
			exec: func(a *App, args map[string]any) (any, error) {
				ax, datos, err := adxuntoPorID(argTxt(args, "id"))
				if err != nil {
					return nil, err
				}
				n := argInt(args, "max")
				if n <= 0 || n > 30 {
					n = 12
				}
				ims, err := imaxesDoAdxunto(ax, datos, n)
				if err != nil {
					return nil, err
				}
				if len(ims) == 0 {
					return "O documento non ten imaxes que extraer (ou só ten formas vectoriais/formatos non compatibles).", nil
				}
				var novas []Adxunto
				for _, im := range ims {
					nome := strings.TrimSuffix(ax.Nome, filepath.Ext(ax.Nome)) + "-" + im.Nome
					na, err := gardarAdxunto(nome, im.Datos, ax.ID)
					if err != nil {
						continue // unha imaxe rara non debe botar todo abaixo
					}
					novas = append(novas, na)
				}
				return map[string]any{"extraidas": novas, "total": len(novas)}, nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxunto_eliminar", Tipo: tipoEscritura, Onde: "backend",
				Descricion: "Quita un adxunto do almacén temporal da sesión.",
				Parametros: esqObx(map[string]any{"id": esqTxt("Id do adxunto.")}, "id"),
			},
			exec: func(a *App, args map[string]any) (any, error) {
				if err := a.AdxuntoEliminar(argTxt(args, "id")); err != nil {
					return nil, err
				}
				return "adxunto eliminado", nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "adxunto_gardar_en_plantilla", Tipo: tipoEscritura, Onde: "backend",
				Descricion: "GARDA DEFINITIVAMENTE un ou VARIOS adxuntos (logotipos...) no cartafol de imaxes dunha plantilla, nunha soa chamada: pasa 'ids' (lista) e, se queres, 'nomes' (mesma orde). Admite imaxes (PNG, JPEG, GIF, WebP, BMP, TIFF; tamén RAW, SVG, HEIC... se o equipo ten conversor: xa quedan en PNG) e PDF, que se garda COMO PDF (vectorial: o LaTeX e o PDF do informe usan o logotipo sen perder calidade; Word e OpenDocument usan unha imaxe de píxeles sacada del automaticamente). Se a plantilla é de exemplo, Lúa fai antes unha copia propia (e a copia herda ser a activa): o resultado di o id da plantilla onde quedaron. Un ficheiro que falle non impide gardar os demais: o resultado lista o que se gardou e os erros. Usos: LaTeX \\includegraphics{nome}; Word/ODT: @imaxe nome (na portada, cabeceira ou pé).",
				Parametros: esqObx(map[string]any{
					"plantilla":  esqTxt("Id da plantilla onde gardalos."),
					"ids":        map[string]any{"type": "array", "description": "Ids dos adxuntos a gardar (varios logotipos á vez).", "items": map[string]any{"type": "string"}},
					"id":         esqTxt("Id dun só adxunto (alternativa a 'ids')."),
					"nomes":      map[string]any{"type": "array", "description": "Nomes de ficheiro desexados, na mesma orde ca 'ids' (opcional). Sen espazos.", "items": map[string]any{"type": "string"}},
					"nome":       esqTxt("Nome desexado se só hai un (opcional), p.ex. logotipo.png."),
					"substituir": esqBool("Se xa existe un ficheiro con ese nome, substituílo (por defecto crea outro nome)."),
					"formato":    esqEnum("Só para PDF: 'orixinal' (por defecto: gardar o PDF) ou 'png' (gardalo coma imaxe de píxeles).", "orixinal", "png"),
					"paxina":     esqNum("Só para PDF: páxina a usar (por defecto a 1)."),
				}, "plantilla"),
			},
			exec: func(a *App, args map[string]any) (any, error) { return gardarAdxuntosEnPlantilla(a, args) },
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "plantilla_imaxe_eliminar", Tipo: tipoConfirmar, Onde: "backend",
				Descricion: "Elimina unha imaxe do cartafol dunha plantilla propia. Non se pode desfacer; o usuario ten que confirmar.",
				Parametros: esqObx(map[string]any{
					"plantilla": esqTxt("Id da plantilla."),
					"nome":      esqTxt("Nome da imaxe (de plantilla_ler, campo imaxes)."),
				}, "plantilla", "nome"),
			},
			exec: func(a *App, args map[string]any) (any, error) {
				p, err := plantillaPorID(a, argTxt(args, "plantilla"))
				if err != nil {
					return nil, err
				}
				if p.DeFabrica {
					return nil, fmt.Errorf("«%s» é unha plantilla de exemplo: non se lle poden quitar imaxes (duplícaa antes con plantilla_duplicar)", p.Nome)
				}
				if err := a.EliminarImaxePlantilla(p.ID, argTxt(args, "nome")); err != nil {
					return nil, err
				}
				return "imaxe eliminada", nil
			},
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "plantilla_previsualizar", Tipo: tipoLectura, Onde: "backend",
				Descricion: "VÉ como queda unha plantilla: xera un informe de exemplo e AMOSA á IA unha páxina (imaxe). Con formato pdf (por defecto) usa o LaTeX da plantilla; con docx ou odt usa o deseño de Word/OpenDocument (precisa LibreOffice). Pasa o id dunha plantilla (e opcionalmente campos para probar cambios sen gardar). Úsaa para axustar a estética: mira, corrixe e volve mirar.",
				Parametros: esqPlantilla(map[string]any{
					"id":        esqTxt("Id da plantilla base (opcional)."),
					"formato":   esqEnum("Formato a previsualizar.", "pdf", "docx", "odt"),
					"paxina":    esqNum("Páxina que quere ver (desde 1; a portada é a 1)."),
					"documento": esqEnum("Documento de exemplo: 'cronograma' (por defecto) ou 'actas'.", "cronograma", "actas"),
				}),
			},
			exec: func(a *App, args map[string]any) (any, error) { return previsualizarPlantillaParaIA(a, args) },
		},
	)
}

// plantillaPropiaParaEscribir devolve a plantilla onde se poden gardar
// imaxes: se é de exemplo, Lua crea antes unha copia propia (que herda ser a
// activa e os documentos que a usaban, coma ao editala) e devolve a nota.
func plantillaPropiaParaEscribir(a *App, p PlantillaLatex) (PlantillaLatex, string, error) {
	if !p.DeFabrica {
		return p, "", nil
	}
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return p, "", err
	}
	alm, errCopia := depositoPlantillas.BifurcarEnSitio(alm, p, time.Now().Format(time.RFC3339))
	if err := saveAlmacenPlantillas(alm); err != nil {
		return p, "", err
	}
	if errCopia != nil {
		return p, "", errCopia
	}
	nova := alm.Plantillas[len(alm.Plantillas)-1]
	return nova, fmt.Sprintf("«%s» é unha plantilla de exemplo e non se pisa: creouse unha copia propia («%s», id %s) e as imaxes gardáronse nela; a copia pasou a ser a plantilla activa se a orixinal o era", p.Nome, nova.Nome, nova.ID), nil
}

func argLista(args map[string]any, k string) []string {
	var out []string
	if v, ok := args[k].([]any); ok {
		for _, x := range v {
			if s := strings.TrimSpace(fmt.Sprint(x)); s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
	}
	return out
}

type imaxeGardadaEnPlantilla struct {
	Adxunto      string `json:"adxunto"`
	Nome         string `json:"nome"`
	Tipo         string `json:"tipo"`
	UsoLatex     string `json:"uso_latex"`
	UsoOfimatica string `json:"uso_ofimatica"`
	UsoMarkdown  string `json:"uso_markdown"`
	Nota         string `json:"nota,omitempty"`
}

func gardarAdxuntosEnPlantilla(a *App, args map[string]any) (any, error) {
	p, err := plantillaPorID(a, argTxt(args, "plantilla"))
	if err != nil {
		return nil, err
	}
	ids := argLista(args, "ids")
	if id := strings.TrimSpace(argTxt(args, "id")); id != "" {
		ids = append([]string{id}, ids...)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("indica 'ids' (ou 'id') cos adxuntos a gardar (usa adxuntos_listar)")
	}
	nomes := argLista(args, "nomes")
	if n := strings.TrimSpace(argTxt(args, "nome")); n != "" && len(ids) == 1 && len(nomes) == 0 {
		nomes = []string{n}
	}
	propia, nota, err := plantillaPropiaParaEscribir(a, p)
	if err != nil {
		return nil, err
	}
	dir, err := plantillaImaxesDir(propia.ID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	substituir, _ := argBool(args["substituir"])
	formatoPDF := strings.ToLower(argTxt(args, "formato"))
	pax := max(argInt(args, "paxina"), 1)

	var gardadas []imaxeGardadaEnPlantilla
	var erros []map[string]string
	for i, id := range ids {
		nomeDeseado := ""
		if i < len(nomes) {
			nomeDeseado = nomes[i]
		}
		g, err := gardarUnAdxuntoNoCartafol(dir, id, nomeDeseado, substituir, formatoPDF, pax)
		if err != nil {
			erros = append(erros, map[string]string{"adxunto": id, "erro": err.Error()})
			continue
		}
		gardadas = append(gardadas, g)
	}
	if len(gardadas) == 0 {
		msgs := make([]string, len(erros))
		for i, e := range erros {
			msgs[i] = e["adxunto"] + ": " + e["erro"]
		}
		return nil, fmt.Errorf("non se gardou ningún ficheiro: %s", strings.Join(msgs, " | "))
	}
	res := map[string]any{"plantilla": propia.ID, "nome_plantilla": propia.Nome, "gardadas": gardadas}
	if nota != "" {
		res["nota"] = nota
	}
	if len(erros) > 0 {
		res["erros"] = erros
	}
	return res, nil
}

// gardarUnAdxuntoNoCartafol garda un adxunto (imaxe ou PDF) no cartafol de
// imaxes dunha plantilla e devolve como usalo.
func gardarUnAdxuntoNoCartafol(dir, id, nomeDeseado string, substituir bool, formatoPDF string, pax int) (imaxeGardadaEnPlantilla, error) {
	ax, datos, err := adxuntoPorID(id)
	if err != nil {
		return imaxeGardadaEnPlantilla{}, err
	}
	var out []byte
	var ext, nota string
	switch ax.Tipo {
	case tipoAdxPDF:
		if formatoPDF == "png" {
			if out, err = pdfAPNGLogo(datos, pax, 300); err != nil {
				return imaxeGardadaEnPlantilla{}, err
			}
			ext = "png"
		} else {
			// O PDF vai tal cal: o LaTeX úsao coma vectorial.
			out, ext = datos, "pdf"
			nota = "PDF vectorial: o informe en PDF úsao sen perder calidade; en Word/OpenDocument usarase unha imaxe de píxeles sacada da páxina 1"
		}
	case tipoAdxImaxe:
		// PNG e JPEG gárdanse tal cales (se non son enormes); o resto pasa a PNG.
		out, ext = datos, "png"
		switch ax.Mime {
		case "image/jpeg":
			ext = "jpg"
		case "image/png":
		default:
			out = nil
		}
		if out == nil || max(ax.Ancho, ax.Alto) > 3000 {
			fmtSaida := "png"
			if ax.Mime == "image/jpeg" {
				fmtSaida = "jpeg"
			}
			var mime string
			if out, mime, _, _, err = redimensionarImaxe(datos, 3000, fmtSaida); err != nil {
				return imaxeGardadaEnPlantilla{}, err
			}
			ext = "png"
			if mime == "image/jpeg" {
				ext = "jpg"
			}
		}
		if ax.Formato != "" {
			nota = "convertida a PNG desde " + strings.ToUpper(ax.Formato)
		}
	default:
		return imaxeGardadaEnPlantilla{}, fmt.Errorf("«%s» non é unha imaxe nin un PDF (é %s): para un documento de Word/OpenDocument usa adxunto_extraer_imaxes e garda as imaxes que saian", ax.Nome, ax.Tipo)
	}
	if strings.TrimSpace(nomeDeseado) == "" {
		nomeDeseado = ax.Nome
	}
	nome := nomeSeguroImaxe(nomeDeseado, ext)
	if !substituir {
		nome = dedupeFileName(dir, nome, "imaxe")
	}
	if err := os.WriteFile(filepath.Join(dir, nome), out, 0o644); err != nil {
		return imaxeGardadaEnPlantilla{}, fmt.Errorf("non se puido escribir no cartafol da plantilla: %w", err)
	}
	return imaxeGardadaEnPlantilla{
		Adxunto: ax.ID, Nome: nome, Tipo: ext,
		UsoLatex:     `\includegraphics[height=20mm]{` + nome + `}`,
		UsoOfimatica: "@imaxe " + nome + " 12   (na portada, cabeceira ou pé de Word/OpenDocument; o número é o alto en mm)",
		UsoMarkdown:  "![](images/" + nome + ")",
		Nota:         nota,
	}, nil
}

func previsualizarPlantillaParaIA(a *App, args map[string]any) (any, error) {
	p, err := plantillaDeArgs(a, args)
	if err != nil {
		return nil, err
	}
	pax := max(argInt(args, "paxina"), 1)
	formato := strings.ToLower(argTxt(args, "formato"))
	documento := documentoDeMostra(args)
	p.Textos = normalizarTextosDoc(p.Textos)
	var png []byte
	var total int
	switch formato {
	case "", "pdf":
		r, err := a.PrevisualizarPlantillaDe(p, documento)
		if err != nil {
			return nil, fmt.Errorf("non compila: %s", finalDe(err.Error(), 3000))
		}
		total = len(r.PageImages)
		if total == 0 {
			return nil, fmt.Errorf("compilou, pero non se puido debuxar a vista previa (falta pdftoppm)")
		}
		if pax > total {
			pax = total
		}
		uri := r.PageImages[pax-1]
		i := strings.Index(uri, ",")
		if i < 0 {
			return nil, fmt.Errorf("vista previa non válida")
		}
		if png, err = base64.StdEncoding.DecodeString(uri[i+1:]); err != nil {
			return nil, err
		}
	case "docx", "odt":
		ofi, err := normalizarOfimatica(p.Ofimatica)
		if err != nil {
			return nil, err
		}
		p.Ofimatica = ofi
		datos, _, err := datosOfimaticaDe(pedidoDemo(documento), &p, formato)
		if err != nil {
			return nil, err
		}
		tmp := Adxunto{ID: "previa", Nome: "previa." + formato, Tipo: tipoAdxDocx}
		if formato == "odt" {
			tmp.Tipo = tipoAdxOdt
		}
		if png, err = renderizarPaxina(tmp, datos, pax); err != nil {
			return nil, err
		}
		total = -1
	default:
		return nil, fmt.Errorf("formato non válido: usa pdf, docx ou odt")
	}
	if formato == "" {
		formato = "pdf"
	}
	nome := fmt.Sprintf("previsualizacion-%s-paxina-%d.png", formato, pax)
	nova, err := engadirImaxeTemporal(nome, png, "plantilla_previsualizar")
	if err != nil {
		return nil, err
	}
	txt := fmt.Sprintf("Páxina %d da vista previa de «%s» (%s).", pax, p.Nome, formato)
	if total > 0 {
		txt += fmt.Sprintf(" O informe de exemplo ten %d páxinas.", total)
	}
	return resultadoComImaxes{Texto: txt, Imaxes: []string{nova.ID}}, nil
}

// paxinarTexto devolve un anaco de `texto` (desde o carácter inicio, máximo
// maxCaracteres) e avisa de como seguir se queda máis.
func paxinarTexto(texto string, inicio, maxCaracteres int) string {
	if maxCaracteres <= 0 {
		maxCaracteres = webCaracteresPorDef
	}
	if maxCaracteres > webCaracteresMax {
		maxCaracteres = webCaracteresMax
	}
	runas := []rune(texto)
	total := len(runas)
	if inicio < 0 {
		inicio = 0
	}
	if inicio >= total {
		return fmt.Sprintf("[o texto ten %d caracteres; o inicio pedido (%d) está fóra]", total, inicio)
	}
	fin := min(inicio+maxCaracteres, total)
	saida := string(runas[inicio:fin])
	if fin < total {
		saida += fmt.Sprintf("\n\n[... continúa: amosáronse os caracteres %d-%d de %d. Para seguir, chama de novo con inicio=%d ...]", inicio, fin, total, fin)
	}
	return saida
}

// documentoDeMostra é o documento sobre o que se proba unha plantilla: o do
// argumento «documento» ("actas") ou, se non, o cronograma.
func documentoDeMostra(args map[string]any) string {
	if strings.EqualFold(strings.TrimSpace(argTxt(args, "documento")), ferramentaActas) {
		return ferramentaActas
	}
	return ferramentaCronograma
}
