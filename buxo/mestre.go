package main

// Mestre leva a cola de PDF dunha bóveda: copia/descarga cada PDF a pdf/ e,
// segundo van chegando, convérteos a Markdown en raw/ sen intervención do
// usuario. Un PDF que falla (descarga ou conversión) queda marcado con erro e
// anotado no rexistro, pero nunca para os demais.

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Estados dun ficheiro da cola.
const (
	estPendente    = "pendente"
	estDescargando = "descargando"
	estConvertendo = "convertendo"
	estFeito       = "feito"
	estErro        = "erro"
)

// Item é un ficheiro da cola (o que ve a interface).
type Item struct {
	ID     int    `json:"id"`
	Orixe  string `json:"orixe"` // local | url
	Fonte  string `json:"fonte"` // ruta ou URL de onde vén
	Nome   string `json:"nome"`  // nome en pdf/ (baleiro ata que se descarga)
	Estado string `json:"estado"`
	Erro   string `json:"erro,omitempty"`
	// Modo de conversión (modoAuto, modoVision, modoTexto) e o que se está a
	// facer agora («a IA le as páxinas 3-4 de 12»).
	Modo    string `json:"modo"`
	Detalle string `json:"detalle,omitempty"`
	// Vision: convertiuse coa IA (fórmulas, debuxos) e por que.
	Vision string `json:"vision,omitempty"`
}

type Mestre struct {
	raiz    string
	materia string
	log     *Rexistro
	cliente *http.Client
	sidecar string // "" = buscalo ao converter

	ctx    context.Context
	parar  context.CancelFunc
	cambio func() // avisa a interface
	// cambioTarefas avisa cando muda o progreso da construción (pode ser nil).
	cambioTarefas func()

	// ia: a IA para ler as páxinas (vision.go); nil = sen IA. modo: o modo
	// de conversión dos ficheiros que se engaden.
	ia   *ClienteIA
	modo string
	// visor: a cola das conversións coa IA (unha de cada vez).
	visor chan tarefaVision

	mu      sync.Mutex
	items   []*Item
	tarefas []*Tarefa
	seq     int
	cola    chan *Item
	descarg chan struct{} // semáforo de descargas
	vivos   sync.WaitGroup
}

// NovoMestre crea o mestre e arrinca o seu converso. cambio avisa cada vez que
// a cola muda (pode ser nil).
func NovoMestre(raiz, materia string, log *Rexistro, cambio func()) *Mestre {
	ctx, parar := context.WithCancel(context.Background())
	m := &Mestre{
		raiz: raiz, materia: materia, log: log, cambio: cambio,
		cliente: &http.Client{Timeout: timeoutDescarga},
		ctx:     ctx, parar: parar,
		cola:    make(chan *Item, 4096),
		descarg: make(chan struct{}, 3),
		visor:   make(chan tarefaVision, 4096),
		modo:    modoAuto,
	}
	m.vivos.Add(2)
	go m.converso()
	go m.visorIA()
	return m
}

// tarefaVision: un ficheiro para a IA e por que.
type tarefaVision struct {
	it     *Item
	motivo string
	// textoPrevio: o que deu markitdown (se a IA falla, queda iso).
	textoPrevio bool
}

// CambiarModo fixa o modo de conversión dos ficheiros que se engadan.
func (m *Mestre) CambiarModo(modo string) {
	m.mu.Lock()
	m.modo = modoValido(modo)
	m.mu.Unlock()
}

func (m *Mestre) iaVision() *ClienteIA {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ia
}

// visorIA fai as conversións coa IA, unha tras outra.
func (m *Mestre) visorIA() {
	defer m.vivos.Done()
	for {
		select {
		case t := <-m.visor:
			m.poñer(t.it, func(i *Item) { i.Estado, i.Vision, i.Detalle = estConvertendo, t.motivo, "agardando pola IA" })
			err := m.converterConVision(m.ctx, t.it, t.motivo)
			switch {
			case err == nil:
				m.poñer(t.it, func(i *Item) { i.Estado, i.Detalle = estFeito, "" })
				m.log.Info("rex.convertido_ia", "Convertido coa IA raw/%s", nomeMD(t.it.Nome))
			case t.textoPrevio:
				m.poñer(t.it, func(i *Item) { i.Estado, i.Detalle, i.Vision = estFeito, "", "" })
				m.log.Aviso("rex.ia_non_le", "%s: a IA non o puido ler (%v); queda o texto de markitdown (sen fórmulas nin debuxos)", t.it.Nome, err)
			default:
				m.fallar(t.it, err.Error())
			}
		case <-m.ctx.Done():
			return
		}
	}
}

