// Package actualizacion implementa a autoactualización dunha aplicación de
// escritorio a partir das Releases dun repositorio de GitHub: comprobar se hai
// versión nova, descargala, verificar o seu SHA-256 e substituír o executable
// en execución.
//
// É xenérico agás por catro datos, que cada aplicación aporta nun Axustes: o
// seu nome (só para as mensaxes), o repositorio, o prefixo dos assets e o nome
// do executable. A lóxica —e as salvagardas de seguridade— viven unha soa vez
// aquí (ver PLAN-modulo-edicion.md).
//
// O módulo non coñece Wails: o rexistro do evento e o binding son da
// aplicación. Si usa edicion/plataforma para re-asinar o bundle en macOS e
// para relanzar o executable.
package actualizacion

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"edicion/plataforma"
)

// Axustes reúne o que distingue a actualización dunha aplicación da doutra.
type Axustes struct {
	// Nome é o nome visible da aplicación, usado nas mensaxes de erro.
	Nome string
	// Repo é o repositorio de GitHub onde se publican as Releases, en forma
	// "propietario/repo" (ex. "piztusistemas/yang").
	Repo string
	// PrefixoAsset é o prefixo obrigatorio dos assets da Release desta
	// aplicación (ex. "yang_"). Non abonda con "calquera .zip": a Release
	// podería levar outros adxuntos e collelo errado.
	PrefixoAsset string
	// Executable é o nome do executable dentro do .zip, sen a extensión .exe
	// (ex. "yang").
	Executable string
	// SoUltimaRelease: só se mira a ÚLTIMA Release do repositorio (a máis nova
	// que non sexa borrador nin preversión), non a máis recente que traia un
	// asset da aplicación. Úsano os módulos que viaxan coa propia aplicación
	// (Buxo vai na Release de Lua coa mesma versión): se esa Release non ten o
	// asset do módulo, non hai módulo novo, e non se vai buscar a unha Release vella.
	SoUltimaRelease bool
	// Release, se non é baleira, é a etiqueta (tag) da Release MÍNIMA que se
	// mira (ex. "26.10.1", a versión da aplicación anfitrioa): cóntanse esa
	// Release e calquera outra de versión maior (sen borradores nin
	// preversións), e de todas elas gaña o asset de versión maior. Así un
	// módulo republicado na Release actual ou publicado nunha máis nova
	// instálase sen agardar por outra versión da aplicación. Gaña sobre
	// SoUltimaRelease. Hai «versión dispoñible» só se a remota é maior ca
	// instalada.
	Release string

	// urlProbas permite sobrescribir a URL da API nas probas deste paquete.
	urlProbas string
}

const (
	timeoutComprobar = 5 * time.Second
	timeoutDescarga  = 15 * time.Minute // o zip de Buxo pasa de 120 MB
)

// urlDaAPI devolve a URL da API de Releases. per_page=100 cobre folgadamente
// todas as Releases dun módulo; se algún día se superan as 100 totais faría
// falta paxinar, pero non paga a pena a complexidade mentres non chegue ese
// caso.
func (ax Axustes) urlDaAPI() string {
	if ax.urlProbas != "" {
		return ax.urlProbas
	}
	return "https://api.github.com/repos/" + ax.Repo + "/releases?per_page=100"
}

// NomeExecutable é o nome do ficheiro que ten que haber dentro do .zip da
// Release: o executable tal cal o produce "wails build".
func (ax Axustes) NomeExecutable() string {
	if runtime.GOOS == "windows" {
		return ax.Executable + ".exe"
	}
	return ax.Executable
}

// Estado é o resultado de comprobar unha actualización — os nomes de campo van
// en minúscula inicial en JSON (camelCase) para encaixar coa convención xa
// usada polo resto de structs das aplicacións (Settings, InstallStatus...) que
// o frontend le directamente.
type Estado struct {
	Disponible    bool   `json:"disponible"`
	VersionLocal  string `json:"versionLocal"`
	VersionRemota string `json:"versionRemota"`
	URLDescarga   string `json:"urlDescarga"` // "" se non hai asset .zip descargable
	URLChecksum   string `json:"urlChecksum"` // "" se a Release non publica "<zip>.sha256"
	Notas         string `json:"notas"`       // descrición da Release (changelog)
}

