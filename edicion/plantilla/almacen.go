package plantilla

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Deposito é a biblioteca de plantillas dunha aplicación: o ficheiro
// plantillas.json, o cartafol de imaxes de cada plantilla e a fábrica de
// exemplos.
//
// Todo o que depende de rutas ou do dominio entra por estas costuras, de xeito
// que o módulo non coñece nin o nome da aplicación nin onde garda a súa
// configuración (ver a regra de ouro en edicion/doc.go).
type Deposito struct {
	// RutaAlmacen devolve a ruta completa do ficheiro plantillas.json.
	RutaAlmacen func() (string, error)
	// DirImaxes devolve o cartafol de imaxes DUNHA plantilla. Un cartafol por
	// plantilla (non un común) para que duplicar/borrar unha plantilla sexa
	// copiar/borrar un cartafol, sen levar contas de que logo usa quen.
	DirImaxes func(id string) (string, error)
	// Fabrica devolve as plantillas de exemplo que hai que sementar. Se é nil,
	// a aplicación non ten fábrica.
	Fabrica func() []Plantilla
	// FabricaObsoleta son as plantillas de exemplo de versións anteriores que
	// xa non se sementan: ID -> prefixo hex do SHA-256 do LaTeX orixinal, que é
	// o que permite saber se alguén as tocou. Pode ser nil.
	FabricaObsoleta map[string]string
	// ExisteFicheiro comproba que un documento segue no disco (poda da memoria
	// "que plantilla usaba cada ficheiro"). Se é nil, non se poda.
	ExisteFicheiro func(ruta string) bool
	// TextosIguais compara dous mapas de textos de documento (o catálogo de
	// claves é de dominio). Se é nil, compáranse literalmente.
	TextosIguais func(a, b map[string]string) bool
}

// agora devolve o instante actual no formato que usan Creado/Modificado.
func agora() string { return time.Now().Format(time.RFC3339) }

// novaID xera a ID dunha plantilla nova.
func novaID() string { return strconv.FormatInt(time.Now().UnixNano(), 36) }

// Ler abre o plantillas.json da aplicación. Se non existe, sementa a fábrica e
// créao. En calquera caso pasa polas migracións (reparación de fábrica editada,
// retirada de fábrica obsoleta e alta das de fábrica que falten).
func (d *Deposito) Ler() (Almacen, error) {
	var alm Almacen
	path, err := d.RutaAlmacen()
	if err != nil {
		return alm, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return d.Sementar(Almacen{})
		}
		return alm, err
	}
	if err := json.Unmarshal(b, &alm); err != nil {
		return alm, fmt.Errorf("plantillas.json ilexible: %w", err)
	}
	if err := alm.ComprobarVersion(); err != nil {
		return alm, err
	}
	alm, err = d.RepararFabrica(alm)
	if err != nil {
		return alm, err
	}
	alm, err = d.RetirarFabricaObsoleta(alm)
	if err != nil {
		return alm, err
	}
	return d.Sementar(alm)
}

