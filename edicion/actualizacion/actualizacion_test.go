package actualizacion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func axProbas() Axustes {
	return Axustes{
		Nome:         "Probex",
		Repo:         "piztusistemas/probex",
		PrefixoAsset: "probex_",
		Executable:   "probex",
	}
}

func TestVersionDesdeNomeZip(t *testing.T) {
	ax := axProbas()
	casos := map[string]string{
		"probex_26.9.zip":                  "26.9",
		"PROBEX_26.9.6.zip":                "26.9.6",
		"probex_.zip":                      "",
		"probex_26.10.1_linux_amd64.zip":   "26.10.1",
		"PROBEX_26.10.1_windows_arm64.zip": "26.10.1",
	}
	for entrada, agardado := range casos {
		if obtido := versionDesdeNomeZip(ax, entrada); obtido != agardado {
			t.Errorf("versionDesdeNomeZip(%q) = %q, agardábase %q", entrada, obtido, agardado)
		}
	}
}

func TestVersionMaior(t *testing.T) {
	casos := []struct {
		remota, local string
		maior         bool
	}{
		{"26.9.6", "26.9.5", true},
		{"26.9.5", "26.9.6", false},
		{"26.9.6", "26.9.6", false},
		{"v26.10", "26.9.9", true},
		{"27", "26.9.9", true},
		{"26.9", "26.9.0", false},
		{"26.9.0.1", "26.9", true},
		{"non-version", "26.9", false}, // versión remota inválida: nunca actualiza
		{"26.9", "lixo", true},         // local inválida: trátase coma 0
	}
	for _, c := range casos {
		if obtido := VersionMaior(c.remota, c.local); obtido != c.maior {
			t.Errorf("VersionMaior(%q, %q) = %v, agardábase %v", c.remota, c.local, obtido, c.maior)
		}
	}
}

// TestEscollerAssets comproba o filtro polo prefixo propio: a Release pode
// levar adxuntos doutros módulos e non se pode coller "calquera .zip".
func TestEscollerAssets(t *testing.T) {
	ax := axProbas()
	assets := []assetGitHub{
		{Name: "outromodulo_1.0.zip", BrowserDownloadURL: "https://exemplo/outro.zip"},
		{Name: "PROBEX_26.9.zip", BrowserDownloadURL: "https://exemplo/probex.zip"},
		{Name: "probex_26.9.zip.sha256", BrowserDownloadURL: "https://exemplo/probex.zip.sha256"},
		{Name: "notas.txt", BrowserDownloadURL: "https://exemplo/notas.txt"},
	}
	urlZip, nomeZip, urlChecksum := escollerAssets(ax, assets)
	if urlZip != "https://exemplo/probex.zip" {
		t.Errorf("urlZip = %q", urlZip)
	}
	if nomeZip != "PROBEX_26.9.zip" {
		t.Errorf("nomeZip = %q", nomeZip)
	}
	if urlChecksum != "https://exemplo/probex.zip.sha256" {
		t.Errorf("urlChecksum = %q", urlChecksum)
	}

	// Sen ningún asset propio non se escolle nada (nin o .zip doutro módulo).
	if u, _, _ := escollerAssets(ax, assets[:1]); u != "" {
		t.Errorf("escolleu o asset doutro módulo: %q", u)
	}
	// Sen .sha256, o zip escóllese igual pero sen checksum (quen chame cancela).
	if _, _, c := escollerAssets(ax, assets[1:2]); c != "" {
		t.Errorf("checksum inesperado: %q", c)
	}
}

func TestLerHashEsperado(t *testing.T) {
	const hash = "34cb17c2f8b82e3650569b0f14a5cbdf690bf9e60a57f712fd6204a8ff818d65"
	casos := []struct {
		entrada  string
		agardado string
		erro     bool
	}{
		{hash, hash, false},
		{hash + "  probex_26.9.zip\n", hash, false},
		{hash + " *probex_26.9.zip\n", hash, false},
		{strings.ToUpper(hash), hash, false},
		{"", "", true},
		{"   \n", "", true},
	}
	for _, c := range casos {
		obtido, err := lerHashEsperado(strings.NewReader(c.entrada))
		if c.erro {
			if err == nil {
				t.Errorf("lerHashEsperado(%q) non deu erro", c.entrada)
			}
			continue
		}
		if err != nil {
			t.Errorf("lerHashEsperado(%q): %v", c.entrada, err)
			continue
		}
		if obtido != c.agardado {
			t.Errorf("lerHashEsperado(%q) = %q, agardábase %q", c.entrada, obtido, c.agardado)
		}
	}
}

