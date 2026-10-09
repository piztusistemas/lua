package main

// GenerateDocResult é o que devolve a exportación a Word/OpenDocument (e a
// .tex): a ruta onde se gardou (baleira se se cancelou) e os avisos.
type GenerateDocResult struct {
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
}
