package main

// Autoactualización de Lua: a lóxica xenérica (consulta das Releases de
// GitHub, verificación do SHA-256, substitución do executable en execución e
// relanzamento) vive unha soa vez en edicion/actualizacion, compartida con
// Yang e cos módulos futuros. Aquí queda só o que é propio de Lua: o nome, o
// repositorio, o prefixo do asset, o executable, o ficheiro VERSION embebido
// e o nome do evento de Wails.
//
// A Release ten que publicar un asset "lua_<versión>.zip" (que conteña o
// executable "lua"; o resto —frontend, idiomas— vai embebido con go:embed) e
// mais o seu ".sha256" en formato "sha256sum". AplicarActualizacionLuaEnCaliente
// compróbao antes de instalar nada: sen isto, quen puidese publicar unha
// Release podería facer executar calquera cousa, coas mesmas credenciais ca
// Lua, en cada equipo que actualizase.
import (
	_ "embed"

	"edicion/actualizacion"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed VERSION
var versionLuaFicheiro string

var axustesActualizacionLua = actualizacion.Axustes{
	Nome:         "Lúa",
	Repo:         "piztusistemas/lua",
	PrefixoAsset: "lua_",
	Executable:   "lua",
}

// EstadoActualizacionLua é o estado de actualización que se expón ao frontend.
// É un alias do tipo do módulo (non unha copia), de xeito que os bindings de
// Wails e as sinaturas de app.go seguen iguais e os nomes de campo JSON son
// exactamente os mesmos.
type EstadoActualizacionLua = actualizacion.Estado

func init() {
	application.RegisterEvent[EstadoActualizacionLua]("actualizacion_lua_cambiada")
}

// VersionLocalLua devolve a versión instalada de Lua, lida do VERSION
// embebido.
func VersionLocalLua() string { return actualizacion.VersionLocal(versionLuaFicheiro) }

// ComprobarActualizacionLua consulta as Releases de piztusistemas/lua en
// GitHub, queda coa máis recente que traia un asset de Lua adxunto, e compáraa
// coa versión instalada. Nunca devolve erro cara ao chamador: se falla a rede,
// non hai releases publicadas (ou ningunha é de Lua), ou a resposta non é
// válida, devolve Disponible=false para non bloquear o arranque de Lua.
func ComprobarActualizacionLua() EstadoActualizacionLua {
	return actualizacion.Comprobar(axustesActualizacionLua, VersionLocalLua())
}

// AplicarActualizacionLuaEnCaliente descarga o .zip da Release, verifica o seu
// SHA-256, extrae dentro del o executable de Lua e substitúeo polo que está en
// execución agora mesmo (← botón "Actualizar" no diálogo de información). Non
// reinicia por si mesma: quen chama decide cando (ver ReiniciarLua).
func AplicarActualizacionLuaEnCaliente(urlZip, urlChecksum, versionNova string) error {
	return actualizacion.AplicarEnCaliente(axustesActualizacionLua, urlZip, urlChecksum, versionNova)
}

// ReiniciarLua arrinca unha nova copia de Lua (xa a versión instalada por
// AplicarActualizacionLuaEnCaliente) coma proceso independente, para que quen
// chame poida pechar esta xanela decontado.
func ReiniciarLua() error { return actualizacion.Reiniciar(axustesActualizacionLua) }
