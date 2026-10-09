package main

// GenerateMarkdownResult é o que devolve a exportación a Markdown: a ruta
// onde se gardou (baleira se se cancelou o diálogo) e os avisos (imaxes
// descartadas...). Ver ExportarInformeMarkdown, informe_cronograma.go.
type GenerateMarkdownResult struct {
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
}