// TestComprobarEscolleAReleasePropia e TestComprobarSenAssetPropio exercitan a
// selección contra un servidor de probas: o JSON da API reproducese tal cal o
// devolve GitHub (a Release máis recente primeiro) e compróbase que se queda
// coa nosa e non coa doutro módulo.
func TestComprobarEscolleAReleasePropia(t *testing.T) {
	rels := []releaseGitHub{
		{TagName: "v1.0", Body: "outro módulo", Assets: []assetGitHub{
			{Name: "outro_1.0.zip", BrowserDownloadURL: "https://exemplo/outro.zip"},
		}},
		{TagName: "v26.9.6", Body: "novidades", Assets: []assetGitHub{
			{Name: "probex_26.9.6.zip", BrowserDownloadURL: "https://exemplo/probex.zip"},
			{Name: "probex_26.9.6.zip.sha256", BrowserDownloadURL: "https://exemplo/probex.zip.sha256"},
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("falta o Accept da API de GitHub: %q", got)
		}
		_ = json.NewEncoder(w).Encode(rels)
	}))
	defer srv.Close()

	ax := axProbas()
	ax.urlProbas = srv.URL

	estado := Comprobar(ax, "26.9.5")
	if !estado.Disponible {
		t.Errorf("non detectou a versión nova: %+v", estado)
	}
	if estado.VersionRemota != "26.9.6" || estado.VersionLocal != "26.9.5" {
		t.Errorf("versións mal: %+v", estado)
	}
	if estado.URLDescarga != "https://exemplo/probex.zip" {
		t.Errorf("URLDescarga = %q", estado.URLDescarga)
	}
	if estado.URLChecksum != "https://exemplo/probex.zip.sha256" {
		t.Errorf("URLChecksum = %q", estado.URLChecksum)
	}
	if estado.Notas != "novidades" {
		t.Errorf("Notas = %q", estado.Notas)
	}

	// Coa mesma versión instalada, non hai nada que actualizar.
	if estado := Comprobar(ax, "26.9.6"); estado.Disponible {
		t.Errorf("marcou como dispoñible unha actualización á mesma versión: %+v", estado)
	}
}

func TestComprobarSenAssetPropio(t *testing.T) {
	rels := []releaseGitHub{
		{TagName: "v1.0", Assets: []assetGitHub{{Name: "outro_1.0.zip", BrowserDownloadURL: "https://exemplo/outro.zip"}}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rels)
	}))
	defer srv.Close()

	ax := axProbas()
	ax.urlProbas = srv.URL

	estado := Comprobar(ax, "1.0")
	if estado.Disponible || estado.URLDescarga != "" || estado.VersionRemota != "" {
		t.Errorf("con só assets doutro módulo debería quedar baleiro: %+v", estado)
	}
}

func TestComprobarNonRompeConHTTPNinJSONMalo(t *testing.T) {
	ax := axProbas()

	// Servidor que responde 500.
	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()
	ax.urlProbas = srv500.URL
	if estado := Comprobar(ax, "1.0"); estado.Disponible {
		t.Errorf("un 500 non pode dar actualización dispoñible: %+v", estado)
	}

	// Servidor que responde algo que non é JSON.
	srvMalo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>non son JSON</html>"))
	}))
	defer srvMalo.Close()
	ax.urlProbas = srvMalo.URL
	if estado := Comprobar(ax, "1.0"); estado.Disponible {
		t.Errorf("unha resposta inválida non pode dar actualización dispoñible: %+v", estado)
	}
}

