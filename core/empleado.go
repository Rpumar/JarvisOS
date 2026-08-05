package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================
// TAREAS
// ============================================================

type Tarea struct {
	ID        int    `json:"id"`
	Nombre    string `json:"nombre"`
	Detalle   string `json:"detalle,omitempty"`
	Fecha     string `json:"fecha,omitempty"`
	Hecha     bool   `json:"hecha"`
	Creada    string `json:"creada"`
	Completada string `json:"completada,omitempty"`
}

type GestorTareas struct {
	mu     sync.Mutex
	ruta   string
	nextID int
	tareas []Tarea
}

func NuevoGestorTareas(ruta string) *GestorTareas {
	g := &GestorTareas{ruta: ruta, nextID: 1}
	g.cargar()
	return g
}

func (g *GestorTareas) cargar() {
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var tareas []Tarea
	if err := json.Unmarshal(datos, &tareas); err != nil {
		return
	}
	g.tareas = tareas
	for _, t := range tareas {
		if t.ID >= g.nextID {
			g.nextID = t.ID + 1
		}
	}
}

func (g *GestorTareas) guardar() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(g.ruta), 0o700); err != nil {
		return
	}
	datos, _ := json.MarshalIndent(g.tareas, "", "  ")
	if err := os.WriteFile(g.ruta, datos, 0o600); err != nil {
		return
	}
}

func (g *GestorTareas) Agregar(nombre, detalle, fecha string) Tarea {
	g.mu.Lock()
	t := Tarea{
		ID:     g.nextID,
		Nombre: nombre,
		Detalle: detalle,
		Fecha:  fecha,
		Hecha:  false,
		Creada: time.Now().Format("2006-01-02 15:04"),
	}
	g.nextID++
	g.tareas = append(g.tareas, t)
	g.mu.Unlock()
	g.guardar()
	return t
}

func (g *GestorTareas) ListarPendientes() []Tarea {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Tarea, 0, len(g.tareas))
	for _, t := range g.tareas {
		if !t.Hecha {
			res = append(res, t)
		}
	}
	return res
}

func (g *GestorTareas) ListarTodas() []Tarea {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Tarea, len(g.tareas))
	copy(res, g.tareas)
	return res
}

func (g *GestorTareas) ContarPendientes() int {
	return len(g.ListarPendientes())
}

// MarcarHecha marca como hecha la primera tarea pendiente que coincida con el
// texto de búsqueda (por número o por nombre). Devuelve un mensaje de resultado.
func (g *GestorTareas) MarcarHecha(busqueda string) string {
	busqueda = strings.ToLower(strings.TrimSpace(busqueda))
	g.mu.Lock()
	idx := -1
	for i := range g.tareas {
		t := &g.tareas[i]
		if t.Hecha {
			continue
		}
		if matchTarea(t, busqueda) {
			idx = i
			break
		}
	}
	if idx == -1 {
		g.mu.Unlock()
		return fmt.Sprintf("No encontré una tarea pendiente que coincida con '%s', señor.", busqueda)
	}
	t := &g.tareas[idx]
	t.Hecha = true
	t.Completada = time.Now().Format("2006-01-02 15:04")
	g.mu.Unlock()
	g.guardar()
	return fmt.Sprintf("Tarea '%s' marcada como hecha, señor.", t.Nombre)
}

// Borrar elimina la primera tarea que coincida con la búsqueda.
func (g *GestorTareas) Borrar(busqueda string) bool {
	busqueda = strings.ToLower(strings.TrimSpace(busqueda))
	g.mu.Lock()
	idx := -1
	for i := range g.tareas {
		if matchTarea(&g.tareas[i], busqueda) {
			idx = i
			break
		}
	}
	if idx == -1 {
		g.mu.Unlock()
		return false
	}
	g.tareas = append(g.tareas[:idx], g.tareas[idx+1:]...)
	g.mu.Unlock()
	g.guardar()
	return true
}

