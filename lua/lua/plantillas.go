package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Biblioteca de PLANTILLAS ("estilos"): o que permite que o mesmo informe
// sirva para centros distintos. Unha plantilla decide COMO se maqueta o
// informe (cabeceira co logotipo e o nome do centro, pé co nome de quen
// orienta e os teléfonos, marxes, tipografía...); o curso en edición segue
// decidindo QUE contén. Aplícase á vista previa (que xa é o PDF real) e a
// todas as exportacións: PDF, .tex, .md, .docx e .odt. Pódense exportar a un
// ficheiro e importar noutro equipo (plantillas_intercambio.go), así un centro
// prepara a súa maqueta unha vez e compárteo co resto do departamento.
//
// Formato dunha plantilla: LaTeX normal e corrente cun marcador {{CORPO}}
// no sitio onde vai o informe, máis {{VARIABLES}} en maiúsculas que se
// enchen unha vez (CENTRO, TELEFONO, ORIENTADOR...). Hai dúas formas válidas,
// distinguidas SÓ por levar \documentclass ou non:
//
//   - Plantilla COMPLETA (leva \documentclass): manda ela de todo. Lua só
//     lle inxecta, xusto antes do \begin{document}, os \usepackage e as
//     definicións que lle falten ao informe (cores, etiquetas, formato dos
//     títulos...) - ver inxectarPreambuloObrigatorio.
//   - Plantilla FRAGMENTO (sen \documentclass): Lua envólvea no preámbulo
//     do informe (preambuloInformeCrono, con fancyhdr cargado para poder
//     facer cabeceiras e pés). Abonda para o caso máis común - engadir unha
//     cabeceira e un pé ao informe - sen ter que saber nada de preámbulos.
//
// As imaxes (logotipos) non se meten na plantilla en base64: gárdanse en
// <config>/lua/plantillas/<id>/ e cópianse a carón do documento no intre
// de compilar (ver copiarImaxesPlantilla), así que na plantilla
// referéncianse polo seu nome de ficheiro a secas.

// Os tipos da biblioteca (PlantillaVar, PlantillaLatex, almacenPlantillas,
// PlantillasEstado) e mais os marcadores reservados viven no módulo común
// edicion/plantilla e están aliasados en edicion_plantilla.go, para que o
// resto do proxecto (e os bindings de Wails) sigan usando os mesmos nomes.

// varRe e usepackageRe viven en edicion/plantilla (composicion.go).

func plantillasPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "lua")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "plantillas.json"), nil
}

// plantillaImaxesDir é o cartafol de imaxes DUNHA plantilla. Un cartafol por
// plantilla (non un común) para que duplicar/borrar unha plantilla sexa
// copiar/borrar un cartafol, sen levar contas de que logo usa quen.
func plantillaImaxesDir(id string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lua", "plantillas", id), nil
}

// fabricaObsoleta son as plantillas de exemplo das versións de Lua que aínda
// eran un xerador de exames ("Exame de instituto", "Boletín de exercicios",
// "Orzamento"): xa non se sementan, porque non teñen sentido para un informe.
// A clave é o ID de fábrica; o valor, o SHA-256 do LaTeX orixinal, que é o
// que permite saber se alguén as tocou.
var fabricaObsoleta = map[string]string{
	"fabrica-exame":     "a0b0ce766ab7cd3c",
	"fabrica-boletin":   "9cf5ae060a50a0c8",
	"fabrica-orzamento": "eb87dceecc86d751",
}

// A maquinaria da biblioteca (ler/gardar o almacén, sementar a fábrica,
// reparar as de fábrica editadas, bifurcar, podar e copiar imaxes) vive en
// edicion/plantilla; as funcións de abaixo son os adaptadores da aplicación.

// ---------- API para o frontend ----------

// PlantillasEstado está aliasado en edicion_plantilla.go (tipo do módulo).

// ListarPlantillas devolve a biblioteca ordenada por nome, con `imaxes` xa
// resolto, e a plantilla activa.
func (a *App) ListarPlantillas() (PlantillasEstado, error) {
	return depositoPlantillas.Estado()
}

