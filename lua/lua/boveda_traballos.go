package main

// TRABALLOS NA BÓVEDA (spec traballos-boveda): un documento feito en Lúa
// (boletín, exame, cronograma, actas) pódese engadir á bóveda de Obsidian
// coma unha nota Markdown co seu PDF ao lado, en wiki/traballos/. Só cando o
// usuario o pide (botón, ou a marca «engadir sempre» dun formulario).
//
// A bóveda é a mesma que usa o asistente para ese documento (bovedaDe); se
// non hai ningunha, créase a do proxecto. Un formulario en PRODUCIÓN nunca
// vai á bóveda: o asistente le a bóveda e os datos reais non deben chegarlle.

import (
	"edicion/idioma"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const dirTraballosBoveda = "wiki/traballos"

// ResultadoTraballoBoveda di onde quedou o traballo.
type ResultadoTraballoBoveda struct {
	Boveda string   `json:"boveda"`
	Nota   string   `json:"nota"`
	PDF    string   `json:"pdf,omitempty"`
	Creada bool     `json:"creada"` // creouse a bóveda do proxecto agora
	Avisos []string `json:"avisos,omitempty"`
}

// documentoDaPeticion: o tipo de documento (para escoller a bóveda).
func documentoDaPeticion(req XerarInformeCronogramaRequest) string {
	switch {
	case req.eFormulario():
		return "formulario"
	case req.eActas():
		return "actas"
	}
	return "cronograma"
}

// peticionEnProduccion: a petición é dun formulario en produción (mírase o que
// chega e tamén o formulario gardado, por se o frontend vai atrasado).
func peticionEnProduccion(req XerarInformeCronogramaRequest) bool {
	if !req.eFormulario() {
		return false
	}
	f := req.Formulario.Formulario
	if f.enProduccion() {
		return true
	}
	if gardado, err := lerFormulario(f.ID); err == nil && gardado.enProduccion() {
		return true
	}
	return false
}

func erroProduccionBoveda() error {
	return idioma.Errorf("go.boveda.produccion", "este formulario está en PRODUCIÓN: os seus datos poden ser reais e o asistente le a bóveda, así que non se engade")
}

// bovedaParaTraballo devolve a bóveda onde vai o traballo; se non hai
// ningunha, crea a do proxecto aberto.
func (a *App) bovedaParaTraballo(documento string) (raiz string, creada bool, err error) {
	if _, ruta := a.bovedaDe(documento); ruta != "" {
		r, err := validarRaizBoveda(ruta)
		return r, false, err
	}
	pid := proxectoAbertoValido()
	if pid == "" {
		return "", false, idioma.Errorf("go.catalogo.sen_proxecto", "non hai ningún proxecto aberto")
	}
	ruta, err := rutaBovedaProxecto(pid)
	if err != nil {
		return "", false, err
	}
	for _, d := range []string{"raw", "wiki", "pdf"} {
		if err := os.MkdirAll(filepath.Join(ruta, d), 0o755); err != nil {
			return "", false, err
		}
	}
	r, err := validarRaizBoveda(ruta)
	return r, true, err
}

// engadirTraballoBoveda escribe a nota (e o PDF, se hai) do documento.
func (a *App) engadirTraballoBoveda(req XerarInformeCronogramaRequest, pdfBase64, nome string) (ResultadoTraballoBoveda, error) {
	var res ResultadoTraballoBoveda
	if peticionEnProduccion(req) {
		return res, erroProduccionBoveda()
	}
	var pdf []byte
	if strings.TrimSpace(pdfBase64) != "" {
		var err error
		if pdf, err = base64.StdEncoding.DecodeString(pdfBase64); err != nil {
			return res, fmt.Errorf("o PDF non é válido: %w", err)
		}
	}
	documento := documentoDaPeticion(req)
	raiz, creada, err := a.bovedaParaTraballo(documento)
	if err != nil {
		return res, err
	}
	res.Boveda, res.Creada = raiz, creada
	base := nomeFicheiroPlantilla(strings.TrimSpace(nome))
	if base == "" || base == "plantilla" {
		base = documento
	}
	dir := filepath.Join(raiz, filepath.FromSlash(dirTraballosBoveda))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, err
	}
	plantilla := a.plantillaActiva()
	md, avisos := markdownInformeConPlantilla(a.conMotorProblemas(req), plantilla, "md", dir)
	res.Avisos = avisos
	if err := copiarImaxesPlantillaMarkdown(plantilla, dir); err != nil {
		res.Avisos = append(res.Avisos, err.Error())
	}
	agora := time.Now()
	nomeProxecto := ""
	if pid := proxectoAbertoValido(); pid != "" {
		if p, err := lerProxecto(pid); err == nil {
			nomeProxecto = p.Nome
		}
	}
	var cab strings.Builder
	cab.WriteString("---\n")
	cab.WriteString("tipo: traballo\n")
	fmt.Fprintf(&cab, "documento: %s\n", documento)
	if nomeProxecto != "" {
		fmt.Fprintf(&cab, "proxecto: %q\n", nomeProxecto)
	}
	fmt.Fprintf(&cab, "data: %s\n", agora.Format("2006-01-02"))
	if pdf != nil {
		fmt.Fprintf(&cab, "pdf: %q\n", base+".pdf")
	}
	cab.WriteString("---\n\n")
	if pdf != nil {
		fmt.Fprintf(&cab, "> %s · [[%s.pdf|PDF]]\n\n", idioma.T("go.boveda.traballo_cab", "Traballo engadido por Lúa o %s", agora.Format("2006-01-02")), base)
	}
	// Primeiro o PDF e despois a nota: unha nota nunca liga a un PDF que falta.
	if pdf != nil {
		res.PDF = filepath.Join(dir, base+".pdf")
		if err := escribirAtomicoCrono(res.PDF, pdf); err != nil {
			return res, err
		}
	}
	res.Nota = filepath.Join(dir, base+".md")
	if err := escribirAtomicoCrono(res.Nota, []byte(cab.String()+md)); err != nil {
		return res, err
	}
	entrada := fmt.Sprintf("## [%s] traballo | %s\n", agora.Format("2006-01-02"), base)
	if _, err := escribirNaBoveda(&boveda{raiz}, "wiki/log.md", entrada, true); err != nil {
		res.Avisos = append(res.Avisos, err.Error())
	}
	return res, nil
}

