package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ============================================================
// ÓRDENES: el núcleo del "empleado que no abandona su trabajo".
// Una orden no se cierra hasta que se cumple y se reporta (o el
// dueño la cancela). Al arrancar, Jarvis retoma las pendientes.
// ============================================================

const (
	OrdenPendiente           = "pendiente"
	OrdenEnProgreso          = "en_progreso"
	OrdenVerificando         = "verificando"
	OrdenEsperandoAprobacion = "esperando_aprobacion"
	OrdenTerminada           = "terminada"
	OrdenBloqueada           = "bloqueada"
	OrdenCancelada           = "cancelada"
)

type AccionOrden struct {
	Momento   string `json:"momento"`
	Accion    string `json:"accion"`
	Resultado string `json:"resultado"`
}

type Orden struct {
	ID                 int           `json:"id"`
	Objetivo           string        `json:"objetivo"`
	PedidoPor          string        `json:"pedido_por,omitempty"`
	Estado             string        `json:"estado"`
	FechaCreacion      string        `json:"fecha_creacion"`
	Historial          []AccionOrden `json:"historial"`
	Reporte            string        `json:"reporte,omitempty"`
	PendienteAccion    string        `json:"pendiente_accion,omitempty"`
	PendienteDescripcion string      `json:"pendiente_descripcion,omitempty"`
}

type GestorOrdenes struct {
	mu      sync.Mutex
	ruta    string
	nextID  int
	ordenes []Orden
}

func NuevoGestorOrdenes(ruta string) *GestorOrdenes {
	g := &GestorOrdenes{ruta: ruta, nextID: 1}
	g.cargar()
	return g
}

func (g *GestorOrdenes) cargar() {
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var ordenes []Orden
	if err := json.Unmarshal(datos, &ordenes); err != nil {
		return
	}
	g.ordenes = ordenes
	for _, o := range ordenes {
		if o.ID >= g.nextID {
			g.nextID = o.ID + 1
		}
	}
}

func (g *GestorOrdenes) guardar() {
	g.mu.Lock()
	defer g.mu.Unlock()
	os.MkdirAll(filepath.Dir(g.ruta), 0o700)
	datos, _ := json.MarshalIndent(g.ordenes, "", "  ")
	if err := escribirJSONAtomico(g.ruta, datos); err != nil {
		fmt.Printf("[ORDENES] No se pudo guardar: %v\n", err)
	}
}

// Agregar crea una orden en estado pendiente.
func (g *GestorOrdenes) Agregar(objetivo, pedidoPor string) Orden {
	g.mu.Lock()
	o := Orden{
		ID:            g.nextID,
		Objetivo:      objetivo,
		PedidoPor:     pedidoPor,
		Estado:        OrdenPendiente,
		FechaCreacion: time.Now().Format("2006-01-02 15:04"),
		Historial:     []AccionOrden{},
	}
	g.nextID++
	g.ordenes = append(g.ordenes, o)
	g.mu.Unlock()
	g.guardar()
	return o
}

func (g *GestorOrdenes) Obtener(id int) (Orden, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, o := range g.ordenes {
		if o.ID == id {
			return o, true
		}
	}
	return Orden{}, false
}

func (g *GestorOrdenes) Listar() []Orden {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Orden, len(g.ordenes))
	copy(res, g.ordenes)
	return res
}

// Activas devuelve las órdenes que siguen en juego (no terminadas/canceladas).
func (g *GestorOrdenes) Activas() []Orden {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Orden, 0, len(g.ordenes))
	for _, o := range g.ordenes {
		if o.Estado == OrdenTerminada || o.Estado == OrdenCancelada {
			continue
		}
		res = append(res, o)
	}
	return res
}

// Continuables devuelve las órdenes que se pueden retomar (pendientes,
// en progreso o bloqueadas).
func (g *GestorOrdenes) Continuables() []Orden {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Orden, 0, len(g.ordenes))
	for _, o := range g.ordenes {
		if o.Estado == OrdenPendiente || o.Estado == OrdenEnProgreso || o.Estado == OrdenBloqueada {
			res = append(res, o)
		}
	}
	return res
}

