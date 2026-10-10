package main

// Cliente de IA CONVERSACIONAL con chamadas a ferramentas ("function
// calling"), para o asistente de Lua (asistente.go). Fala os mesmos tres
// formatos de fío ca chamarIA (Gemini, Anthropic, "compatible con
// OpenAI"), e o código que fala con eles vive en edicion/ia, pero en vez dunha soa pregunta/resposta leva unha conversa
// enteira: mensaxes do usuario, respostas da IA (con texto e/ou chamadas a
// ferramentas) e resultados das ferramentas.
//
// O bucle (pedir -> executar as ferramentas -> volver pedir...) vive no
// frontend (frontend/src/asistente/), porque a maioría das ferramentas tocan
// o curso en edición, que só existe alí. Este ficheiro fai UN turno: recibe a
// conversa ata agora e devolve o que respondeu a IA.
//
// Cada resposta da IA gárdase tamén "en bruto" (IAMensaxe.Bruto, o JSON do
// provedor) e devólvese tal cal ao pedir o turno seguinte: hai provedores que
// esixen recibir de volta campos que Lua nin le (as "sinaturas de
// pensamento" de Gemini, o razoamento de DeepSeek...) e sen eles rexeitan a
// conversa.

import (
	"edicion/ia"
)

// Os tipos da conversa son os de edicion/ia: alias, non copias, para que os
// bindings de Wails e o JSON das conversas gardadas sigan iguais.
type (
	IAChamada    = ia.IAChamada
	IAResultado  = ia.IAResultado
	IAMensaxe    = ia.IAMensaxe
	IAFerramenta = ia.IAFerramenta
	IATurno      = ia.IATurno
)

// configDe traduce as Opcións de Lúa á configuración de edicion/ia, cos
// valores xa resoltos coma antes: Anthropic leva 8192 tokens se non hai
// límite (a súa API esixe un número) e os demais omiten o parámetro; o
// tempo máximo de espera segue a regra «< 30 s → 240 s».
func configDe(s Settings) ia.Config {
	max := s.IAMaxTokensResolto()
	if ProvedorIA(s.IAProvedor) == ProvedorAnthropic {
		max = s.IAMaxTokensAnthropicResolto()
	}
	return ia.Config{
		Provedor:        s.IAProvedor,
		BaseURL:         s.IABaseURL,
		APIKey:          s.IAAPIKey,
		Modelo:          s.IAModel,
		MaxTokens:       max,
		TimeoutSegundos: int(s.IATimeoutResolto().Seconds()),
	}
}

// comprobarIA di se as Opcións teñen clave e modelo, cos mesmos erros que
// daría unha chamada.
func comprobarIA(s Settings) error { return configDe(s).Validar() }

// chamarIAConversa pide á IA UN turno da conversa.
func chamarIAConversa(s Settings, sistema string, msgs []IAMensaxe, ferramentas []IAFerramenta) (IATurno, error) {
	return ia.Turno(configDe(s), sistema, msgs, ferramentas)
}