// aVision manda un ficheiro á cola da IA.
func (m *Mestre) aVision(it *Item, motivo string, textoPrevio bool) {
	m.poñer(it, func(i *Item) { i.Estado, i.Vision, i.Detalle = estConvertendo, motivo, "en cola para a IA" })
	select {
	case m.visor <- tarefaVision{it: it, motivo: motivo, textoPrevio: textoPrevio}:
	case <-m.ctx.Done():
	}
}

// Pechar para o converso e agarda polos traballos en curso.
func (m *Mestre) Pechar() {
	m.parar()
	m.vivos.Wait()
}

func (m *Mestre) avisar() {
	if m.cambio != nil {
		m.cambio()
	}
}

// Items devolve unha copia da cola.
func (m *Mestre) Items() []Item {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Item, len(m.items))
	for i, it := range m.items {
		out[i] = *it
	}
	return out
}

func (m *Mestre) novoItem(orixe, fonte string) *Item {
	m.mu.Lock()
	m.seq++
	it := &Item{ID: m.seq, Orixe: orixe, Fonte: fonte, Estado: estPendente, Modo: m.modo}
	m.items = append(m.items, it)
	m.mu.Unlock()
	return it
}

func (m *Mestre) poñer(it *Item, f func(*Item)) {
	m.mu.Lock()
	f(it)
	m.mu.Unlock()
	m.avisar()
}