type assetGitHub struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseGitHub struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Body       string        `json:"body"`
	Assets     []assetGitHub `json:"assets"`
}

// VersionLocal devolve a versión instalada, lida do ficheiro VERSION que a
// aplicación embebe na raíz do seu módulo.
func VersionLocal(versionFicheiro string) string {
	return strings.TrimSpace(versionFicheiro)
}

// escollerAssets busca, entre os assets dunha Release, o .zip da propia
// aplicación e o seu .sha256 — o prefixo vai cravado e non abonda con "calquera
// .zip": a Release podería levar outros adxuntos e collelo errado (ver o mesmo
// problema, xa sufrido en produción con Piztu, no commit "Arranxar
// autoactualización" de piztu).
//
// Os assets poden levar a plataforma ao final do nome
// («buxo_26.10.1_linux_amd64.zip»): escóllese o da plataforma actual e
// saltáronse os doutras. Un asset sen plataforma («lua_26.10.zip») vale para
// calquera e só se usa se non hai un da plataforma actual. Entre varios
// candidatos, o de versión maior.
func escollerAssets(ax Axustes, assets []assetGitHub) (urlZip, nomeZip, urlChecksum string) {
	prefixo := strings.ToLower(ax.PrefixoAsset)
	propio, xenerico := -1, -1
	// Se a Release leva varias versións do mesmo asset (buxo_26.10.1_… e
	// buxo_26.10.2_…), gaña a maior, non a primeira da lista.
	mellor := func(actual, i int) bool {
		return actual < 0 || VersionMaior(versionDesdeNomeZip(ax, assets[i].Name), versionDesdeNomeZip(ax, assets[actual].Name))
	}
	for i, a := range assets {
		nome := strings.ToLower(a.Name)
		if !strings.HasPrefix(nome, prefixo) || !strings.HasSuffix(nome, ".zip") {
			continue
		}
		_, plat := separarPlataforma(strings.TrimSuffix(nome, ".zip"))
		switch {
		case plat == plataformaActual && mellor(propio, i):
			propio = i
		case plat == "" && mellor(xenerico, i):
			xenerico = i
		}
	}
	if propio < 0 {
		propio = xenerico
	}
	if propio < 0 {
		return "", "", ""
	}
	urlZip, nomeZip = assets[propio].BrowserDownloadURL, assets[propio].Name
	for _, a := range assets {
		if strings.EqualFold(a.Name, nomeZip+".sha256") {
			urlChecksum = a.BrowserDownloadURL
			break
		}
	}
	return urlZip, nomeZip, urlChecksum
}

// plataformaActual é o sufixo de plataforma dos assets desta máquina
// («linux_amd64»). Variable para as probas.
var plataformaActual = runtime.GOOS + "_" + runtime.GOARCH

var plataformasConhecidas = map[string]bool{"linux": true, "windows": true, "darwin": true}
var arquitecturasConhecidas = map[string]bool{"amd64": true, "arm64": true, "386": true, "arm": true}

// separarPlataforma parte «nome_26.10.1_linux_amd64» en («nome_26.10.1»,
// «linux_amd64»); sen plataforma devolve o nome tal cal e "".
func separarPlataforma(nome string) (resto, plataforma string) {
	partes := strings.Split(nome, "_")
	if n := len(partes); n >= 3 && plataformasConhecidas[partes[n-2]] && arquitecturasConhecidas[partes[n-1]] {
		return strings.Join(partes[:n-2], "_"), partes[n-2] + "_" + partes[n-1]
	}
	return nome, ""
}

// versionDesdeNomeZip extrae o número de versión do nome do asset
// ("yang_26.9.zip" → "26.9"; "buxo_26.10.1_linux_amd64.zip" → "26.10.1").
// Parséase do nome do asset e non do tag_name da Release para non depender de
// que o tag sexa "só" o número de versión.
func versionDesdeNomeZip(ax Axustes, nomeZip string) string {
	nome := strings.ToLower(nomeZip)
	nome = strings.TrimSuffix(nome, ".zip")
	nome, _ = separarPlataforma(nome)
	nome = strings.TrimPrefix(nome, strings.ToLower(ax.PrefixoAsset))
	return strings.TrimSpace(nome)
}