func matchTarea(t *Tarea, busqueda string) bool {
	if busqueda == "" {
		return false
	}
	id := fmt.Sprintf("%d", t.ID)
	if id == busqueda || strings.HasPrefix(busqueda, "#"+id) {
		return true
	}
	nTarea := simplificar(t.Nombre)
	nBusqueda := simplificar(busqueda)
	return strings.Contains(nTarea, nBusqueda) || strings.Contains(nBusqueda, nTarea)
}

func (g *GestorTareas) TextoPendientes() string {
	pendientes := g.ListarPendientes()
	if len(pendientes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tiene %d tareas pendientes: ", len(pendientes)))
	partes := make([]string, 0, len(pendientes))
	for _, t := range pendientes {
		partes = append(partes, fmt.Sprintf("#%d %s", t.ID, t.Nombre))
	}
	b.WriteString(strings.Join(partes, ", "))
	return b.String()
}

// ============================================================
// PROCEDIMIENTOS
// ============================================================

type Procedimiento struct {
	Nombre string   `json:"nombre"`
	Pasos  []string `json:"pasos"`
	Creado string   `json:"creado"`
}

type GestorProcedimientos struct {
	mu    sync.Mutex
	ruta  string
	procs map[string]*Procedimiento
}

func NuevoGestorProcedimientos(ruta string) *GestorProcedimientos {
	m := &GestorProcedimientos{ruta: ruta, procs: make(map[string]*Procedimiento)}
	m.cargar()
	return m
}

func (m *GestorProcedimientos) cargar() {
	datos, err := os.ReadFile(m.ruta)
	if err != nil {
		return
	}
	var lista []Procedimiento
	if err := json.Unmarshal(datos, &lista); err != nil {
		return
	}
	for i := range lista {
		p := lista[i]
		m.procs[p.Nombre] = &p
	}
}

func (m *GestorProcedimientos) guardar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(m.ruta), 0o700); err != nil {
		return
	}
	lista := make([]Procedimiento, 0, len(m.procs))
	for _, p := range m.procs {
		lista = append(lista, *p)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Nombre < lista[j].Nombre })
	datos, _ := json.MarshalIndent(lista, "", "  ")
	if err := os.WriteFile(m.ruta, datos, 0o600); err != nil {
		return
	}
}

func (m *GestorProcedimientos) Crear(nombre string, pasos []string) {
	m.mu.Lock()
	m.procs[nombre] = &Procedimiento{Nombre: nombre, Pasos: pasos, Creado: time.Now().Format("2006-01-02 15:04")}
	m.mu.Unlock()
	m.guardar()
}

func (m *GestorProcedimientos) Borrar(nombre string) bool {
	m.mu.Lock()
	_, existe := m.procs[nombre]
	if existe {
		delete(m.procs, nombre)
	}
	m.mu.Unlock()
	if existe {
		m.guardar()
	}
	return existe
}

func (m *GestorProcedimientos) Obtener(nombre string) (*Procedimiento, bool) {
	m.mu.Lock()
	p, ok := m.procs[nombre]
	m.mu.Unlock()
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func (m *GestorProcedimientos) Listar() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	nombres := make([]string, 0, len(m.procs))
	for n := range m.procs {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	return nombres
}

// Buscar devuelve el procedimiento cuyo nombre coincida con el texto dado.
func (m *GestorProcedimientos) Buscar(texto string) (*Procedimiento, bool) {
	normal := simplificar(texto)
	m.mu.Lock()
	defer m.mu.Unlock()
	for nombre, p := range m.procs {
		if simplificar(nombre) == normal {
			cp := *p
			return &cp, true
		}
		if normal != "" && strings.Contains(normal, simplificar(nombre)) {
			cp := *p
			return &cp, true
		}
	}
	return nil, false
}

// TextoParaIA arma el bloque de instrucciones con los procedimientos que
// aplican al pedido, para que la IA recuerde cómo se hacen las tareas.
func (m *GestorProcedimientos) TextoParaIA(entrada string) string {
	m.mu.Lock()
	if len(m.procs) == 0 {
		m.mu.Unlock()
		return ""
	}
	normal := simplificar(entrada)
	coincidencias := make([]*Procedimiento, 0, 1)
	for nombre, p := range m.procs {
		if normal != "" && strings.Contains(normal, simplificar(nombre)) {
			coincidencias = append(coincidencias, p)
		}
	}
	m.mu.Unlock()

	if len(coincidencias) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Recuerdos de trabajo (procedimientos ya aprendidos):\n")
	for _, p := range coincidencias {
		b.WriteString(fmt.Sprintf("- Para '%s' seguí estos pasos:\n", p.Nombre))
		for i, paso := range p.Pasos {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, paso))
		}
	}
	b.WriteString("Si el pedido del usuario coincide con un procedimiento, usalo.\n")
	return b.String()
}