// Quitar saca un ficheiro da lista se aínda non se está a tratar (os PDF xa
// copiados a pdf/ e os .md de raw/ non se borran).
func (m *Mestre) Quitar(id int) {
	m.mu.Lock()
	for i, it := range m.items {
		if it.ID == id && (it.Estado == estPendente || it.Estado == estErro || it.Estado == estFeito) {
			m.items = append(m.items[:i], m.items[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
	m.avisar()
}

// EngadirLocais pon na cola ficheiros do computador (PDF, imaxes, Markdown, ofimática).
func (m *Mestre) EngadirLocais(rutas []string) {
	for _, r := range rutas {
		it := m.novoItem("local", r)
		m.avisar()
		m.lanzar(it)
	}
}

// EngadirURLs pon na cola URL que apuntan directamente a un PDF.
func (m *Mestre) EngadirURLs(urls []string) {
	for _, u := range urls {
		it := m.novoItem("url", u)
		m.avisar()
		m.lanzar(it)
	}
}

func (m *Mestre) lanzar(it *Item) {
	m.vivos.Add(1)
	go func() {
		defer m.vivos.Done()
		select {
		case m.descarg <- struct{}{}:
			defer func() { <-m.descarg }()
		case <-m.ctx.Done():
			return
		}
		m.obter(it)
	}()
}

// obter deixa o PDF en pdf/ (copia ou descarga) e pásao ao converso.
func (m *Mestre) obter(it *Item) {
	if err := CrearEstrutura(m.raiz, m.materia); err != nil {
		m.fallar(it, "non se puido crear a bóveda: "+err.Error())
		return
	}
	dir := filepath.Join(m.raiz, dirPDF)
	var nome string
	var err error
	if it.Orixe == "url" {
		m.poñer(it, func(i *Item) { i.Estado = estDescargando })
		m.log.Info("rex.descargando", "Descargando %s", it.Fonte)
		nome, err = DescargarPDF(m.ctx, m.cliente, it.Fonte, dir)
	} else {
		m.log.Info("rex.copiando", "Copiando %s", it.Fonte)
		nome, err = CopiarFicheiro(it.Fonte, dir)
	}
	if err != nil {
		m.fallar(it, err.Error())
		return
	}
	m.poñer(it, func(i *Item) { i.Nome, i.Estado = nome, estConvertendo })
	m.log.Info("rex.gardado", "Gardado pdf/%s", nome)
	m.encamiñar(it)
}

// encamiñar manda o ficheiro (xa en pdf/) ao conversor que lle toca.
func (m *Mestre) encamiñar(it *Item) {
	switch tipoDe(it.Nome) {
	case tipoImaxe:
		if it.Modo == modoTexto {
			m.fallar(it, "unha imaxe só se pode ler coa IA (cambia o modo a Automático)")
			return
		}
		m.aVision(it, "é unha imaxe", false)
		return
	case tipoMarkdown:
		m.vivos.Add(1)
		go func() { defer m.vivos.Done(); m.converterMarkdown(it) }()
		return
	case tipoPDF:
		if it.Modo == modoVision {
			m.aVision(it, "lectura coa IA pedida", false)
			return
		}
	}
	select {
	case m.cola <- it:
	case <-m.ctx.Done():
	}
}

// converterMarkdown: un .md vai tal cal a raw/; se ten fórmulas en texto
// plano (ou se pide sempre a IA), a IA pásaas a LaTeX.
func (m *Mestre) converterMarkdown(it *Item) {
	d, err := os.ReadFile(filepath.Join(m.raiz, dirPDF, it.Nome))
	if err != nil {
		m.fallar(it, err.Error())
		return
	}
	md, conversion := string(d), "markdown"
	if it.Modo == modoVision || it.Modo == modoAuto && mdTenFormulasEnTexto(md) {
		if novo, err := m.normalizarMarkdown(m.ctx, it, md); err == nil {
			md, conversion = novo, "markdown (fórmulas pasadas a LaTeX coa IA)"
			m.poñer(it, func(i *Item) { i.Vision = "fórmulas en texto plano" })
		} else if err.Error() != "sen IA" {
			m.log.Aviso("rex.formulas_fallan", "%s: non se puideron pasar as fórmulas a LaTeX (%v); queda tal cal", it.Nome, err)
		}
	}
	if err := m.escribirRaw(it, md, conversion); err != nil {
		m.fallar(it, err.Error())
		return
	}
	m.poñer(it, func(i *Item) { i.Estado, i.Detalle = estFeito, "" })
	m.log.Info("rex.engadido", "Engadido raw/%s", nomeMD(it.Nome))
}

func (m *Mestre) fallar(it *Item, motivo string) {
	m.poñer(it, func(i *Item) { i.Estado, i.Erro = estErro, motivo })
	m.log.Erro("rex.saltase_motivo", "%s: %s (sáltase)", nomeAmosable(it), motivo)
}

func nomeAmosable(it *Item) string {
	if it.Nome != "" {
		return it.Nome
	}
	return it.Fonte
}

// converso converte en lotes os PDF que van chegando: agarda ao primeiro e
// recolle os que cheguen nun momento, para lanzar o conversor (que tarda en
// arrincar) unha soa vez por lote.
func (m *Mestre) converso() {
	defer m.vivos.Done()
	for {
		var lote []*Item
		select {
		case it := <-m.cola:
			lote = append(lote, it)
		case <-m.ctx.Done():
			return
		}
		linger := time.NewTimer(400 * time.Millisecond)
	recoller:
		for {
			select {
			case it := <-m.cola:
				lote = append(lote, it)
			case <-linger.C:
				break recoller
			case <-m.ctx.Done():
				linger.Stop()
				return
			}
		}
		m.converterLote(lote)
	}
}

func (m *Mestre) converterLote(lote []*Item) {
	sidecar := m.sidecar
	if sidecar == "" {
		s, err := rutaSidecar()
		if err != nil {
			for _, it := range lote {
				m.fallar(it, err.Error())
			}
			return
		}
		sidecar = s
	}
	porRuta := map[string]*Item{}
	var rutas []string
	for _, it := range lote {
		p := filepath.Join(m.raiz, dirPDF, it.Nome)
		porRuta[p] = it
		rutas = append(rutas, p)
	}
	m.log.Info("rex.convertendo", "Convertendo %d PDF a Markdown…", len(rutas))
	raw := filepath.Join(m.raiz, dirRaw)
	err := ConverterPDFs(m.ctx, sidecar, raw, rutas, func(r ResultadoConversion) {
		it := porRuta[r.Entrada]
		if it == nil {
			return
		}
		if !r.Ok {
			m.fallar(it, "non se puido converter: "+r.Erro)
			return
		}
		if err := engadirCabeceira(r.Saida, it); err != nil {
			m.fallar(it, err.Error())
			return
		}
		// Un PDF con fórmulas, debuxos ou escaneado: a IA le as páxinas.
		if tipoDe(it.Nome) == tipoPDF && it.Modo == modoAuto && m.iaVision() != nil {
			md, _ := os.ReadFile(r.Saida)
			p := filepath.Join(m.raiz, dirPDF, it.Nome)
			if motivo := motivoVision(string(md), paxinasPDF(p), figurasPDF(p)); motivo != "" {
				m.aVision(it, motivo, true)
				return
			}
		}
		m.poñer(it, func(i *Item) { i.Estado = estFeito })
		m.log.Info("rex.convertido", "Convertido raw/%s", filepath.Base(r.Saida))
	})
	if err != nil {
		for _, it := range lote {
			m.mu.Lock()
			feito := it.Estado == estFeito || it.Estado == estErro
			m.mu.Unlock()
			if !feito {
				m.fallar(it, err.Error())
			}
		}
	}
}

func engadirCabeceira(rutaMD string, it *Item) error {
	d, err := os.ReadFile(rutaMD)
	if err != nil {
		return err
	}
	url := ""
	if it.Orixe == "url" {
		url = it.Fonte
	}
	return os.WriteFile(rutaMD, []byte(cabeceiraFonte(string(d), it.Nome, url)), 0o644)
}

// Recuperar reconstrúe a cola cos PDF que xa hai en pdf/ (ao reabrir Buxo):
// os que teñen o seu .md en raw/ quedan feitos; os demais ponse a converter.
func (m *Mestre) Recuperar() {
	ents, err := os.ReadDir(filepath.Join(m.raiz, dirPDF))
	if err != nil {
		return
	}
	var nomes []string
	for _, e := range ents {
		if !e.IsDir() && tipoDe(e.Name()) != "" {
			nomes = append(nomes, e.Name())
		}
	}
	sort.Strings(nomes)
	var pendentes []*Item
	for _, n := range nomes {
		it := m.novoItem("local", filepath.Join(dirPDF, n))
		it.Nome = n
		md := filepath.Join(m.raiz, dirRaw, nomeMD(n))
		if _, err := os.Stat(md); err == nil {
			it.Estado = estFeito
			continue
		}
		it.Estado = estConvertendo
		pendentes = append(pendentes, it)
	}
	m.avisar()
	for _, it := range pendentes {
		m.encamiñar(it)
	}
}

// Contas di cantos ficheiros hai en cada estado.
func (m *Mestre) Contas() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := map[string]int{}
	for _, it := range m.items {
		c[it.Estado]++
	}
	return c
}

// Ocupado: hai algo en descarga ou conversión.
func (m *Mestre) Ocupado() bool {
	c := m.Contas()
	return c[estPendente]+c[estDescargando]+c[estConvertendo] > 0
}

// Estados dunha tarefa da construción.
const (
	tarPendente   = "pendente"
	tarExecutando = "executando"
	tarFeita      = "feita"
	tarErro       = "erro"
)

// Tarefa é un paso da construción da bóveda (procesar unha fonte, revisar a
// wiki...). Pasos conta as ferramentas que a IA chamou: serve para amosar que
// a tarefa avanza aínda que non se saiba canto lle falta.
type Tarefa struct {
	ID     int    `json:"id"`
	Nome   string `json:"nome"`
	Estado string `json:"estado"`
	Pasos  int    `json:"pasos"`
	Erro   string `json:"erro,omitempty"`
}

// Tarefas devolve unha copia das tarefas da última construción.
func (m *Mestre) Tarefas() []Tarefa {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Tarefa, len(m.tarefas))
	for i, t := range m.tarefas {
		out[i] = *t
	}
	return out
}

func (m *Mestre) novasTarefas(nomes []string) []*Tarefa {
	m.mu.Lock()
	m.tarefas = nil
	for i, n := range nomes {
		m.tarefas = append(m.tarefas, &Tarefa{ID: i + 1, Nome: n, Estado: tarPendente})
	}
	out := append([]*Tarefa(nil), m.tarefas...)
	m.mu.Unlock()
	m.avisarTarefas()
	return out
}

func (m *Mestre) mudarTarefa(t *Tarefa, f func(*Tarefa)) {
	m.mu.Lock()
	f(t)
	m.mu.Unlock()
	m.avisarTarefas()
}

func (m *Mestre) avisarTarefas() {
	if m.cambioTarefas != nil {
		m.cambioTarefas()
	}
}