// partesVersion e VersionMaior copian a comparación "semver-lite" de
// piztu/internal/modulos.VersionMaior — pequena e autónoma dabondo como para
// non merecer facer das aplicacións consumidoras de piztu/internal.
func partesVersion(v string) ([]int, bool) {
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "v"))
	if v == "" {
		return nil, false
	}
	anacos := strings.Split(v, ".")
	partes := make([]int, 0, len(anacos))
	for _, a := range anacos {
		n, err := strconv.Atoi(strings.TrimSpace(a))
		if err != nil {
			return nil, false
		}
		partes = append(partes, n)
	}
	return partes, true
}

// VersionMaior indica se a versión remota é maior ca local.
func VersionMaior(remota, local string) bool {
	r, okR := partesVersion(remota)
	if !okR {
		return false
	}
	l, okL := partesVersion(local)
	if !okL {
		l = []int{0}
	}
	for i := 0; i < len(r) || i < len(l); i++ {
		var rv, lv int
		if i < len(r) {
			rv = r[i]
		}
		if i < len(l) {
			lv = l[i]
		}
		if rv != lv {
			return rv > lv
		}
	}
	return false
}

// Comprobar consulta as Releases do repositorio da aplicación en GitHub, queda
// coa máis recente que traia un asset seu adxunto, e compáraa coa versión
// instalada. Nunca devolve erro cara ao chamador: se falla a rede, non hai
// releases publicadas (ou ningunha é da aplicación), ou a resposta non é
// válida, devolve Disponible=false para non bloquear o arranque.
func Comprobar(ax Axustes, versionLocal string) Estado {
	estado := Estado{VersionLocal: versionLocal}

	req, err := http.NewRequest(http.MethodGet, ax.urlDaAPI(), nil)
	if err != nil {
		return estado
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	cliente := http.Client{Timeout: timeoutComprobar}
	resp, err := cliente.Do(req)
	if err != nil {
		return estado
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return estado
	}

	var rels []releaseGitHub
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return estado
	}
	if ax.Release != "" {
		return comprobarDesdeRelease(ax, rels, estado)
	}

	// A API devolve as Releases máis recentes primeiro; o repo pode ser
	// compartido con outros módulos, así que quedamos coa primeira que traia
	// un asset "<prefixo>*.zip".
	for _, rel := range rels {
		if ax.SoUltimaRelease && (rel.Draft || rel.Prerelease) {
			continue // a «última» é a máis nova publicada de verdade
		}
		urlZip, nomeZip, urlChecksum := escollerAssets(ax, rel.Assets)
		if urlZip == "" {
			if ax.SoUltimaRelease {
				return estado // a última Release non leva o asset: nada novo
			}
			continue // Release doutro módulo (ou sen assets): seguinte
		}
		versionRemota := versionDesdeNomeZip(ax, nomeZip)
		if versionRemota == "" {
			if ax.SoUltimaRelease {
				return estado
			}
			continue
		}
		estado.VersionRemota = versionRemota
		estado.URLDescarga = urlZip
		estado.URLChecksum = urlChecksum
		estado.Notas = rel.Body
		estado.Disponible = VersionMaior(versionRemota, estado.VersionLocal)
		return estado
	}
	return estado // ningunha Release do repo trae un asset da aplicación
}

// comprobarDesdeRelease xunta os assets da Release ax.Release e das
// posteriores e queda co de versión maior (ver Axustes.Release).
func comprobarDesdeRelease(ax Axustes, rels []releaseGitHub, estado Estado) Estado {
	var assets []assetGitHub
	notas := map[string]string{} // nome do zip → notas da súa Release
	for _, rel := range rels {
		tag := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
		if tag != ax.Release && (rel.Draft || rel.Prerelease || !VersionMaior(tag, ax.Release)) {
			continue
		}
		for _, a := range rel.Assets {
			if _, ok := notas[a.Name]; !ok {
				notas[a.Name] = rel.Body
			}
		}
		assets = append(assets, rel.Assets...)
	}
	urlZip, nomeZip, urlChecksum := escollerAssets(ax, assets)
	versionRemota := versionDesdeNomeZip(ax, nomeZip)
	if urlZip == "" || versionRemota == "" {
		return estado // ningunha destas Releases leva o asset
	}
	estado.VersionRemota = versionRemota
	estado.URLDescarga = urlZip
	estado.URLChecksum = urlChecksum
	estado.Notas = notas[nomeZip]
	estado.Disponible = VersionMaior(versionRemota, estado.VersionLocal)
	return estado
}