// ============================================================
// COMANDOS DE VOZ: TAREAS
// ============================================================

func (h *Hands) manejarTarea(cmd string) string {
	entrada := strings.ToLower(strings.TrimSpace(cmd))

	switch {
	case strings.Contains(entrada, "qué tareas tengo") || strings.Contains(entrada, "que tareas tengo"),
		strings.Contains(entrada, "tareas pendientes") || strings.Contains(entrada, "tareas me faltan"),
		strings.Contains(entrada, "mis tareas") || strings.Contains(entrada, "mis tareas pendientes"),
		strings.Contains(entrada, "listá mis tareas") || strings.Contains(entrada, "lista mis tareas"),
		strings.Contains(entrada, "cuántas tareas tengo") || strings.Contains(entrada, "cuantas tareas tengo"):
		return h.listarTareas(false)

	case strings.Contains(entrada, "todas las tareas") || strings.Contains(entrada, "todas mis tareas"),
		strings.Contains(entrada, "historial de tareas"):
		return h.listarTareas(true)

	case strings.Contains(entrada, "borrar tarea") || strings.Contains(entrada, "borrá la tarea"),
		strings.Contains(entrada, "borra la tarea"), strings.Contains(entrada, "eliminar tarea"),
		strings.Contains(entrada, "eliminá la tarea"), strings.Contains(entrada, "elimina la tarea"):
		busqueda := extraerBusquedaTarea(entrada, []string{"borrar tarea", "borrá la tarea", "borra la tarea", "eliminar tarea", "eliminá la tarea", "elimina la tarea"})
		if busqueda == "" {
			return "¿Qué tarea quiere borrar, señor? Diga 'borrar tarea #1' o 'borrar tarea [nombre]'."
		}
		if !h.tareas.Borrar(busqueda) {
			return fmt.Sprintf("No encontré una tarea que coincida con '%s', señor.", busqueda)
		}
		return fmt.Sprintf("Tarea '%s' eliminada, señor.", busqueda)

	case strings.Contains(entrada, "marcar tarea") || strings.Contains(entrada, "marca la tarea"),
		strings.Contains(entrada, "marqué la tarea"), strings.Contains(entrada, "tarea como hecha"),
		strings.Contains(entrada, "tarea como terminada"), strings.Contains(entrada, "completé la tarea"),
		strings.Contains(entrada, "completa la tarea"), strings.Contains(entrada, "completá la tarea"),
		strings.Contains(entrada, "terminé la tarea"), strings.Contains(entrada, "termine la tarea"):
		busqueda := extraerBusquedaTarea(entrada, []string{"marcar tarea", "marca la tarea", "marqué la tarea", "marcar la tarea", "tarea como hecha", "tarea como terminada", "completé la tarea", "completa la tarea", "completá la tarea", "terminé la tarea", "termine la tarea"})
		busqueda = strings.ReplaceAll(busqueda, " como hecha", "")
		busqueda = strings.ReplaceAll(busqueda, " como terminada", "")
		busqueda = strings.TrimSpace(busqueda)
		if busqueda == "" {
			return "¿Qué tarea marcó como hecha, señor? Diga 'marcar tarea #1 como hecha'."
		}
		return h.tareas.MarcarHecha(busqueda)

	case strings.Contains(entrada, "agendá"), strings.Contains(entrada, "agenda"),
		strings.Contains(entrada, "agendame"), strings.Contains(entrada, "agendáme"),
		strings.Contains(entrada, "registrá la tarea"), strings.Contains(entrada, "registra la tarea"),
		strings.Contains(entrada, "registrá una tarea"), strings.Contains(entrada, "registra una tarea"),
		strings.Contains(entrada, "creá una tarea"), strings.Contains(entrada, "crea una tarea"),
		strings.Contains(entrada, "creame una tarea"),
		strings.Contains(entrada, "tarea nueva"), strings.Contains(entrada, "nueva tarea"),
		strings.Contains(entrada, "añadí la tarea"), strings.Contains(entrada, "añadi la tarea"):
		return h.agendarTarea(entrada, cmd)

	default:
		return "¿Qué tarea desea agendar, señor? Diga 'agendá una tarea enviar el informe para mañana'."
	}
}