// GardarPlantilla crea (ID baleira) ou actualiza unha plantilla. Valida o
// mínimo imprescindible aquí, no backend, porque a mesma plantilla pode
// chegar do editor manual, do asistente de IA ou dun plantillas.json
// editado á man.
func (a *App) GardarPlantilla(p PlantillaLatex) (PlantillaLatex, error) {
	p.Nome = strings.TrimSpace(p.Nome)
	if p.Nome == "" {
		return PlantillaLatex{}, fmt.Errorf("dálle un nome á plantilla")
	}
	if strings.TrimSpace(p.Latex) == "" {
		return PlantillaLatex{}, fmt.Errorf("a plantilla non ten código LaTeX")
	}
	if !strings.Contains(p.Latex, marcadorCorpo) {
		return PlantillaLatex{}, fmt.Errorf("o LaTeX da plantilla ten que levar %s no sitio onde vai o documento", marcadorCorpo)
	}
	if strings.TrimSpace(p.Markdown) != "" && !strings.Contains(p.Markdown, marcadorCorpo) {
		return PlantillaLatex{}, fmt.Errorf("o Markdown da plantilla ten que levar %s no sitio onde vai o documento", marcadorCorpo)
	}
	p.Variables = normalizarVariables(p.Variables)
	p.Textos = normalizarTextosDoc(p.Textos)
	ofi, err := normalizarOfimatica(p.Ofimatica)
	if err != nil {
		return PlantillaLatex{}, err
	}
	p.Ofimatica = ofi

	// A partir de aquí (crear, actualizar, bifurcar se é de fábrica, gardar e
	// resolver as imaxes) é maquinaria común: ver edicion/plantilla.Deposito.
	return depositoPlantillas.Meter(p)
}

// EliminarPlantilla bórraa xunto co seu cartafol de imaxes, esquecendo tamén
// cal era a activa e que documentos a usaban.
func (a *App) EliminarPlantilla(id string) error {
	return depositoPlantillas.Eliminar(id)
}

// DuplicarPlantilla fai unha copia editable (imaxes incluídas) - a forma
// natural de partir dunha plantilla de fábrica sen perdela.
func (a *App) DuplicarPlantilla(id string) (PlantillaLatex, error) {
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return PlantillaLatex{}, err
	}
	for _, p := range alm.Plantillas {
		if p.ID != id {
			continue
		}
		copia := p
		copia.ID = ""
		copia.DeFabrica = false
		copia.Nome = p.Nome + " (copia)"
		nova, err := a.GardarPlantilla(copia)
		if err != nil {
			return PlantillaLatex{}, err
		}
		if err := copiarCartafolImaxes(id, nova.ID); err != nil {
			return nova, err
		}
		nova.Imaxes = imaxesDaPlantilla(nova.ID)
		return nova, nil
	}
	return PlantillaLatex{}, fmt.Errorf("non existe esa plantilla")
}

type SubirImaxePlantillaRequest struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
	DataB64  string `json:"dataB64"`
}

