package main

// Os ficheiros que acepta Buxo e como se converte cada un, coma en
// NotebookLM: subas o que subas, acaba en raw/ coma Markdown (co LaTeX das
// fórmulas e os debuxos, se os hai).
//
//	PDF         markitdown (texto) e, se ten fórmulas, debuxos ou está
//	            escaneado, a IA le as páxinas (vision.go)
//	imaxe       a IA (vision.go): unha foto dun exame, un esquema...
//	Markdown    tal cal (é xa o formato de raw/); se ten fórmulas en texto
//	            plano e se pide, a IA pásaas a LaTeX
//	ofimática   markitdown (Word, PowerPoint, Excel, HTML, EPUB)
//
// Os orixinais quedan en pdf/ (o nome do cartafol é o de sempre: tamén garda
// os que non son PDF).

import (
	"edicion/idioma"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Tipos de ficheiro.
const (
	tipoPDF       = "pdf"
	tipoImaxe     = "imaxe"
	tipoMarkdown  = "markdown"
	tipoOfimatica = "ofimatica"
)

var tipoPorExtension = map[string]string{
	".pdf": tipoPDF,
	".png": tipoImaxe, ".jpg": tipoImaxe, ".jpeg": tipoImaxe, ".webp": tipoImaxe, ".gif": tipoImaxe,
	".md": tipoMarkdown, ".markdown": tipoMarkdown, ".txt": tipoMarkdown,
	".docx": tipoOfimatica, ".pptx": tipoOfimatica, ".xlsx": tipoOfimatica, ".html": tipoOfimatica, ".htm": tipoOfimatica, ".epub": tipoOfimatica,
}

// tipoDe di o tipo dun ficheiro pola extensión ("" = non se acepta).
func tipoDe(nome string) string { return tipoPorExtension[strings.ToLower(filepath.Ext(nome))] }

// patronsAceptados: para o filtro do selector de ficheiros.
func patronsAceptados() string {
	var p []string
	for ext := range tipoPorExtension {
		p = append(p, "*"+ext, "*"+strings.ToUpper(ext))
	}
	return strings.Join(p, ";")
}

// Modos de conversión (o selector da interface).
const (
	modoAuto   = "auto"   // a IA le as páxinas só se fai falla (fórmulas, debuxos, escaneado)
	modoVision = "vision" // a IA le sempre as páxinas (o mellor resultado, o máis caro)
	modoTexto  = "texto"  // só markitdown (gratis; as fórmulas e os debuxos pérdense)
)

func modoValido(m string) string {
	switch m {
	case modoVision, modoTexto:
		return m
	}
	return modoAuto
}

// nomeSeguro: coma nomePDFSeguro pero conservando a extensión do ficheiro.
func nomeSeguro(nome string) string {
	ext := strings.ToLower(filepath.Ext(nome))
	return strings.TrimSuffix(nomePDFSeguro(nome), ".pdf") + ext
}

// CopiarFicheiro copia un ficheiro aceptado a dir e devolve o nome co que
// quedou. Un .pdf ten que selo de verdade.
func CopiarFicheiro(orixe, dir string) (string, error) {
	tipo := tipoDe(orixe)
	if tipo == "" {
		return "", idioma.Errorf("erro.tipo_ficheiro", "«%s»: Buxo non sabe tratar ese tipo de ficheiro", filepath.Base(orixe))
	}
	if tipo == tipoPDF {
		return CopiarPDF(orixe, dir)
	}
	f, err := os.Open(orixe)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	nome := nomeLibre(dir, nomeSeguro(filepath.Base(orixe)))
	destino := filepath.Join(dir, nome)
	tmp := destino + ".parte"
	g, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(g, f)
	if e := g.Close(); err == nil {
		err = e
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return nome, os.Rename(tmp, destino)
}

// nomeMD: o .md de raw/ que lle corresponde a un orixinal.
func nomeMD(orixinal string) string {
	return strings.TrimSuffix(orixinal, filepath.Ext(orixinal)) + ".md"
}