func (h *Hands) agendarTarea(entrada, original string) string {
	prefijos := []string{
		"agendá una tarea", "agenda una tarea", "agendá la tarea", "agenda la tarea",
		"agendame una tarea", "agendáme una tarea", "agendame la tarea", "agendáme la tarea",
		"agendame", "agendáme", "agendá", "agenda",
		"registrá la tarea", "registra la tarea", "registrá una tarea", "registra una tarea",
		"creá una tarea", "crea una tarea", "creame una tarea",
		"tarea nueva", "nueva tarea", "añadí la tarea", "añadi la tarea",
	}
	resto := extraerBusquedaTarea(entrada, prefijos)
	resto = strings.TrimSpace(resto)
	if resto == "" {
		return "¿Qué tarea desea agendar, señor?"
	}
	if strings.HasPrefix(resto, "que ") || strings.HasPrefix(resto, "que:") {
		resto = strings.TrimSpace(strings.TrimPrefix(resto, "que "))
		resto = strings.TrimSpace(strings.TrimPrefix(resto, "que:"))
	}

	nombre, fecha, detalle := separarTarea(resto)
	if nombre == "" {
		return "No entendí el nombre de la tarea, señor."
	}
	if h.tareas != nil {
		h.tareas.Agregar(nombre, detalle, fecha)
	}
	msg := fmt.Sprintf("Tarea agendada: '%s', señor.", nombre)
	if fecha != "" {
		msg = fmt.Sprintf("Tarea agendada: '%s' para %s, señor.", nombre, fecha)
	}
	if detalle != "" {
		msg += fmt.Sprintf(" Detalle: %s.", detalle)
	}
	return msg
}

func separarTarea(resto string) (nombre, fecha, detalle string) {
	resto = strings.TrimSpace(resto)
	if i := strings.Index(resto, " para "); i >= 0 {
		nombre = strings.TrimSpace(resto[:i])
		despues := strings.TrimSpace(resto[i+len(" para "):])
		if j := strings.Index(despues, " que "); j >= 0 {
			fecha = strings.TrimSpace(despues[:j])
			detalle = strings.TrimSpace(despues[j+len(" que "):])
		} else {
			fecha = despues
		}
		return
	}
	if i := strings.Index(resto, " que "); i >= 0 {
		nombre = strings.TrimSpace(resto[:i])
		detalle = strings.TrimSpace(resto[i+len(" que "):])
		return
	}
	nombre = resto
	return
}

func extraerBusquedaTarea(entrada string, prefijos []string) string {
	for _, p := range prefijos {
		if strings.Contains(entrada, p) {
			idx := strings.Index(entrada, p)
			resto := strings.TrimSpace(entrada[idx+len(p):])
			resto = strings.TrimPrefix(resto, "la ")
			resto = strings.TrimPrefix(resto, "una ")
			return strings.TrimSpace(resto)
		}
	}
	return ""
}

