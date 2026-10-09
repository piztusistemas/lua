package cas

import (
	"os/exec"

	"edicion/plataforma"
)

// Ponte co módulo común: estes dous axudantes viven unha soa vez en
// edicion/plataforma. Antes estaban duplicados neste paquete (cas é un paquete
// aparte e importar de main sería circular, ver o antigo exec.go), en main de
// Yang e en main de Lua: tres e catro copias da mesma función. Agora a
// implementación é única e consérvanse estes nomes locais para non tocar as
// chamadas que xa existen no paquete.
func commandExists(name string) bool { return plataforma.CommandExists(name) }

func ocultarConsola(cmd *exec.Cmd) { plataforma.OcultarConsola(cmd) }
