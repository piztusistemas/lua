package main

import (
	"encoding/json"
	"os"
)

// contextoModulo é o subconxunto do contrato que Piztu escribe ao lanzar un
// módulo externo (ver ContextoModulo/escribirContexto en piztu/app.go). Lua
// só precisa o idioma.
type contextoModulo struct {
	// Idioma é o código (gl/es/en/pt) que o profesorado ten escollido en ⚙
	// Aula para o propio Piztu - Lua úsao coma valor por defecto do seu
	// idioma SÓ SE aínda non se escolleu un idioma propio en Opcións
	// (Settings.Idioma == ""), ver IdiomaPiztu() e frontend/src/main.js. Lua
	// pódese instalar e usar sen Piztu (executable á man), así que isto é
	// sempre un valor por defecto, nunca unha orde: "" cando non hai
	// contexto de lanzamento.
	Idioma string `json:"idioma"`
}

// lerContextoLanzamento intenta ler o contexto que Piztu escribiu ao abrir
// Lua (variable de contorno PIZTU_CONTEXTO). Devolve ok=false se Lua se
// abriu sen pasar por Piztu (ex.: en desenvolvemento, executando o binario
// á man).
func lerContextoLanzamento() (contextoModulo, bool) {
	var ctx contextoModulo
	ruta := os.Getenv("PIZTU_CONTEXTO")
	if ruta == "" {
		return ctx, false
	}
	dados, err := os.ReadFile(ruta)
	if err != nil {
		return ctx, false
	}
	if err := json.Unmarshal(dados, &ctx); err != nil {
		return ctx, false
	}
	return ctx, true
}