// SubirImaxePlantilla garda un logo (ou calquera imaxe) no cartafol da
// plantilla e devolve o nome co que hai que referenciala: na plantilla
// LaTeX \includegraphics{<nome>}, no Markdown ![](images/<nome>) - ver
// copiarImaxesPlantilla, que as deixa nos dous sitios no intre de xerar.
func (a *App) SubirImaxePlantilla(req SubirImaxePlantillaRequest) (string, error) {
	if strings.TrimSpace(req.ID) == "" {
		return "", fmt.Errorf("garda a plantilla antes de engadirlle imaxes")
	}
	data, err := base64.StdEncoding.DecodeString(req.DataB64)
	if err != nil {
		return "", fmt.Errorf("imaxe non válida: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("o ficheiro está baleiro")
	}
	nomeFicheiro, data, err := prepararImaxePlantilla(req.FileName, data)
	if err != nil {
		return "", err
	}
	dir, err := plantillaImaxesDir(req.ID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	nome := dedupeFileName(dir, nomeFicheiro, "imaxe")
	if err := os.WriteFile(filepath.Join(dir, nome), data, 0o644); err != nil {
		return "", err
	}
	return nome, nil
}

// prepararImaxePlantilla deixa un ficheiro listo para o cartafol de imaxes
// dunha plantilla, que LaTeX (e Word/OpenDocument) saiban usar:
//   - PDF: tal cal (vectorial: non perde calidade; Word/OpenDocument usan unha
//     imaxe de píxeles sacada da súa páxina 1);
//   - PNG e JPEG: tal cales;
//   - GIF, WebP, BMP, TIFF e os formatos raros (RAW, SVG, HEIC...): pasan a PNG;
//   - calquera outra cousa (unha fonte, un .sty...) vai sen tocar, co nome orixinal.
//
// O nome dunha imaxe ou PDF queda sen espazos nin símbolos.
func prepararImaxePlantilla(nome string, datos []byte) (string, []byte, error) {
	base := filepath.Base(strings.ReplaceAll(strings.TrimSpace(nome), "\\", "/"))
	if bytes.HasPrefix(datos, []byte("%PDF")) {
		return nomeSeguroImaxe(base, "pdf"), datos, nil
	}
	if precisaConversorExterno(base) {
		png, err := converterImaxeExternaAPNG(datos, base)
		if err != nil {
			return "", nil, err
		}
		return nomeSeguroImaxe(base, "png"), png, nil
	}
	tipo, mime := detectarTipoAdxunto(base, datos)
	if tipo != tipoAdxImaxe {
		return base, datos, nil
	}
	switch mime {
	case "image/png":
		return nomeSeguroImaxe(base, "png"), datos, nil
	case "image/jpeg":
		return nomeSeguroImaxe(base, "jpg"), datos, nil
	}
	png, _, _, _, err := redimensionarImaxe(datos, 0, "png")
	if err != nil {
		return "", nil, fmt.Errorf("non se puido usar «%s» coma imaxe: %w", base, err)
	}
	return nomeSeguroImaxe(base, "png"), png, nil
}

// PlantillaImaxeVista devolve unha miniatura (data URI) dunha imaxe da
// plantilla para o editor; un PDF debúxase pola súa primeira páxina. Baleiro
// se o ficheiro non é unha imaxe nin un PDF.
func (a *App) PlantillaImaxeVista(id, nome string) (string, error) {
	dir, err := plantillaImaxesDir(id)
	if err != nil {
		return "", err
	}
	datos, err := os.ReadFile(filepath.Join(dir, filepath.Base(nome)))
	if err != nil {
		return "", err
	}
	if bytes.HasPrefix(datos, []byte("%PDF")) {
		if datos, err = pdfAPNGLogo(datos, 1, 120); err != nil {
			return "", nil // sen poppler: sen miniatura, o resto funciona
		}
	}
	out, mime, _, _, err := redimensionarImaxe(datos, 160, "")
	if err != nil {
		return "", nil
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(out), nil
}

// EliminarImaxePlantilla borra unha imaxe da plantilla. filepath.Base sobre
// o nome recibido evita que un "../../algo" saia do cartafol da plantilla.
func (a *App) EliminarImaxePlantilla(id, nome string) error {
	dir, err := plantillaImaxesDir(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(nome) == "" {
		return fmt.Errorf("falta o nome da imaxe")
	}
	return os.Remove(filepath.Join(dir, filepath.Base(nome)))
}

// EscollerPlantilla fixa a plantilla activa e, se hai un documento aberto,
// lembra que ESE documento usa esta plantilla (id "" = ningunha). Chámase
// dende o selector da barra de ferramentas.
// EscollerPlantilla marca a plantilla activa (e, se se indica, lembra a
// escolla para ese documento). A maquinaria está en edicion/plantilla.
func (a *App) EscollerPlantilla(id, ficheiro string) error {
	return depositoPlantillas.Activar(id, ficheiro)
}

// PlantillaParaFicheiro devolve (e activa) a plantilla lembrada para ese
// documento ao abrilo. Se ese documento nunca tivo plantilla, mantense a
// activa actual.
func (a *App) PlantillaParaFicheiro(ficheiro string) (string, error) {
	return depositoPlantillas.ParaFicheiro(ficheiro)
}

// plantillaActiva resolve a plantilla que hai que aplicar agora mesmo, ou nil
// se non hai ningunha. Nunca devolve erro "brando": se o ficheiro está corrupto
// ou a plantilla activa desapareceu, xérase sen plantilla en vez de bloquear o
// traballo.
func (a *App) plantillaActiva() *PlantillaLatex {
	return depositoPlantillas.Activa()
}

// ---------- Substitución e composición ----------

// normalizarNomeVar e normalizarVariables están en edicion_plantilla.go:
// a implementación vive en edicion/plantilla.

// variablesAutomaticas son as que enche Lua soa en cada xeración. Unha
// variable definida na plantilla co mesmo nome ten prioridade (ex.: fixar
// un {{TITULO}} propio en vez do nome do ficheiro).
func variablesAutomaticas(nomeDoc string) map[string]string {
	nomeDoc = strings.TrimSpace(nomeDoc)
	return map[string]string{
		"DATA":     time.Now().Format("02/01/2006"),
		"FICHEIRO": nomeDoc,
		"TITULO":   nomeDoc,
		"ANO":      time.Now().Format("2006"),
	}
}

// baleiroLatex é o que se pon no sitio dunha variable SEN valor cando se
// está a compoñer LaTeX. Non pode ser a cadea baleira: unha liña de
// plantilla coma "{\\LARGE {{EMPRESA}} }\\\\[4pt]" quedaría cun \\\\ sen nada
// diante e pdflatex morre con "There's no line here to end" - un orzamento
// recén creado, coas variables aínda por encher, non compilaría. \\mbox{}
// non se ve no PDF pero xa é "algo" para LaTeX. En Markdown non fai falla
// nada disto (unha liña baleira alí non é erro), así que alí vai baleiro.
const baleiroLatex = `\mbox{}`

// As funcións de composición (substituír variables, paquetes declarados, motor da
// plantilla e copia de imaxes) viven en edicion/plantilla; as de abaixo son os
// adaptadores da aplicación.

// paqueteRequirido e paquetesMatexe viven en edicion_plantilla.go (adaptadores).

// inxectarPreambuloObrigatorio engade, xusto antes do \begin{document} dunha
// plantilla COMPLETA, o que fai falla para que o informe siga funcionando:
// os paquetes que falten, o soporte de UTF-8 propio de cada motor, babel, e
// `bloque` (as definicións do informe: cores, formato dos títulos, etiquetas
// das tarefas). Non toca nada do que a plantilla xa declare - a idea é que
// se poida cambiar clase, marxes e fontes libremente e que o informe siga
// saíndo igual.
// inxectarPreambuloObrigatorio: adaptador (ver edicion/plantilla).

// medidasMaquetaRe pilla as medidas de cabeceira e pé que unha maqueta fixa
// con \setlength{\headheight}{52pt}, \headsep ou \footskip.
var medidasMaquetaRe = regexp.MustCompile(`\\setlength\s*\{\\(headheight|headsep|footskip)\}\s*\{\s*(-?[0-9.]+\s*(?:pt|mm|cm|in|bp))\s*\}`)

// geometriaDaMaqueta arranxa unha trampa moi habitual: o paquete geometry calcula
// as marxes ao cargalo co \headheight, \headsep e \footskip que haxa entón. Se a
// maqueta os cambia despois con \setlength (sobre todo tras \begin{document}),
// o corpo do texto baixa e o pé de páxina (co número) sae fóra da folla. Aquí
// recóllense esas medidas e devólvese un \geometry{...} para poñer no preámbulo,
// antes de \begin{document}, que fai que geometry as teña en conta. Sen medidas
// devolve cadea baleira; se geometry non está cargado non fai nada. Se hai
// \headheight engade tamén includehead (ver abaixo).
func geometriaDaMaqueta(latex string) string {
	valores := map[string]string{}
	for _, m := range medidasMaquetaRe.FindAllStringSubmatch(latex, -1) {
		valores[m[1]] = strings.ReplaceAll(m[2], " ", "")
	}
	if len(valores) == 0 {
		return ""
	}
	var partes []string
	for _, nome := range []string{"headheight", "headsep", "footskip"} {
		if v, ok := valores[nome]; ok {
			partes = append(partes, nome+"="+v)
		}
	}
	// Unha cabeceira máis alta ca a por defecto (12 pt) non cabe na marxe de
	// arriba: sen isto saía por riba da folla. includehead mide a marxe ata o
	// TOPE da cabeceira, así fica á distancia da marxe e o texto empeza debaixo.
	if _, ok := valores["headheight"]; ok {
		partes = append(partes, "includehead")
	}
	return `\ifdefined\geometry\geometry{` + strings.Join(partes, ",") + `}\fi` + "\n"
}

// partesInformeCrono son os anacos do informe en LaTeX: a portada e a sinatura
// por defecto (aínda con {{VARIABLES}} sen substituír) e o corpo (índice,
// introdución, cronograma e desenvolvemento).
type partesInformeCrono struct {
	PortadaDef, Corpo, SinaturaDef string
	// SenPortada e SenSinatura quitan esas partes (unha acta solta non leva
	// portada; as actas asinan cada unha no seu corpo).
	SenPortada, SenSinatura bool
}

// documentoLatexConPlantilla compón o .tex final a partir do preámbulo do
// informe, as partes do informe e a plantilla escollida.
//
//   - Sen plantilla, ou cunha plantilla "clásica" (sen {{PORTADA}} nin
//     {{SINATURA}}): portada por defecto + corpo + sinatura por defecto, todo
//     xunto onde vaia {{CORPO}}.
//   - Cunha plantilla con algún deses marcadores: {{PORTADA}} e {{SINATURA}}
//     levan o fragmento propio da plantilla (ou o por defecto se está
//     baleiro) e {{CORPO}} só o corpo. Un marcador que falte quita esa parte.
//
// `paquetes`/`bloque` só se usan para unha plantilla COMPLETA (ver
// inxectarPreambuloObrigatorio); `auto` son as variables automáticas (ver
// variablesAutomaticasCrono).
func documentoLatexConPlantilla(preambulo string, partes partesInformeCrono, engine string, plantilla *PlantillaLatex, auto map[string]string, paquetes []paqueteRequirido, bloque string) (string, []string, error) {
	vacia := &PlantillaLatex{}
	if plantilla == nil {
		plantilla = vacia
		portada, av1 := substituirVariables(partes.PortadaDef, plantilla, auto, baleiroLatex, EscapeLatexProse)
		sinatura, av2 := substituirVariables(partes.SinaturaDef, plantilla, auto, baleiroLatex, EscapeLatexProse)
		if partes.SenPortada {
			portada = ""
		}
		if partes.SenSinatura {
			sinatura = ""
		}
		return preambulo + "\\begin{document}\n" + portada + partes.Corpo + sinatura + latexPostamble,
			engadirSenRepetir(av1, av2), nil
	}
	if !strings.Contains(plantilla.Latex, marcadorCorpo) {
		return "", nil, fmt.Errorf("a plantilla %q non leva %s", plantilla.Nome, marcadorCorpo)
	}
	texto, avisos := substituirVariables(plantilla.Latex, plantilla, auto, baleiroLatex, EscapeLatexProse)

	// Cada parte enche as variables coas da PROPIA plantilla (as suas
	// sobreescriben as do curso, coma na cabeceira).
	parte := func(fragmento, defecto string) string {
		if strings.TrimSpace(fragmento) == "" {
			fragmento = defecto
		}
		out, av := substituirVariables(fragmento, plantilla, auto, baleiroLatex, EscapeLatexProse)
		avisos = engadirSenRepetir(avisos, av)
		return out
	}

	corpo := partes.Corpo
	portada, sinatura := "", ""
	if strings.Contains(texto, marcadorPortada) || strings.Contains(texto, marcadorSinatura) {
		if strings.Contains(texto, marcadorPortada) {
			if !partes.SenPortada {
				portada = parte(plantilla.Portada, partes.PortadaDef)
			}
		} else if strings.TrimSpace(plantilla.Portada) != "" && !partes.SenPortada {
			avisos = append(avisos, "a plantilla ten unha portada pero o LaTeX non leva "+marcadorPortada+": non se usa")
		}
		if strings.Contains(texto, marcadorSinatura) {
			if !partes.SenSinatura {
				sinatura = parte(plantilla.Sinatura, partes.SinaturaDef)
			}
		} else if strings.TrimSpace(plantilla.Sinatura) != "" && !partes.SenSinatura {
			avisos = append(avisos, "a plantilla ten unha sinatura pero o LaTeX non leva "+marcadorSinatura+": non se usa")
		}
	} else {
		corpo = partes.Corpo
		if !partes.SenPortada {
			corpo = parte("", partes.PortadaDef) + corpo
		}
		if !partes.SenSinatura {
			corpo += parte("", partes.SinaturaDef)
		}
	}

	// As partes insírense DESPOIS de substituír as variables da maqueta: así
	// un {{...}} que aparecese no propio informe queda intacto.
	doc := strings.ReplaceAll(texto, marcadorPortada, portada)
	doc = strings.ReplaceAll(doc, marcadorSinatura, sinatura)
	doc = strings.ReplaceAll(doc, marcadorCorpo, corpo)
	if strings.Contains(doc, `\documentclass`) {
		return inxectarPreambuloObrigatorio(doc, engine, paquetes, bloque+geometriaDaMaqueta(texto)), avisos, nil
	}
	// Fragmento: a plantilla é só cabeceira/pé, o preámbulo pono Lua.
	return preambulo + geometriaDaMaqueta(texto) + "\\begin{document}\n" + doc + latexPostamble, avisos, nil
}

// engadirSenRepetir xunta avisos sen duplicar os que xa están (a mesma
// variable sen valor aparece na maqueta, na portada e na sinatura).
func engadirSenRepetir(base []string, novos ...[]string) []string {
	vistos := map[string]bool{}
	for _, a := range base {
		vistos[a] = true
	}
	for _, lista := range novos {
		for _, a := range lista {
			if !vistos[a] {
				vistos[a] = true
				base = append(base, a)
			}
		}
	}
	return base
}

// documentoMarkdownConPlantilla: adaptador (ver edicion/plantilla).

// copiarImaxesPlantilla* e escribirImaxesPlantilla: adaptadores (ver edicion/plantilla).

// motorParaPlantilla: adaptador (ver edicion/plantilla).

// PrevisualizarPlantilla compila a plantilla cun cronograma de exemplo e
// devolve as páxinas coma imaxes, exactamente polo mesmo camiño ca un Xerar
// normal (xerarPDFCrono) - así o que se ve na vista previa é o que vai saír de
// verdade, erros de LaTeX incluídos. A plantilla NON ten por que estar
// gardada; se xa o está, tamén se copian as súas imaxes.
func (a *App) PrevisualizarPlantilla(p PlantillaLatex) (GeneratePDFResult, error) {
	return a.PrevisualizarPlantillaDe(p, "")
}

// pedidoDemo é a petición co documento de mostra da ferramenta ("actas" ou, se
// non, o cronograma).
func pedidoDemo(ferramenta string) XerarInformeCronogramaRequest {
	if ferramenta == ferramentaActas {
		demo := planDemoActas()
		return XerarInformeCronogramaRequest{Ferramenta: ferramentaActas, Actas: &demo, Ambito: ambitoCursoActas, NomeBase: "actas-exemplo"}
	}
	return XerarInformeCronogramaRequest{Plan: planDemoCrono(), NomeBase: "cronograma-exemplo"}
}

// PrevisualizarPlantillaDe é PrevisualizarPlantilla co documento de mostra da
// ferramenta indicada ("cronograma" ou "actas").
func (a *App) PrevisualizarPlantillaDe(p PlantillaLatex, ferramenta string) (GeneratePDFResult, error) {
	return a.previsualizarPlantillaPedido(p, pedidoDemo(ferramenta))
}

// PrevisualizarPlantillaDocumento previsualiza a plantilla co documento que o
// usuario ten en edición (a mesma petición que se manda ao xerar), non cun
// documento de mostra.
func (a *App) PrevisualizarPlantillaDocumento(p PlantillaLatex, req XerarInformeCronogramaRequest) (GeneratePDFResult, error) {
	return a.previsualizarPlantillaPedido(p, req)
}

func (a *App) previsualizarPlantillaPedido(p PlantillaLatex, pedido XerarInformeCronogramaRequest) (GeneratePDFResult, error) {
	if strings.TrimSpace(p.Latex) == "" {
		return GeneratePDFResult{}, fmt.Errorf("a plantilla non ten código LaTeX")
	}
	if !strings.Contains(p.Latex, marcadorCorpo) {
		return GeneratePDFResult{}, fmt.Errorf("o LaTeX da plantilla ten que levar %s no sitio onde vai o informe", marcadorCorpo)
	}
	p.Variables = normalizarVariables(p.Variables)

	tmpDir, err := os.MkdirTemp("", "lua-plantilla-*")
	if err != nil {
		return GeneratePDFResult{}, err
	}
	defer os.RemoveAll(tmpDir)
	p.Textos = normalizarTextosDoc(p.Textos)
	return a.xerarPDFPedido(pedido, &p, tmpDir)
}