// SolicitarAprobacion marca una orden como esperando aprobación y guarda
// la acción sensible que el dueño debe autorizar.
func (g *GestorOrdenes) SolicitarAprobacion(id int, accion, descripcion string) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].Estado = OrdenEsperandoAprobacion
			g.ordenes[i].PendienteAccion = accion
			g.ordenes[i].PendienteDescripcion = descripcion
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// LimpiarAprobacion quita la acción pendiente sin cambiar el estado.
func (g *GestorOrdenes) LimpiarAprobacion(id int) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].PendienteAccion = ""
			g.ordenes[i].PendienteDescripcion = ""
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// DenegarAprobacion vuelve la orden a bloqueada y descarta la acción.
func (g *GestorOrdenes) DenegarAprobacion(id int) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].Estado = OrdenBloqueada
			g.ordenes[i].PendienteAccion = ""
			g.ordenes[i].PendienteDescripcion = ""
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// EsperandoAprobacion devuelve las órdenes que aguardan el PIN del dueño.
func (g *GestorOrdenes) EsperandoAprobacion() []Orden {
	g.mu.Lock()
	defer g.mu.Unlock()
	res := make([]Orden, 0, len(g.ordenes))
	for _, o := range g.ordenes {
		if o.Estado == OrdenEsperandoAprobacion {
			res = append(res, o)
		}
	}
	return res
}

func (g *GestorOrdenes) CambiarEstado(id int, estado string) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].Estado = estado
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// RegistrarAccion agrega una acción al historial de la orden.
func (g *GestorOrdenes) RegistrarAccion(id int, accion, resultado string) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].Historial = append(g.ordenes[i].Historial, AccionOrden{
				Momento:   time.Now().Format("15:04:05"),
				Accion:    accion,
				Resultado: resultado,
			})
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// Terminar cierra la orden con su reporte final (con verificación).
func (g *GestorOrdenes) Terminar(id int, reporte string) bool {
	g.mu.Lock()
	for i := range g.ordenes {
		if g.ordenes[i].ID == id {
			g.ordenes[i].Estado = OrdenTerminada
			g.ordenes[i].Reporte = reporte
			g.mu.Unlock()
			g.guardar()
			return true
		}
	}
	g.mu.Unlock()
	return false
}

// ObtenerMaxID devuelve el mayor ID ya usado (útil para el panel del dueño).
func (g *GestorOrdenes) ObtenerMaxID() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nextID - 1
}

func (g *GestorOrdenes) TextoPendientes() string {	activas := g.Activas()
	if len(activas) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tiene %d órdenes en juego: ", len(activas)))
	partes := make([]string, 0, len(activas))
	for _, o := range activas {
		partes = append(partes, fmt.Sprintf("#%d [%s] %s", o.ID, o.Estado, o.Objetivo))
	}
	b.WriteString(strings.Join(partes, "; "))
	return b.String()
}