// Gardar escribe o almacén no ficheiro da aplicación, coa versión de formato
// actual e coa memoria por ficheiro podada.
func (d *Deposito) Gardar(alm Almacen) error {
	path, err := d.RutaAlmacen()
	if err != nil {
		return err
	}
	alm.PorFicheiro = d.podarPorFicheiro(alm.PorFicheiro)
	alm.MarcarVersion() // deixa o ficheiro coa versión de formato actual
	b, err := json.MarshalIndent(alm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// podarPorFicheiro quita as entradas de documentos que xa non existen no disco
// - sen isto, a memoria "que plantilla usaba cada ficheiro" medraría para
// sempre con ficheiros renomeados ou borrados hai anos.
func (d *Deposito) podarPorFicheiro(m map[string]string) map[string]string {
	if len(m) == 0 || d.ExisteFicheiro == nil {
		return m
	}
	out := make(map[string]string, len(m))
	for ruta, id := range m {
		if d.ExisteFicheiro(ruta) {
			out[ruta] = id
		}
	}
	return out
}

// Imaxes le o cartafol de imaxes da plantilla. Un cartafol que non existe (o
// normal: aínda non se subiu ningunha) non é erro.
func (d *Deposito) Imaxes(id string) []string {
	dir, err := d.DirImaxes(id)
	if err != nil {
		return nil
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var nomes []string
	for _, e := range entradas {
		if !e.IsDir() {
			nomes = append(nomes, e.Name())
		}
	}
	sort.Strings(nomes)
	return nomes
}

// CopiarImaxes copia o cartafol de imaxes dunha plantilla a outra (ao duplicar
// ou ao bifurcar unha de fábrica). Unha orixe sen cartafol non é erro.
func (d *Deposito) CopiarImaxes(orixeID, destinoID string) error {
	orixe, err := d.DirImaxes(orixeID)
	if err != nil {
		return err
	}
	entradas, err := os.ReadDir(orixe)
	if err != nil {
		return nil // sen imaxes que copiar
	}
	destino, err := d.DirImaxes(destinoID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destino, 0o755); err != nil {
		return err
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(orixe, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destino, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// BorrarImaxes retira o cartafol de imaxes dunha plantilla (ao eliminarla).
func (d *Deposito) BorrarImaxes(id string) error {
	if id == "" {
		return nil
	}
	dir, err := d.DirImaxes(id)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// Estado devolve a biblioteca ordenada por nome, con `imaxes` xa resolto, e a
// plantilla activa: é o que precisa a barra de ferramentas do frontend.
func (d *Deposito) Estado() (Estado, error) {
	alm, err := d.Ler()
	if err != nil {
		return Estado{}, err
	}
	items := alm.Plantillas
	for i := range items {
		items[i].Imaxes = d.Imaxes(items[i].ID)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Nome) < strings.ToLower(items[j].Nome)
	})
	if items == nil {
		items = []Plantilla{}
	}
	return Estado{Plantillas: items, Activa: alm.Activa}, nil
}

// Meter crea (ID baleira) ou actualiza unha plantilla no almacén e devolve a
// gardada coas súas imaxes resoltas.
//
// Regras (as mesmas de sempre, agora nun só sitio):
//   - gardar por riba dunha plantilla DE FÁBRICA non a pisa: bifúrcase;
//   - unha ID que xa non existe (borrada noutra xanela) gárdase coma nova, en
//     vez de perder o traballo do profesorado.
func (d *Deposito) Meter(p Plantilla) (Plantilla, error) {
	alm, err := d.Ler()
	if err != nil {
		return Plantilla{}, err
	}
	if strings.TrimSpace(p.ID) == "" {
		p.ID = novaID()
		p.Creado = agora()
		p.Modificado = p.Creado
		alm.Plantillas = append(alm.Plantillas, p)
	} else {
		atopada := false
		for i := range alm.Plantillas {
			if alm.Plantillas[i].ID != p.ID {
				continue
			}
			// Gardar por riba dunha DE FÁBRICA non a pisa: bifúrcase.
			if alm.Plantillas[i].DeFabrica {
				return d.bifurcarDeFabrica(alm, p)
			}
			p.Creado = alm.Plantillas[i].Creado
			p.DeFabrica = alm.Plantillas[i].DeFabrica
			p.Modificado = agora()
			alm.Plantillas[i] = p
			atopada = true
			break
		}
		if !atopada {
			p.Creado = agora()
			p.Modificado = p.Creado
			alm.Plantillas = append(alm.Plantillas, p)
		}
	}
	if err := d.Gardar(alm); err != nil {
		return Plantilla{}, err
	}
	p.Imaxes = d.Imaxes(p.ID)
	return p, nil
}

// Eliminar bórraa xunto co seu cartafol de imaxes. Se era a activa, queda
// "ningunha" - senón as xeracións seguintes fallarían apuntando a unha
// plantilla que xa non está.
func (d *Deposito) Eliminar(id string) error {
	alm, err := d.Ler()
	if err != nil {
		return err
	}
	out := alm.Plantillas[:0]
	for _, p := range alm.Plantillas {
		if p.ID != id {
			out = append(out, p)
			continue
		}
		// Anotar as de fábrica borradas para que Sementar non as volva dar de
		// alta no seguinte arranque.
		if p.DeFabrica {
			alm.FabricaBorradas = append(alm.FabricaBorradas, p.ID)
		}
	}
	alm.Plantillas = out
	if alm.Activa == id {
		alm.Activa = ""
	}
	for ruta, usada := range alm.PorFicheiro {
		if usada == id {
			delete(alm.PorFicheiro, ruta)
		}
	}
	if err := d.Gardar(alm); err != nil {
		return err
	}
	return d.BorrarImaxes(id)
}

// Sementar dá de alta as plantillas de fábrica que FALTEN, comparando por ID -
// non só a primeira vez. Iso é o que permite REPARTIR plantillas: unha versión
// nova que engada unha plantilla de fábrica chégalle tamén a quen xa tiña a
// aplicación instalada, sen ter que borrar a súa configuración.
//
// Dúas regras que fan que iso sexa seguro:
//   - unha ID que xa existe NON se toca nunca (o profesorado pode ter editado
//     esa plantilla: pisarlla nunha actualización sería perder o seu traballo
//     sen avisar),
//   - unha que se borrou a propósito non revive (FabricaBorradas).
func (d *Deposito) Sementar(alm Almacen) (Almacen, error) {
	xaEstan := make(map[string]bool, len(alm.Plantillas))
	for _, p := range alm.Plantillas {
		xaEstan[p.ID] = true
	}
	borradas := make(map[string]bool, len(alm.FabricaBorradas))
	for _, id := range alm.FabricaBorradas {
		borradas[id] = true
	}

	var novas []Plantilla
	if d.Fabrica != nil {
		for _, p := range d.Fabrica() {
			if xaEstan[p.ID] || borradas[p.ID] {
				continue
			}
			novas = append(novas, p)
		}
	}
	if len(novas) == 0 && alm.Sementado {
		return alm, nil
	}
	alm.Sementado = true
	alm.Plantillas = append(alm.Plantillas, novas...)
	if err := d.Gardar(alm); err != nil {
		return alm, err
	}
	return alm, nil
}

// RetirarFabricaObsoleta limpa da biblioteca as plantillas de exemplo vellas
// (FabricaObsoleta) sen levar por diante nada que fose de alguén: unha que
// segue co LaTeX orixinal e sen imaxes bórrase; unha que ten cambios ou imaxes
// (por exemplo, un logotipo subido) queda, só que xa non coma "de exemplo",
// para que non a volva pisar unha actualización. Tamén se limpan as referencias
// (activa, memoria por ficheiro) ás borradas.
func (d *Deposito) RetirarFabricaObsoleta(alm Almacen) (Almacen, error) {
	if len(d.FabricaObsoleta) == 0 {
		return alm, nil
	}
	cambiou := false
	quedan := alm.Plantillas[:0:0]
	borradas := map[string]bool{}
	for _, p := range alm.Plantillas {
		hashOrixinal, obsoleta := d.FabricaObsoleta[p.ID]
		if !obsoleta || !p.DeFabrica {
			quedan = append(quedan, p)
			continue
		}
		cambiou = true
		soma := sha256.Sum256([]byte(p.Latex))
		intacta := hex.EncodeToString(soma[:])[:len(hashOrixinal)] == hashOrixinal && len(d.Imaxes(p.ID)) == 0
		if !intacta {
			p.DeFabrica = false
			quedan = append(quedan, p)
			continue
		}
		borradas[p.ID] = true
		_ = d.BorrarImaxes(p.ID)
	}
	if !cambiou {
		return alm, nil
	}
	alm.Plantillas = quedan
	if borradas[alm.Activa] {
		alm.Activa = ""
	}
	for ruta, id := range alm.PorFicheiro {
		if borradas[id] {
			delete(alm.PorFicheiro, ruta)
		}
	}
	return alm, d.Gardar(alm)
}

// RepararFabrica arranxa, unha soa vez e por equipo, o estropicio que deixaban
// as versións anteriores a bifurcarDeFabrica: alí gardar por riba dunha
// plantilla de fábrica pisábaa EN SITIO, deixando unha entrada marcada
// DeFabrica pero con contido do profesorado. Iso tiña dous efectos malos que
// non se ven ata que é tarde:
//
//   - a ID de fábrica quedaba ocupada, e Sementar (que só repón as que FALTAN)
//     xa nunca lle daba as melloras dunha versión nova,
//   - datos privados (o IBAN dunha empresa, o nome dun centro) quedaban baixo
//     unha entrada que a interface segue a amosar coma "de exemplo", cun risco
//     evidente en canto alguén comparte o seu plantillas.json.
//
// A reparación é a mesma operación que fai agora bifurcarDeFabrica, pero
// aplicada ao que xa está gardado: o contido editado pasa a unha plantilla
// propia (ID nova, DeFabrica=false) que herda imaxes, escolla activa e
// documentos, e a de fábrica recupera o seu contido orixinal. Non se perde
// nada e non hai que pedirlle nada ao profesorado.
//
// Detección por comparación co orixinal (non por unha marca no ficheiro): así
// funciona tamén nun plantillas.json que veña doutro equipo, e unha plantilla
// de fábrica que nunca se tocou non se move do sitio.
func (d *Deposito) RepararFabrica(alm Almacen) (Almacen, error) {
	if d.Fabrica == nil {
		return alm, nil
	}
	orixinais := make(map[string]Plantilla)
	for _, p := range d.Fabrica() {
		orixinais[p.ID] = p
	}
	agora := agora()
	cambiou := false
	// Percórrese por índice: bifurcar engade ao final, e o engadido non hai
	// que volvelo examinar (xa non é DeFabrica).
	for i := 0; i < len(alm.Plantillas); i++ {
		actual := alm.Plantillas[i]
		orixinal, ok := orixinais[actual.ID]
		if !ok || !actual.DeFabrica || !d.FabricaFoiEditada(actual, orixinal) {
			continue
		}
		// A de fábrica volve ao seu contido; o editado vaise a unha copia.
		orixinal.Creado, orixinal.Modificado = actual.Creado, agora
		alm.Plantillas[i] = orixinal
		var err error
		if alm, err = d.BifurcarEnSitio(alm, actual, agora); err != nil {
			return alm, err
		}
		cambiou = true
	}
	if !cambiou {
		return alm, nil
	}
	return alm, d.Gardar(alm)
}

// FabricaFoiEditada compara só o que o profesorado pode cambiar desde o editor
// de plantillas. Creado/Modificado/Imaxes quedan fóra a propósito: son
// metadatos que a aplicación toca por conta propia (Imaxes recalcúlase en cada
// listaxe) e dispararían falsos positivos.
func (d *Deposito) FabricaFoiEditada(actual, orixinal Plantilla) bool {
	textosIguais := d.TextosIguais
	if textosIguais == nil {
		textosIguais = func(a, b map[string]string) bool { return reflect.DeepEqual(a, b) }
	}
	if actual.Nome != orixinal.Nome || actual.Descricion != orixinal.Descricion ||
		actual.Latex != orixinal.Latex || actual.Markdown != orixinal.Markdown ||
		actual.Portada != orixinal.Portada || actual.Sinatura != orixinal.Sinatura ||
		actual.Introducion != orixinal.Introducion || actual.Ofimatica != orixinal.Ofimatica ||
		actual.Estilo != orixinal.Estilo || !textosIguais(actual.Textos, orixinal.Textos) ||
		actual.Motor != orixinal.Motor || len(actual.Variables) != len(orixinal.Variables) ||
		!reflect.DeepEqual(actual.Deseno, orixinal.Deseno) {
		return true
	}
	for i := range actual.Variables {
		if actual.Variables[i] != orixinal.Variables[i] {
			return true
		}
	}
	return false
}

// BifurcarEnSitio é o núcleo compartido por bifurcarDeFabrica (ao gardar) e por
// RepararFabrica (ao cargar), e mais polo asistente cando quere gardar imaxes
// nunha plantilla de exemplo: mete `p` coma plantilla propia nova e traspásalle
// todo o que colgaba da de fábrica. NON garda - déixao a quen chama, que sabe
// se hai máis cambios que escribir na mesma pasada.
func (d *Deposito) BifurcarEnSitio(alm Almacen, p Plantilla, instante string) (Almacen, error) {
	orixeID := p.ID
	p.ID = novaID()
	p.DeFabrica = false
	p.Creado = instante
	p.Modificado = instante
	alm.Plantillas = append(alm.Plantillas, p)

	if alm.Activa == orixeID {
		alm.Activa = p.ID
	}
	for ruta, id := range alm.PorFicheiro {
		if id == orixeID {
			alm.PorFicheiro[ruta] = p.ID
		}
	}
	return alm, d.CopiarImaxes(orixeID, p.ID)
}

// bifurcarDeFabrica: editar unha plantilla DE FÁBRICA e gardala non a
// sobrescribe - créase unha copia propia (ID novo, DeFabrica=false) co que o
// profesorado acaba de escribir, e a de fábrica queda intacta ao lado.
//
// A bifurcación herda TODO o que a de fábrica tiña asociado - as imaxes, ser a
// plantilla activa e os documentos que a usaban - a propósito: de cara ao
// profesorado, gardar ten que seguir facendo o de sempre (o meu traballo
// aplícase aos meus documentos). O único que cambia é que a de fábrica
// sobrevive, limpa, en vez de desaparecer baixo os cambios.
func (d *Deposito) bifurcarDeFabrica(alm Almacen, p Plantilla) (Plantilla, error) {
	alm, errImaxes := d.BifurcarEnSitio(alm, p, agora())
	nova := alm.Plantillas[len(alm.Plantillas)-1]
	if err := d.Gardar(alm); err != nil {
		return Plantilla{}, err
	}
	// O erro de copiar imaxes devólvese DESPOIS de gardar, non antes: se falla,
	// a bifurcación xa existe e o texto (o caro de reescribir) está a salvo;
	// só faltarían as imaxes, que se poden volver subir.
	if errImaxes != nil {
		return nova, errImaxes
	}
	nova.Imaxes = d.Imaxes(nova.ID)
	return nova, nil
}

// Activar marca a plantilla activa e, se se indica un ficheiro, lembra a
// escolla para ese documento. Un id baleiro significa "ningunha".
func (d *Deposito) Activar(id, ficheiro string) error {
	alm, err := d.Ler()
	if err != nil {
		return err
	}
	alm.Activa = id
	if ruta := strings.TrimSpace(ficheiro); ruta != "" {
		if alm.PorFicheiro == nil {
			alm.PorFicheiro = map[string]string{}
		}
		if id == "" {
			delete(alm.PorFicheiro, ruta)
		} else {
			alm.PorFicheiro[ruta] = id
		}
	}
	return d.Gardar(alm)
}

// ParaFicheiro devolve (e activa) a plantilla lembrada para ese documento ao
// abrilo. Se ese ficheiro nunca tivo plantilla, mantense a activa actual:
// cambiar de documento non debe desfacer a escolla que acaba de facer o
// profesorado para traballar. A plantilla lembrada pode ter sido borrada
// mentres tanto, e entón tamén se mantén a activa.
func (d *Deposito) ParaFicheiro(ficheiro string) (string, error) {
	alm, err := d.Ler()
	if err != nil {
		return "", err
	}
	ruta := strings.TrimSpace(ficheiro)
	if ruta == "" {
		return alm.Activa, nil
	}
	id, ok := alm.PorFicheiro[ruta]
	if !ok {
		return alm.Activa, nil
	}
	existe := false
	for _, p := range alm.Plantillas {
		if p.ID == id {
			existe = true
			break
		}
	}
	if !existe {
		return alm.Activa, nil
	}
	if alm.Activa != id {
		alm.Activa = id
		if err := d.Gardar(alm); err != nil {
			return "", err
		}
	}
	return id, nil
}

// Activa resolve a plantilla que hai que aplicar agora mesmo, ou nil se non hai
// ningunha (comportamento de sempre). Nunca devolve erro "brando": se o
// ficheiro está corrupto ou a plantilla activa desapareceu, xérase sen
// plantilla en vez de bloquear o traballo.
func (d *Deposito) Activa() *Plantilla {
	alm, err := d.Ler()
	if err != nil || alm.Activa == "" {
		return nil
	}
	for i := range alm.Plantillas {
		if alm.Plantillas[i].ID == alm.Activa {
			p := alm.Plantillas[i]
			return &p
		}
	}
	return nil
}