// Os assets por plataforma: escóllese o desta máquina, nunca o doutro sistema, e o
// xenérico só se non hai un propio.
func TestEscollerAssetsPorPlataforma(t *testing.T) {
	ax := axProbas()
	antes := plataformaActual
	defer func() { plataformaActual = antes }()
	assets := []assetGitHub{
		{Name: "probex_26.10.1_windows_amd64.zip", BrowserDownloadURL: "u/win"},
		{Name: "probex_26.10.1_linux_amd64.zip", BrowserDownloadURL: "u/linux"},
		{Name: "probex_26.10.1_linux_amd64.zip.sha256", BrowserDownloadURL: "u/linux.sha"},
		{Name: "probex_26.10.1_darwin_arm64.zip", BrowserDownloadURL: "u/mac"},
		{Name: "probex_26.10.zip", BrowserDownloadURL: "u/xenerico"},
	}
	plataformaActual = "linux_amd64"
	if u, n, c := escollerAssets(ax, assets); u != "u/linux" || n != "probex_26.10.1_linux_amd64.zip" || c != "u/linux.sha" {
		t.Errorf("linux: %q %q %q", u, n, c)
	}
	plataformaActual = "darwin_arm64"
	if u, _, _ := escollerAssets(ax, assets); u != "u/mac" {
		t.Errorf("mac: %q", u)
	}
	// Plataforma sen asset propio: cae ao xenérico (se non hai, a nada).
	plataformaActual = "windows_arm64"
	if u, _, _ := escollerAssets(ax, assets); u != "u/xenerico" {
		t.Errorf("sen asset propio debe usar o xenérico: %q", u)
	}
	if u, _, _ := escollerAssets(ax, assets[:4]); u != "" {
		t.Errorf("nunca o binario doutro sistema: %q", u)
	}
}

// Unha Release con dúas versións do mesmo asset (o módulo republicado na
// Release da aplicación): escóllese a maior, estea onde estea na lista.
func TestEscollerAssetsVersionMaior(t *testing.T) {
	ax := axProbas()
	antes := plataformaActual
	defer func() { plataformaActual = antes }()
	plataformaActual = "darwin_arm64"
	assets := []assetGitHub{
		{Name: "probex_26.10.1_darwin_arm64.zip", BrowserDownloadURL: "u/vello"},
		{Name: "probex_26.10.1_darwin_arm64.zip.sha256", BrowserDownloadURL: "u/vello.sha"},
		{Name: "probex_26.10.10_darwin_arm64.zip", BrowserDownloadURL: "u/novo"},
		{Name: "probex_26.10.10_darwin_arm64.zip.sha256", BrowserDownloadURL: "u/novo.sha"},
		{Name: "probex_26.10.2_darwin_arm64.zip", BrowserDownloadURL: "u/medio"},
	}
	if u, n, c := escollerAssets(ax, assets); u != "u/novo" || n != "probex_26.10.10_darwin_arm64.zip" || c != "u/novo.sha" {
		t.Errorf("debe escoller a versión maior: %q %q %q", u, n, c)
	}
}

// Con SoUltimaRelease só se mira a última Release publicada: se non leva o asset
// non se vai buscar a unha vella; os borradores e preversións non contan.
func TestComprobarSoUltimaRelease(t *testing.T) {
	rels := []releaseGitHub{
		{TagName: "26.11-rc", Prerelease: true, Assets: []assetGitHub{{Name: "probex_26.11.zip", BrowserDownloadURL: "u/rc"}}},
		{TagName: "26.10.2", Assets: []assetGitHub{{Name: "lua_26.10.2.zip", BrowserDownloadURL: "u/lua"}}}, // sen o módulo
		{TagName: "26.10.1", Assets: []assetGitHub{
			{Name: "probex_26.10.1.zip", BrowserDownloadURL: "u/vella"},
			{Name: "probex_26.10.1.zip.sha256", BrowserDownloadURL: "u/vella.sha"},
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(rels) }))
	defer srv.Close()
	ax := axProbas()
	ax.urlProbas = srv.URL

	// Modo normal: a máis recente co asset, aínda que non sexa a última (e a preversión tamén vale).
	if e := Comprobar(ax, ""); e.URLDescarga != "u/rc" {
		t.Errorf("modo normal: %+v", e)
	}
	// Só a última: a 26.10.2 non leva o módulo, non hai nada que descargar (nin da vella nin da preversión).
	ax.SoUltimaRelease = true
	if e := Comprobar(ax, ""); e.URLDescarga != "" || e.Disponible {
		t.Errorf("só a última sen o asset: %+v", e)
	}
	// Se a última o leva, esa é a que vale e coa versión dela.
	rels[1].Assets = append(rels[1].Assets,
		assetGitHub{Name: "probex_26.10.2_linux_amd64.zip", BrowserDownloadURL: "u/nova"},
		assetGitHub{Name: "probex_26.10.2_linux_amd64.zip.sha256", BrowserDownloadURL: "u/nova.sha"})
	antes := plataformaActual
	plataformaActual = "linux_amd64"
	defer func() { plataformaActual = antes }()
	e := Comprobar(ax, "26.10.1")
	if !e.Disponible || e.URLDescarga != "u/nova" || e.URLChecksum != "u/nova.sha" || e.VersionRemota != "26.10.2" {
		t.Errorf("última co asset: %+v", e)
	}
	if e := Comprobar(ax, "26.10.2"); e.Disponible {
		t.Errorf("mesma versión: %+v", e)
	}
}

