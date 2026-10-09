// Comando sincronizarweb copia o frontend compartido do módulo de edición
// (edicion/web/estatico/) ás aplicacións que o consomen, escribindo en cada
// destino un MANIFEST.sha256.
//
// Uso:
//
//	cd edicion
//	go run ./cmd/sincronizarweb ../yang/yang/frontend/src/comun \
//	                                ../lua/lua/frontend/src/comun
//
// Os ficheiros de destino son XERADOS: edítanse en edicion/web/estatico/ e
// vólvense sincronizar. Cada aplicación ten un test que verifica a copia.
package main

import (
	"fmt"
	"os"

	"edicion/web"
)

func main() {
	destinos := os.Args[1:]
	if len(destinos) == 0 {
		fmt.Fprintln(os.Stderr, "uso: go run ./cmd/sincronizarweb <destino>...")
		os.Exit(2)
	}
	for _, destino := range destinos {
		if err := web.Sincronizar(destino); err != nil {
			fmt.Fprintf(os.Stderr, "erro sincronizando %s: %v\n", destino, err)
			os.Exit(1)
		}
		nomes, err := web.Ficheiros()
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro lendo a fonte: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("sincronizado %s (%d ficheiros)\n", destino, len(nomes))
	}
}