// ---------- Bindings ----------

// EngadirTraballoBoveda engade o documento (nota Markdown e PDF) á bóveda.
func (a *App) EngadirTraballoBoveda(req XerarInformeCronogramaRequest, pdfBase64, nome string) (ResultadoTraballoBoveda, error) {
	return a.engadirTraballoBoveda(req, pdfBase64, nome)
}

// BovedaAutomaticaDe di se o documento `ferramenta` do proxecto aberto se
// engade á bóveda en cada Xerar.
func (a *App) BovedaAutomaticaDe(ferramenta string) bool {
	pid := proxectoAbertoValido()
	if pid == "" {
		return false
	}
	p, err := lerProxecto(pid)
	return err == nil && contemString(p.BovedaAutomatica, ferramenta)
}

// FixarBovedaAutomatica activa ou desactiva «engadir sempre á bóveda ao
// xerar» para un formulario do proxecto aberto (nunca en produción).
func (a *App) FixarBovedaAutomatica(ferramenta string, activa bool) error {
	pid := proxectoAbertoValido()
	if pid == "" {
		return idioma.Errorf("go.catalogo.sen_proxecto", "non hai ningún proxecto aberto")
	}
	fid, ok := idFerramentaFormulario(ferramenta)
	if !ok {
		return fmt.Errorf("só os formularios teñen esta opción: %q", ferramenta)
	}
	if activa {
		if f, err := lerFormulario(fid); err == nil && f.enProduccion() {
			return erroProduccionBoveda()
		}
	}
	p, err := lerProxecto(pid)
	if err != nil {
		return err
	}
	p.BovedaAutomatica = quitarString(p.BovedaAutomatica, ferramenta)
	if activa {
		p.BovedaAutomatica = append(p.BovedaAutomatica, ferramenta)
	}
	if len(p.BovedaAutomatica) == 0 {
		p.BovedaAutomatica = nil
	}
	return escribirProxecto(p)
}