// Con Release mírase a Release desa etiqueta e as posteriores: gaña o asset de
// versión maior (republicado na actual ou nunha Release máis nova), e só hai
// descarga se é maior ca instalada.
func TestComprobarRelease(t *testing.T) {
	rels := []releaseGitHub{
		{TagName: "26.11", Prerelease: true, Assets: []assetGitHub{
			{Name: "probex_26.11_windows_amd64.zip", BrowserDownloadURL: "u/pre"},
		}},
		{TagName: "26.10.2", Assets: []assetGitHub{
			{Name: "probex_26.10.2_linux_amd64.zip", BrowserDownloadURL: "u/linux"},
		}},
		{TagName: "26.10.1", Assets: []assetGitHub{
			{Name: "probex_26.10.1_windows_amd64.zip", BrowserDownloadURL: "u/win"},
			{Name: "probex_26.10.1_windows_amd64.zip.sha256", BrowserDownloadURL: "u/win.sha"},
		}},
		{TagName: "26.9", Assets: []assetGitHub{
			{Name: "probex_26.12_windows_amd64.zip", BrowserDownloadURL: "u/vella"},
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(rels) }))
	defer srv.Close()
	ax := axProbas()
	ax.urlProbas = srv.URL
	ax.Release = "26.10.1"
	antes := plataformaActual
	plataformaActual = "windows_amd64"
	defer func() { plataformaActual = antes }()

	// Só a Release propia ten asset de Windows (a preversión e a anterior non contan).
	for _, c := range []struct {
		local string
		disp  bool
	}{{"", true}, {"26.9", true}, {"26.10.2", false}, {"26.10.1", false}} {
		e := Comprobar(ax, c.local)
		if e.Disponible != c.disp || e.URLDescarga != "u/win" || e.URLChecksum != "u/win.sha" || e.VersionRemota != "26.10.1" {
			t.Errorf("local %q: %+v", c.local, e)
		}
	}
	// Republicado na Release actual: gaña a versión maior.
	rels[2].Assets = append(rels[2].Assets,
		assetGitHub{Name: "probex_26.10.3_windows_amd64.zip", BrowserDownloadURL: "u/win3"})
	if e := Comprobar(ax, "26.10.1"); !e.Disponible || e.URLDescarga != "u/win3" {
		t.Errorf("republicado na actual: %+v", e)
	}
	// Nunha Release máis nova: tamén se colle.
	rels[1].Assets = append(rels[1].Assets,
		assetGitHub{Name: "probex_26.10.4_windows_amd64.zip", BrowserDownloadURL: "u/win4"})
	if e := Comprobar(ax, "26.10.3"); !e.Disponible || e.URLDescarga != "u/win4" || e.VersionRemota != "26.10.4" {
		t.Errorf("Release máis nova: %+v", e)
	}
	plataformaActual = "darwin_arm64"
	if e := Comprobar(ax, ""); e.URLDescarga != "" || e.Disponible {
		t.Errorf("sen asset da plataforma: %+v", e)
	}
}
