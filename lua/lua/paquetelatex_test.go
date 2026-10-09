package main

import "testing"

// TestPaqueteLatexBinding comproba só o encaixe da aplicación: que o binding
// que le o frontend reflicte o que anotaron as compilacións e que limpar o
// deixa baleiro. A lóxica (recoñecemento no log, tradución a paquete do
// sistema, comando de instalación) ten os seus propios tests en
// edicion/latex/paquetes, que é onde vive.
func TestPaqueteLatexBinding(t *testing.T) {
	defer limparPaqueteQueFalta()
	a := &App{}

	anotarPaqueteQueFalta("! LaTeX Error: File `tcolorbox.sty' not found.\nl.7 \\usepackage")
	info := a.PaqueteLatexQueFalta()
	if info.Ficheiro != "tcolorbox.sty" || info.Nome != "tcolorbox" {
		t.Fatalf("o binding non reflicte o paquete anotado: %+v", info)
	}

	limparPaqueteQueFalta()
	if info := a.PaqueteLatexQueFalta(); info.Ficheiro != "" {
		t.Fatalf("tras limpar non pode quedar nada: %+v", info)
	}

	// Sen nada pendente, instalar non executa nada (nin se pode dar por boa).
	if st := a.InstalarPaqueteLatex(); st.Success {
		t.Errorf("non pode haber instalación satisfactoria sen paquete pendente: %+v", st)
	}
}

// TestPaqueteLatexErroAlleo: un erro de LaTeX que non é de paquete non pode
// deixar o botón de "instalar" pendurado.
func TestPaqueteLatexErroAlleo(t *testing.T) {
	defer limparPaqueteQueFalta()
	a := &App{}

	anotarPaqueteQueFalta("! Undefined control sequence.\nl.4 \\foo")
	if info := a.PaqueteLatexQueFalta(); info.Ficheiro != "" {
		t.Errorf("un erro que non é de paquete non pode deixar nada pendente: %+v", info)
	}
}