func (h *Hands) listarTareas(todas bool) string {
	if h.tareas == nil {
		return "No tengo acceso a las tareas, señor."
	}
	if todas {
		todasTareas := h.tareas.ListarTodas()
		if len(todasTareas) == 0 {
			return "No tiene tareas registradas, señor. Diga 'agendá una tarea X para mañana' para crear una."
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Tiene %d tareas en total, señor:", len(todasTareas)))
		for _, t := range todasTareas {
			estado := "pendiente"
			if t.Hecha {
				estado = "hecha"
			}
			b.WriteString(fmt.Sprintf("\n  #%d %s (%s)", t.ID, t.Nombre, estado))
		}
		return b.String()
	}
	pendientes := h.tareas.ListarPendientes()
	if len(pendientes) == 0 {
		return "No tiene tareas pendientes, señor. Todo al día."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tiene %d tareas pendientes, señor:", len(pendientes)))
	for _, t := range pendientes {
		b.WriteString(fmt.Sprintf("\n  #%d %s", t.ID, t.Nombre))
		if t.Fecha != "" {
			b.WriteString(fmt.Sprintf(" (para %s)", t.Fecha))
		}
	}
	return b.String()
}

// ============================================================
// COMANDOS DE VOZ: PROCEDIMIENTOS
// ============================================================

func (h *Hands) manejarProcedimiento(cmd string) string {
	entrada := strings.ToLower(strings.TrimSpace(cmd))

	if h.procedimientoPendiente != "" && !empiezaAccionNueva(entrada) {
		pasos := dividirPasos(limpiarPasos(cmd))
		nombre := h.procedimientoPendiente
		h.procedimientoPendiente = ""
		if len(pasos) == 0 {
			return fmt.Sprintf("No entendí los pasos de '%s', señor. Intente: 'los pasos son abrir chrome, escribir hola y cerrar chrome'.", nombre)
		}
		if h.procedimientos != nil {
			h.procedimientos.Crear(nombre, pasos)
		}
		return fmt.Sprintf("Aprendido, señor. Ya sé cómo %s (%d pasos).", nombre, len(pasos))
	}

	switch {
	case strings.Contains(entrada, "qué procedimientos sabés") || strings.Contains(entrada, "que procedimientos sabes"),
		strings.Contains(entrada, "qué sabés hacer") || strings.Contains(entrada, "que sabes hacer"),
		strings.Contains(entrada, "qué procedimientos tenés") || strings.Contains(entrada, "que procedimientos tenes"),
		strings.Contains(entrada, "qué aprendiste") || strings.Contains(entrada, "que aprendiste"):
		nombres := h.procedimientos.Listar()
		if len(nombres) == 0 {
			return "Todavía no aprendí ningún procedimiento, señor. Diga 'aprendé que para hacer [tarea]: paso 1, paso 2' y lo incorporaré."
		}
		return fmt.Sprintf("Sé hacer esto, señor: %s.", strings.Join(nombres, ", "))

	case strings.Contains(entrada, "olvidate el procedimiento") || strings.Contains(entrada, "olvida el procedimiento"),
		strings.Contains(entrada, "olvidá el procedimiento"), strings.Contains(entrada, "olvida el proceso"),
		strings.Contains(entrada, "borrá el procedimiento"), strings.Contains(entrada, "borrar procedimiento"),
		strings.Contains(entrada, "borrá el proceso"), strings.Contains(entrada, "borrar proceso"):
		nombre := strings.TrimSpace(extraerBusquedaTarea(entrada, []string{"olvidate el procedimiento", "olvida el procedimiento", "olvidá el procedimiento", "olvida el proceso", "borrá el procedimiento", "borrar procedimiento", "borrá el proceso", "borrar proceso"}))
		if nombre == "" {
			return "¿Qué procedimiento quiere que olvide, señor?"
		}
		if !h.procedimientos.Borrar(nombre) {
			return fmt.Sprintf("No tengo un procedimiento llamado '%s', señor.", nombre)
		}
		return fmt.Sprintf("Procedimiento '%s' olvidado, señor.", nombre)

	case strings.Contains(entrada, "aprendé a hacer"), strings.Contains(entrada, "aprende a hacer"),
		strings.Contains(entrada, "aprendé a"), strings.Contains(entrada, "aprende a"),
		strings.Contains(entrada, "aprendé que para"), strings.Contains(entrada, "aprende que para"),
		strings.Contains(entrada, "aprendé que"), strings.Contains(entrada, "aprende que"),
		strings.Contains(entrada, "recordá que para"), strings.Contains(entrada, "recuerda que para"),
		strings.Contains(entrada, "enseñame a"), strings.Contains(entrada, "enseñáme a"), strings.Contains(entrada, "ensename a"):
		return h.aprenderProcedimiento(entrada)

	case strings.Contains(entrada, "cómo hago"), strings.Contains(entrada, "como hago"),
		strings.Contains(entrada, "cómo se hace"), strings.Contains(entrada, "como se hace"),
		strings.Contains(entrada, "cómo haces"), strings.Contains(entrada, "como haces"),
		strings.Contains(entrada, "ejecutá el procedimiento"), strings.Contains(entrada, "ejecuta el procedimiento"),
		strings.Contains(entrada, "ejecutá el proceso"), strings.Contains(entrada, "ejecuta el proceso"),
		strings.Contains(entrada, "hacé el procedimiento"), strings.Contains(entrada, "hace el procedimiento"):
		nombre := extraerBusquedaTarea(entrada, []string{"cómo hago", "como hago", "cómo se hace", "como se hace", "cómo haces", "como haces", "ejecutá el procedimiento", "ejecuta el procedimiento", "ejecutá el proceso", "ejecuta el proceso", "hacé el procedimiento", "hace el procedimiento"})
		nombre = strings.TrimSpace(nombre)
		if nombre == "" {
			nombres := h.procedimientos.Listar()
			if len(nombres) == 0 {
				return "Todavía no aprendí ningún procedimiento, señor."
			}
			return fmt.Sprintf("Sé hacer esto, señor: %s. Diga 'ejecutá el procedimiento' seguido del nombre.", strings.Join(nombres, ", "))
		}
		return h.ejecutarProcedimiento(nombre)

	default:
		if p, ok := h.procedimientos.Buscar(entrada); ok {
			return h.ejecutarProcedimiento(p.Nombre)
		}
		if esPedidoDeTrabajo(entrada) {
			return "No sé cómo hacer eso todavía, señor. Enséñeme: 'aprendé que para hacer [eso]: paso 1, paso 2' y lo incorporaré al instante."
		}
		return ComandoNoReconocido
	}
}

func (h *Hands) aprenderProcedimiento(entrada string) string {
	prefijos := []string{
		"aprendé que para hacer", "aprende que para hacer", "aprendé que para", "aprende que para",
		"aprendé a hacer", "aprende a hacer", "aprendé que", "aprende que",
		"recordá que para", "recuerda que para", "enseñame a", "enseñáme a", "ensename a",
	}
	resto := extraerBusquedaTarea(entrada, prefijos)
	resto = strings.TrimSpace(resto)
	if resto == "" {
		return "¿Qué quiere enseñarme, señor? Ejemplo: 'aprendé que para hacer el informe: abrir word, escribir el resumen y guardar'."
	}
	resto = strings.TrimPrefix(resto, "a hacer ")
	resto = strings.TrimSpace(resto)

	nombre := ""
	pasos := []string{}
	if i := strings.Index(resto, ":"); i >= 0 {
		nombre = strings.TrimSpace(resto[:i])
		acciones := strings.TrimSpace(resto[i+1:])
		if acciones != "" {
			pasos = dividirPasos(acciones)
		}
	}
	if nombre == "" && len(pasos) == 0 {
		if i := strings.Index(resto, " haciendo "); i >= 0 {
			nombre = strings.TrimSpace(resto[:i])
			pasos = dividirPasos(strings.TrimSpace(resto[i+len(" haciendo "):]))
		}
	}
	if nombre == "" && len(pasos) == 0 {
		nombre = resto
	}
	if nombre == "" {
		return "Dígame el nombre de la tarea a aprender, señor."
	}

	if len(pasos) == 0 {
		h.procedimientoPendiente = nombre
		return fmt.Sprintf("Entendido, señor. Voy a aprender '%s'. ¿Cuáles son los pasos? Diga 'los pasos son: paso 1, paso 2, paso 3'.", nombre)
	}

	if h.procedimientos != nil {
		h.procedimientos.Crear(nombre, pasos)
	}
	return fmt.Sprintf("Aprendido, señor. Ya sé cómo %s (%d pasos).", nombre, len(pasos))
}

func (h *Hands) ejecutarProcedimiento(nombre string) string {
	p, ok := h.procedimientos.Obtener(nombre)
	if !ok {
		nombres := h.procedimientos.Listar()
		if len(nombres) == 0 {
			return fmt.Sprintf("No tengo un procedimiento llamado '%s', señor. Enséñemelo: 'aprendé que para hacer %s: paso 1, paso 2'.", nombre, nombre)
		}
		return fmt.Sprintf("No tengo un procedimiento llamado '%s', señor. Sé hacer: %s.", nombre, strings.Join(nombres, ", "))
	}
	for _, paso := range p.Pasos {
		_ = h.RunCommand(paso)
	}
	return fmt.Sprintf("Procedimiento '%s' ejecutado con %d pasos, señor.", nombre, len(p.Pasos))
}

func limpiarPasos(cmd string) string {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	cmd = strings.ReplaceAll(cmd, "los pasos son: ", "")
	cmd = strings.ReplaceAll(cmd, "los pasos son ", "")
	cmd = strings.ReplaceAll(cmd, "los pasos son", "")
	cmd = strings.ReplaceAll(cmd, "los pasos: ", "")
	cmd = strings.ReplaceAll(cmd, "los pasos ", "")
	cmd = strings.ReplaceAll(cmd, "los pasos", "")
	cmd = strings.ReplaceAll(cmd, "pasos son: ", "")
	cmd = strings.ReplaceAll(cmd, "pasos son ", "")
	cmd = strings.ReplaceAll(cmd, "son ", "")
	return strings.TrimSpace(cmd)
}

func empiezaAccionNueva(entrada string) bool {
	for _, prefijo := range []string{"aprendé", "aprende", "ejecutá el procedimiento", "ejecuta el procedimiento", "cómo hago", "como hago", "borrá el procedimiento", "borrar procedimiento", "olvidate el procedimiento", "qué procedimientos", "que procedimientos", "qué sabés", "que sabes"} {
		if strings.HasPrefix(entrada, prefijo) {
			return true
		}
	}
	return false
}

func esPedidoDeTrabajo(entrada string) bool {
	verbos := []string{
		"hacé", "hace ", "hacéme", "hacerme", "hazme", "armá", "arma ", "creá", "crea ", "generá", "genera ",
		"escribí", "escribi", "escribíme", "escribime", "redactá", "redacta ", "prepará", "prepara ",
		"enviá", "envia ", "enviáme", "envíame", "buscá", "busca ", "organizá", "organiza ",
		"descargá", "descarga ", "instalá", "instala ", "configurá", "configura ", "analizá", "analiza ",
		"revisá", "revisa ", "controlá", "controla ", "auditá", "audita ", "documentá", "documenta ",
		"investigá", "investiga ", "calculá", "calcula ", "estimá", "estima ",
	}
	for _, v := range verbos {
		if strings.Contains(entrada, v) {
			return true
		}
	}
	return false
}