func descargar(url, destinoDir, nomeFicheiro string) (string, error) {
	if err := os.MkdirAll(destinoDir, 0o755); err != nil {
		return "", fmt.Errorf("non se puido crear %q: %w", destinoDir, err)
	}
	destino := filepath.Join(destinoDir, nomeFicheiro)

	cliente := http.Client{Timeout: timeoutDescarga}
	resp, err := cliente.Get(url)
	if err != nil {
		return "", fmt.Errorf("non se puido descargar a actualización: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non se puido descargar a actualización: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destino)
	if err != nil {
		return "", fmt.Errorf("non se puido crear %q: %w", destino, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("non se puido gardar a actualización: %w", err)
	}
	return destino, nil
}

func extraerExecutable(ax Axustes, zipRuta, destDir string) (string, error) {
	zr, err := zip.OpenReader(zipRuta)
	if err != nil {
		return "", fmt.Errorf("o ficheiro descargado non é un zip válido: %w", err)
	}
	defer zr.Close()

	buscado := ax.NomeExecutable()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || filepath.Base(f.Name) != buscado {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("non se puido abrir %q dentro do zip: %w", f.Name, err)
		}
		defer rc.Close()

		destino := filepath.Join(destDir, buscado)
		out, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", fmt.Errorf("non se puido crear %q: %w", destino, err)
		}
		defer out.Close()
		if _, err := io.Copy(out, rc); err != nil {
			return "", fmt.Errorf("non se puido extraer %q: %w", f.Name, err)
		}
		return destino, nil
	}
	return "", fmt.Errorf("o zip da actualización non contén %q — revisa como o empaquetaches", buscado)
}

func lerHashEsperado(r io.Reader) (string, error) {
	datos, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return "", fmt.Errorf("non se puido ler o .sha256: %w", err)
	}
	campos := strings.Fields(string(datos))
	if len(campos) == 0 {
		return "", fmt.Errorf("o ficheiro .sha256 da Release está baleiro")
	}
	return strings.ToLower(campos[0]), nil
}

// comprobarChecksum descarga urlChecksum e verifica que coincide co SHA-256
// real de ficheiroRuta. Devolve erro (e non deixa continuar) se falta, non se
// pode descargar, ou non coincide — mellor cancelar a actualización ca instalar
// algo sen verificar.
func comprobarChecksum(ficheiroRuta, urlChecksum string) error {
	if urlChecksum == "" {
		return fmt.Errorf("esta Release non publica un .sha256 do .zip — non se pode verificar a súa integridade, así que se cancela a actualización por seguridade")
	}

	cliente := http.Client{Timeout: timeoutComprobar}
	resp, err := cliente.Get(urlChecksum)
	if err != nil {
		return fmt.Errorf("non se puido descargar o .sha256 da actualización: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non se puido descargar o .sha256 da actualización: HTTP %d", resp.StatusCode)
	}
	esperado, err := lerHashEsperado(resp.Body)
	if err != nil {
		return err
	}

	f, err := os.Open(ficheiroRuta)
	if err != nil {
		return fmt.Errorf("non se puido abrir %q para verificalo: %w", ficheiroRuta, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("non se puido calcular o SHA-256 de %q: %w", ficheiroRuta, err)
	}
	obtido := hex.EncodeToString(h.Sum(nil))

	if obtido != esperado {
		return fmt.Errorf("o SHA-256 do .zip descargado non coincide co publicado na Release — podería estar adulterado; cancélase a actualización")
	}
	return nil
}

func copiarFicheiro(orixe, destino string) error {
	in, err := os.Open(orixe)
	if err != nil {
		return fmt.Errorf("non se puido abrir %q: %w", orixe, err)
	}
	defer in.Close()
	out, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("non se puido crear %q: %w", destino, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("non se puido copiar a %q: %w", destino, err)
	}
	return nil
}

// actualizarVersionModuloJSON reescribe (best-effort) o campo "version" do
// modulo.json que vive canda o executable, para que ⚙ Aula → Módulos de Piztu
// reflicta a nova versión sen agardar ao próximo escaneo do catálogo. Non é
// fatal se falla (modulo.json podería non estar presente en execucións soltas
// fóra do sistema de módulos): AplicarEnCaliente xa fixo o importante
// (substituír o executable) antes de chamar aquí.
func actualizarVersionModuloJSON(dirModulo, versionNova string) {
	ruta := filepath.Join(dirModulo, "modulo.json")
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return
	}
	var m map[string]any
	if err := json.Unmarshal(datos, &m); err != nil {
		return
	}
	m["version"] = versionNova
	novosDatos, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(ruta, novosDatos, 0o644)
}