// escribirJSONAtomico evita corrupción si el proceso se corta a mitad de
// escritura: escribe a un temporal y renombra al final.
func escribirJSONAtomico(ruta string, datos []byte) error {
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// ============================================================
// COMANDOS DE VOZ: ÓRDENES
// ============================================================

func (h *Hands) manejarOrden(cmd string) string {
	entrada := strings.ToLower(strings.TrimSpace(cmd))

	switch {
	case strings.Contains(entrada, "qué órdenes tengo") || strings.Contains(entrada, "que ordenes tengo"),
		strings.Contains(entrada, "órdenes pendientes") || strings.Contains(entrada, "ordenes pendientes"),
		strings.Contains(entrada, "listá las órdenes") || strings.Contains(entrada, "lista las ordenes"),
		strings.Contains(entrada, "mis órdenes") || strings.Contains(entrada, "mis ordenes"):
		return h.listarOrdenes(false)

	case strings.Contains(entrada, "todas las órdenes") || strings.Contains(entrada, "todas las ordenes"),
		strings.Contains(entrada, "historial de órdenes") || strings.Contains(entrada, "historial de ordenes"):
		return h.listarOrdenes(true)

	case strings.Contains(entrada, "retomá las órdenes") || strings.Contains(entrada, "retoma las ordenes"),
		strings.Contains(entrada, "retomá las ordenes") || strings.Contains(entrada, "continuá con las órdenes"),
		strings.Contains(entrada, "continua con las ordenes"):
		return h.retomarOrdenes()

	case strings.Contains(entrada, "reportá la orden") || strings.Contains(entrada, "reporta la orden"),
		strings.Contains(entrada, "reporte de la orden") || strings.Contains(entrada, "reportá la orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere que reporte, señor? Diga 'reportá la orden #1'."
		}
		return h.reportarOrden(id)

	case strings.Contains(entrada, "marcar la orden") || strings.Contains(entrada, "marca la orden"),
		strings.Contains(entrada, "terminá la orden") || strings.Contains(entrada, "termina la orden"),
		strings.Contains(entrada, "cerrá la orden") || strings.Contains(entrada, "cierra la orden"),
		strings.Contains(entrada, "orden como terminada") || strings.Contains(entrada, "como terminada"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere cerrar, señor? Diga 'terminá la orden #1'."
		}
		if !h.ordenes.Terminar(id, "Cerrada por confirmación del dueño.") {
			return fmt.Sprintf("No encontré la orden #%d, señor.", id)
		}
		return fmt.Sprintf("Orden #%d terminada, señor.", id)

	case strings.Contains(entrada, "bloquear la orden") || strings.Contains(entrada, "bloqueá la orden"),
		strings.Contains(entrada, "bloquea la orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere bloquear, señor? Diga 'bloquear la orden #1'."
		}
		if !h.ordenes.CambiarEstado(id, OrdenBloqueada) {
			return fmt.Sprintf("No encontré la orden #%d, señor.", id)
		}
		return fmt.Sprintf("Orden #%d bloqueada, señor.", id)

	case strings.Contains(entrada, "cancelar la orden") || strings.Contains(entrada, "cancela la orden"),
		strings.Contains(entrada, "cancelá la orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere cancelar, señor? Diga 'cancelar la orden #1'."
		}
		if !h.ordenes.CambiarEstado(id, OrdenCancelada) {
			return fmt.Sprintf("No encontré la orden #%d, señor.", id)
		}
		return fmt.Sprintf("Orden #%d cancelada, señor.", id)

	case strings.Contains(entrada, "aprobar la orden") || strings.Contains(entrada, "aprobar orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere aprobar, señor? Diga 'aprobar la orden #1' (y el PIN si lo configuró)."
		}
		return h.AprobarOrden(id, extraerPIN(entrada))

	case strings.Contains(entrada, "denegar la orden") || strings.Contains(entrada, "denegar orden"),
		strings.Contains(entrada, "rechazar la orden") || strings.Contains(entrada, "rechazar orden"),
		strings.Contains(entrada, "rechazo la orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere rechazar, señor? Diga 'denegar la orden #1'."
		}
		return h.DenegarOrden(id)

	case strings.Contains(entrada, "tomá la orden") || strings.Contains(entrada, "toma la orden"),
		strings.Contains(entrada, "ejecutá la orden") || strings.Contains(entrada, "ejecuta la orden"),
		strings.Contains(entrada, "hacé la orden") || strings.Contains(entrada, "hace la orden"):
		id := extraerID(entrada)
		if id == 0 {
			return "¿Qué orden quiere que ejecute, señor? Diga 'ejecutá la orden #1'."
		}
		return h.procesarOrden(id)

	case strings.Contains(entrada, "agendá una orden") || strings.Contains(entrada, "agenda una orden"),
		strings.Contains(entrada, "nueva orden") || strings.Contains(entrada, "registrá la orden"),
		strings.Contains(entrada, "registra la orden"):
		return h.agendarOrden(entrada)

	default:
		if strings.HasPrefix(entrada, "orden:") || strings.HasPrefix(entrada, "orden ") {
			objetivo := strings.TrimSpace(strings.TrimPrefix(entrada, "orden:"))
			objetivo = strings.TrimSpace(strings.TrimPrefix(objetivo, "orden "))
			if objetivo != "" {
				return h.agendarOrden("nueva orden " + objetivo)
			}
		}
		return ComandoNoReconocido
	}
}

func (h *Hands) agendarOrden(entrada string) string {
	prefijos := []string{"agendá una orden", "agenda una orden", "nueva orden", "registrá la orden", "registra la orden"}
	resto := extraerBusquedaTarea(entrada, prefijos)
	resto = strings.TrimSpace(resto)
	resto = strings.TrimPrefix(resto, "que ")
	if resto == "" {
		return "¿Qué orden desea darme, señor? Diga 'agendá una orden preparar la presentación mensual'."
	}
	if h.ordenes != nil {
		o := h.ordenes.Agregar(resto, "dueño")
		return fmt.Sprintf("Orden #%d registrada: '%s', señor. La cumplo y no la cierro hasta terminarla. Diga 'ejecutá la orden #%d' o 'retomá las órdenes'.", o.ID, o.Objetivo, o.ID)
	}
	return "No tengo acceso al gestor de órdenes, señor."
}

func (h *Hands) listarOrdenes(todas bool) string {
	if h.ordenes == nil {
		return "No tengo acceso al gestor de órdenes, señor."
	}
	if todas {
		todasOrdenes := h.ordenes.Listar()
		if len(todasOrdenes) == 0 {
			return "No hay órdenes registradas, señor. Diga 'agendá una orden [objetivo]'."
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Historial de órdenes (%d), señor:", len(todasOrdenes)))
		for _, o := range todasOrdenes {
			b.WriteString(fmt.Sprintf("\n  #%d [%s] %s (%s)", o.ID, o.Estado, o.Objetivo, o.FechaCreacion))
		}
		return b.String()
	}
	activas := h.ordenes.Activas()
	if len(activas) == 0 {
		return "No tiene órdenes en juego, señor. Todo cumplido."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tiene %d órdenes en juego, señor:", len(activas)))
	for _, o := range activas {
		b.WriteString(fmt.Sprintf("\n  #%d [%s] %s", o.ID, o.Estado, o.Objetivo))
	}
	return b.String()
}

func (h *Hands) retomarOrdenes() string {
	if h.ordenes == nil {
		return "No tengo acceso al gestor de órdenes, señor."
	}
	continuables := h.ordenes.Continuables()
	if len(continuables) == 0 {
		return "No hay órdenes que retomar, señor. Todo en orden."
	}
	hechas := 0
	bloqueadas := 0
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Retomando %d órdenes, señor:", len(continuables)))
	for _, o := range continuables {
		res := h.procesarOrden(o.ID)
		b.WriteString("\n  #")
		b.WriteString(fmt.Sprintf("%d %s", o.ID, o.Objetivo))
		if strings.Contains(res, "cumplida") {
			hechas++
			b.WriteString(" -> cumplida")
		} else {
			bloqueadas++
			b.WriteString(" -> necesita ayuda")
		}
	}
	b.WriteString(fmt.Sprintf("\nResultado: %d cumplidas, %d necesitan ayuda, señor.", hechas, bloqueadas))
	return b.String()
}

func (h *Hands) procesarOrden(id int) string {
	orden, ok := h.ordenes.Obtener(id)
	if !ok {
		return fmt.Sprintf("No encontré la orden #%d, señor.", id)
	}
	if orden.Estado == OrdenTerminada || orden.Estado == OrdenCancelada {
		return fmt.Sprintf("La orden #%d ya está %s, señor.", id, orden.Estado)
	}
	if orden.Estado == OrdenEsperandoAprobacion {
		return fmt.Sprintf("La orden #%d está esperando su aprobación, señor: %s. Apruebe desde el panel, o diga 'aprobar la orden #%d' (o 'aprobar la orden #%d PIN'). Para rechazarla, 'denegar la orden #%d'.", id, orden.PendienteDescripcion, id, id, id)
	}

	h.ordenes.CambiarEstado(id, OrdenEnProgreso)
	h.ordenes.RegistrarAccion(id, "iniciar orden", orden.Objetivo)

	p, conocido := h.procedimientos.Buscar(orden.Objetivo)
	if !conocido {
		h.ordenes.RegistrarAccion(id, "buscar procedimiento", "no se encontró procedimiento")
		if h.IA != nil && h.IA.Disponible() {
			return h.ejecutarOrdenConIA(orden)
		}
		h.ordenes.CambiarEstado(id, OrdenBloqueada)
		return fmt.Sprintf("No sé cómo cumplir la orden '%s', señor. Enséñeme los pasos ('aprendé que para hacer %s: paso 1, paso 2') o active la IA.", orden.Objetivo, orden.Objetivo)
	}

	if h.IA != nil && h.IA.Disponible() {
		// Procedimiento conocido + IA: se ejecuta y la IA verifica
		// si la orden quedó cumplida, ajustando si hace falta.
		prompt := construirPromptAgente(orden.Objetivo)
		ejecutadas := make([]string, 0, len(p.Pasos))
		for _, paso := range p.Pasos {
			r := h.RunCommand(paso)
			h.auditar(id, paso, r)
			h.ordenes.RegistrarAccion(id, paso, r)
			if r != ComandoNoReconocido {
				ejecutadas = append(ejecutadas, paso)
			}
			prompt += fmt.Sprintf("\n[Ejecuté '%s']: %s", paso, r)
		}
		prompt += "\nRevisá si la orden quedó cumplida. Si falta algo, respondé con JSON {\"accion\":\"<comando del catálogo>\",\"razon\":\"<por qué>\"} para completarlo. Si está cumplida, respondé con JSON {\"fin\":\"<resumen del resultado>\"}."
		return h.verificarOrdenConIA(orden, prompt, ejecutadas)
	}

	resultados := 0
	for _, paso := range p.Pasos {
		r := h.RunCommand(paso)
		h.auditar(id, paso, r)
		h.ordenes.RegistrarAccion(id, paso, r)
		if r != ComandoNoReconocido {
			resultados++
		}
	}
	reporte := fmt.Sprintf("Orden '%s' cumplida con %d pasos ejecutados.", orden.Objetivo, resultados)
	h.ordenes.RegistrarAccion(id, "verificar", reporte)
	h.ordenes.Terminar(id, reporte)
	return reporte
}

func (h *Hands) reportarOrden(id int) string {
	orden, ok := h.ordenes.Obtener(id)
	if !ok {
		return fmt.Sprintf("No encontré la orden #%d, señor.", id)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Orden #%d '%s' [%s], señor.", orden.ID, orden.Objetivo, orden.Estado))
	if orden.Reporte != "" {
		b.WriteString(fmt.Sprintf("\nReporte: %s", orden.Reporte))
	}
	if len(orden.Historial) == 0 {
		b.WriteString("\nSin acciones registradas.")
	} else {
		b.WriteString("\nAcciones:")
		for _, a := range orden.Historial {
			b.WriteString(fmt.Sprintf("\n  [%s] %s -> %s", a.Momento, a.Accion, a.Resultado))
		}
	}
	return b.String()
}

func extraerID(entrada string) int {
	for _, campo := range strings.Fields(entrada) {
		campo = strings.TrimPrefix(campo, "#")
		if campo == "" {
			continue
		}
		id := 0
		if _, err := fmt.Sscanf(campo, "%d", &id); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

// extraerPIN toma el primer grupo de 4-6 dígitos de la entrada (el PIN
// que el dueño dice junto al comando de aprobación).
func extraerPIN(entrada string) string {
	for _, campo := range strings.Fields(entrada) {
		campo = strings.Trim(campo, ".,;")
		if len(campo) >= 4 && len(campo) <= 6 {
			todoDigitos := true
			for _, c := range campo {
				if c < '0' || c > '9' {
					todoDigitos = false
					break
				}
			}
			if todoDigitos {
				return campo
			}
		}
	}
	return ""
}