// AplicarEnCaliente descarga o .zip da Release en `urlZip`, verifica o seu
// SHA-256 contra o publicado en `urlChecksum`, extrae dentro del o executable
// da aplicación e substitúeo polo que está en execución agora mesmo — sen
// pechar a aplicación nin descomprimir nada á man (← botón "Actualizar" no
// diálogo de información). Non reinicia por si mesma: quen chama decide cando
// (ver Reiniciar).
//
// Se falta o checksum, non se pode descargar, ou non coincide, non se toca
// nada — mellor cancelar a actualización ca instalar un binario sen verificar
// (ver comprobarChecksum).
//
// A substitución escribe primeiro o binario novo canda o vello (mesmo cartafol
// → mesmo sistema de ficheiros) e só ao final fai un os.Rename atómico enriba
// del: se algo falla a metade, o binario orixinal non se toca. Antes gárdase
// tamén unha copia coma "<executable>.anterior".
func AplicarEnCaliente(ax Axustes, urlZip, urlChecksum, versionNova string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("a actualización automática aínda non está soportada en Windows")
	}

	exeActual, err := os.Executable()
	if err != nil {
		return fmt.Errorf("non se puido localizar o executable actual: %w", err)
	}
	exeActual, err = filepath.EvalSymlinks(exeActual)
	if err != nil {
		return fmt.Errorf("non se puido resolver o executable actual: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", ax.Executable+"-actualizacion-")
	if err != nil {
		return fmt.Errorf("non se puido crear un cartafol temporal: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipRuta, err := descargar(urlZip, tmpDir, "actualizacion.zip")
	if err != nil {
		return err
	}
	if err := comprobarChecksum(zipRuta, urlChecksum); err != nil {
		return err
	}
	novoExe, err := extraerExecutable(ax, zipRuta, tmpDir)
	if err != nil {
		return err
	}

	swapTmp := exeActual + ".novo"
	defer os.Remove(swapTmp) // non-op se xa se renomeou
	if err := copiarFicheiro(novoExe, swapTmp); err != nil {
		return err
	}

	_ = copiarFicheiro(exeActual, exeActual+".anterior") // copia de reserva; non crítico se falla

	if err := os.Rename(swapTmp, exeActual); err != nil {
		return fmt.Errorf("non se puido instalar o binario novo: %w", err)
	}

	// En macOS substituír o executable rompe a sinatura do .app enteiro e a
	// app deixaría de abrir - hai que volver asinala (no-op noutras
	// plataformas, ver edicion/plataforma/bundle_darwin.go). Se non se pode,
	// desfacer: mellor quedar coa versión anterior funcionando ca cunha
	// instalación que o sistema xa non deixa arrincar.
	if err := plataforma.ResinarBundleMacOS(exeActual); err != nil {
		if errRestaurar := copiarFicheiro(exeActual+".anterior", exeActual); errRestaurar != nil {
			return fmt.Errorf("%w (e ademais non se puido restaurar a versión anterior: %v)", err, errRestaurar)
		}
		_ = plataforma.ResinarBundleMacOS(exeActual) // best-effort: devolver o bundle ao estado de antes
		return fmt.Errorf("actualización desfeita, segues na versión anterior: %w", err)
	}

	if versionNova != "" {
		actualizarVersionModuloJSON(filepath.Dir(exeActual), versionNova)
	}
	return nil
}

// Reiniciar arrinca unha nova copia da aplicación (xa a versión instalada por
// AplicarEnCaliente) coma proceso independente, para que quen chame poida
// pechar esta xanela decontado.
func Reiniciar(ax Axustes) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("non se puido localizar o executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("non se puido resolver o executable: %w", err)
	}

	// En macOS hai que relanzar o .app, non o binario de dentro (ver
	// edicion/plataforma/bundle_darwin.go); noutras plataformas isto é
	// exec.Command(exe).
	cmd := plataforma.ComandoRelanzar(exe)
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("non se puido reiniciar %s: %w", ax.Nome, err)
	}
	return nil
}
