package webui

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"JarvisOS/core"
	"JarvisOS/core/audit"
	"JarvisOS/core/security"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type ProcesadorChat interface {
	Process(input string) string
}

type EstadoProvider interface {
	EstadoPanel() core.EstadoPanel
}

type DiagnosticoProvider interface {
	Diagnostico() (*core.Diagnostico, error)
}

// Aprobador autoriza o deniega las acciones sensibles que el agente deja
// en espera, y expone las órdenes activas para el panel del dueño.
type Aprobador interface {
	AprobarOrden(id int, pin string) string
	DenegarOrden(id int) string
	OrdenesParaPanel() []core.Orden
}

// Auditor expone el registro inmutable para el visor de auditoría del panel.
type Auditor interface {
	AuditoriaPanel() []audit.Entrada
}

// GestorSkills habilita el CRUD de skills desde la WebUI.
type GestorSkills interface {
	ListarDetallado() []core.SkillInfo
	Obtener(nombre string) (core.Skill, bool)
	CrearOActualizar(core.Skill) error
	Eliminar(nombre string) error
}

// GestorRoles habilita el CRUD de roles desde la WebUI.
type GestorRoles interface {
	ListarDetallado() []core.RolInfo
	Obtener(nombre string) (core.Rol, bool)
	CrearOActualizar(core.Rol) error
	Eliminar(nombre string) error
}

// GestorEmpresa expone el perfil de la empresa para verlo/guardarlo desde la WebUI.
type GestorEmpresa interface {
	Obtener() core.PerfilEmpresa
	Reemplazar(core.PerfilEmpresa) error
	Resumen() string
}

// GestorPerfil expone quién opera JarvisOS para verlo y seleccionarlo desde
// la WebUI (dueño, admin, empleado o un usuario registrado).
type GestorPerfil interface {
	Usuarios() []core.PerfilUsuario
	Activo() string
	ActivoRol() string
	Seleccionar(texto string) bool
	AgregarUsuario(nombre, area, rol string) bool
	Eliminar(nombre string) bool
}

// nombreCookieSesion identifica la cookie de sesión del panel.
const nombreCookieSesion = "jarvis_sesion"

// duracionSesion es cuánto dura el inicio de sesión del dueño.
const duracionSesion = 12 * time.Hour

type ServidorWeb struct {
	brain       ProcesadorChat
	estado      EstadoProvider
	diagnostico DiagnosticoProvider
	aprobador   Aprobador
	auditor     Auditor
	skills      GestorSkills
	roles       GestorRoles
	empresa     GestorEmpresa
	perfil      GestorPerfil
	historial   []HistorialEntry
	mu          sync.Mutex
	port        int
	rutaHist    string

	contrasenaHash string
	sesionesMu     sync.Mutex
	sesiones       map[string]time.Time
}

type HistorialEntry struct {
	Usuario   string `json:"usuario"`
	Jarvis    string `json:"jarvis"`
	Timestamp string `json:"timestamp"`
}

func NuevoServidor(brain ProcesadorChat, port int, opciones ...ServidorOpciones) *ServidorWeb {
	s := &ServidorWeb{
		brain:     brain,
		historial: make([]HistorialEntry, 0),
		port:      port,
		sesiones:  make(map[string]time.Time),
	}
	for _, o := range opciones {
		if o.Estado != nil {
			s.estado = o.Estado
		}
		if o.Diagnostico != nil {
			s.diagnostico = o.Diagnostico
		}
		if o.Aprobador != nil {
			s.aprobador = o.Aprobador
		}
		if o.Auditor != nil {
			s.auditor = o.Auditor
		}
		if o.Skills != nil {
			s.skills = o.Skills
		}
		if o.Roles != nil {
			s.roles = o.Roles
		}
		if o.Empresa != nil {
			s.empresa = o.Empresa
		}
		if o.Perfil != nil {
			s.perfil = o.Perfil
		}
		if o.RutaHistorial != "" {
			s.rutaHist = o.RutaHistorial
		}
		if o.ContrasenaHash != "" {
			s.contrasenaHash = o.ContrasenaHash
		}
	}
	s.cargarHistorial()
	return s
}

type ServidorOpciones struct {
	Estado         EstadoProvider
	Diagnostico    DiagnosticoProvider
	Aprobador      Aprobador
	Auditor        Auditor
	Skills         GestorSkills
	Roles          GestorRoles
	Empresa        GestorEmpresa
	Perfil         GestorPerfil
	RutaHistorial  string
	ContrasenaHash string
}

type AprobacionRequest struct {
	OrdenID int    `json:"orden_id"`
	PIN     string `json:"pin"`
}

func (s *ServidorWeb) Iniciar() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", s.manejarChat)
	mux.HandleFunc("/api/historial", s.manejarHistorial)
	mux.HandleFunc("/api/limpiar", s.manejarLimpiar)
	mux.HandleFunc("/api/estado", s.manejarEstado)
	mux.HandleFunc("/api/diagnostico", s.manejarDiagnostico)
	mux.HandleFunc("/api/salud", s.manejarSalud)
	mux.HandleFunc("/api/ordenes", s.manejarOrdenes)
	mux.HandleFunc("/api/aprobar", s.manejarAprobar)
	mux.HandleFunc("/api/denegar", s.manejarDenegar)
	mux.HandleFunc("/api/sesion", s.manejarSesion)
	mux.HandleFunc("/api/login", s.manejarLogin)
	mux.HandleFunc("/api/logout", s.manejarLogout)
	mux.HandleFunc("/api/auditoria", s.manejarAuditoria)
	mux.HandleFunc("/api/skills", s.manejarSkills)
	mux.HandleFunc("/api/roles", s.manejarRoles)
	mux.HandleFunc("/api/empresa", s.manejarEmpresa)
	mux.HandleFunc("/api/perfil", s.manejarPerfil)
	mux.HandleFunc("/api/dashboard", s.manejarDashboard)

	mux.Handle("/", http.FileServer(http.FS(archivosEstaticos)))

	var listener net.Listener
	var err error
	for puerto := s.port; puerto < s.port+10; puerto++ {
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", puerto))
		if err == nil {
			s.port = puerto
			break
		}
	}
	if listener == nil {
		return fmt.Errorf("no hay puertos libres (8080-8089): %v", err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	fmt.Printf("[WEBUI] Interfaz disponible en http://%s\n", addr)

	go s.abrirNavegador(addr)

	return http.Serve(listener, s.conRecuperacion(mux))
}

// conRecuperacion evita que un error interno de un handler derribe a todo el
// servidor: convierte el pánico en una respuesta 500 y el proceso sigue vivo.
func (s *ServidorWeb) conRecuperacion(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("[WEBUI] Error interno en %s: %v\n", r.URL.Path, rec)
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "error interno del servidor"})
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (s *ServidorWeb) abrirNavegador(addr string) {
	time.Sleep(500 * time.Millisecond)
	url := fmt.Sprintf("http://%s", addr)
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}

func (s *ServidorWeb) manejarChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		_ = json.NewEncoder(w).Encode(ChatResponse{Response: ""})
		return
	}

	respuesta := s.brain.Process(req.Message)

	s.mu.Lock()
	s.historial = append(s.historial, HistorialEntry{
		Usuario:   req.Message,
		Jarvis:    respuesta,
		Timestamp: time.Now().Format("15:04:05"),
	})
	if len(s.historial) > 100 {
		s.historial = s.historial[len(s.historial)-100:]
	}
	s.mu.Unlock()
	s.guardarHistorial()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ChatResponse{Response: respuesta})
}

func (s *ServidorWeb) manejarEstado(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.estado == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin estado"})
		return
	}
	_ = json.NewEncoder(w).Encode(s.estado.EstadoPanel())
}

func (s *ServidorWeb) manejarDiagnostico(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.diagnostico == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin diagnostico"})
		return
	}
	d, err := s.diagnostico.Diagnostico()
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(d)
}

func (s *ServidorWeb) manejarSalud(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.diagnostico == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin diagnostico"})
		return
	}
	d, err := s.diagnostico.Diagnostico()
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	puntaje, problemas := core.AnalizarSalud(d)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"puntaje":   puntaje,
		"problemas": problemas,
	})
}

// OrdenPanel es la vista del panel del dueño para una orden activa.
type OrdenPanel struct {
	ID                   int    `json:"id"`
	Objetivo             string `json:"objetivo"`
	Estado               string `json:"estado"`
	PendienteAccion      string `json:"pendiente_accion,omitempty"`
	PendienteDescripcion string `json:"pendiente_descripcion,omitempty"`
}

func (s *ServidorWeb) manejarOrdenes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.aprobador == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin aprobador"})
		return
	}
	ordenes := s.aprobador.OrdenesParaPanel()
	panel := make([]OrdenPanel, 0, len(ordenes))
	for _, o := range ordenes {
		panel = append(panel, OrdenPanel{
			ID:                   o.ID,
			Objetivo:             o.Objetivo,
			Estado:               o.Estado,
			PendienteAccion:      o.PendienteAccion,
			PendienteDescripcion: o.PendienteDescripcion,
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ordenes": panel})
}

func (s *ServidorWeb) manejarAprobar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.exigePermiso(r, w, security.PermisoAprobar) {
		return
	}
	if s.aprobador == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin aprobador"})
		return
	}
	var req AprobacionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"response": s.aprobador.AprobarOrden(req.OrdenID, req.PIN)})
}

func (s *ServidorWeb) manejarDenegar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.exigePermiso(r, w, security.PermisoDenegar) {
		return
	}
	if s.aprobador == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin aprobador"})
		return
	}
	var req AprobacionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"response": s.aprobador.DenegarOrden(req.OrdenID)})
}

// === SESIÓN Y ROLES (RBAC F2) ===

type LoginRequest struct {
	Clave string `json:"clave"`
}

// manejarSesion informa al panel su estado de autenticación y qué rol tiene.
func (s *ServidorWeb) manejarSesion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rol := s.rolActual(r)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"autenticado":   rol == security.RolAdmin,
		"rol":           rol,
		"requiere_login": s.contrasenaHash != "",
	})
}

// manejarLogin valida la contraseña del dueño y abre una sesión Admin.
func (s *ServidorWeb) manejarLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
		return
	}
	if s.contrasenaHash != "" && core.HashTexto(normalizarClave(req.Clave)) != s.contrasenaHash {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "contraseña incorrecta"})
		return
	}
	token := nuevoTokenSesion()
	s.sesionesMu.Lock()
	s.sesiones[token] = time.Now().Add(duracionSesion)
	s.sesionesMu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: nombreCookieSesion, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: int(duracionSesion.Seconds()),
	})
	_ = json.NewEncoder(w).Encode(map[string]string{"rol": string(security.RolAdmin)})
}

// manejarLogout cierra la sesión actual.
func (s *ServidorWeb) manejarLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if c, err := r.Cookie(nombreCookieSesion); err == nil {
		s.sesionesMu.Lock()
		delete(s.sesiones, c.Value)
		s.sesionesMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name: nombreCookieSesion, Value: "", Path: "/",
		HttpOnly: true, MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}

// manejarAuditoria devuelve el registro inmutable (solo Admin).
func (s *ServidorWeb) manejarAuditoria(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !s.exigePermiso(r, w, security.PermisoAuditoria) {
		return
	}
	if s.auditor == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin auditor"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entradas": s.auditor.AuditoriaPanel()})
}

// === CRUD DE SKILLS Y ROLES ===

// manejarSkills lista las skills (GET) o crea/actualiza una (POST). La
// escritura es solo para Admin.
func (s *ServidorWeb) manejarSkills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.skills == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin gestor de skills"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if nombre := r.URL.Query().Get("nombre"); nombre != "" {
			if sk, ok := s.skills.Obtener(nombre); ok {
				_ = json.NewEncoder(w).Encode(sk)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "skill no encontrada"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"skills": s.skills.ListarDetallado()})
	case http.MethodPost:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		var sk core.Skill
		if err := json.NewDecoder(r.Body).Decode(&sk); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
			return
		}
		if err := s.skills.CrearOActualizar(sk); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "skill guardada", "nombre": sk.Nombre})
	case http.MethodDelete:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		nombre := r.URL.Query().Get("nombre")
		if err := s.skills.Eliminar(nombre); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "skill eliminada"})
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

// manejarRoles lista los roles (GET) o crea/actualiza uno (POST). La escritura
// es solo para Admin.
func (s *ServidorWeb) manejarRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.roles == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin gestor de roles"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if nombre := r.URL.Query().Get("nombre"); nombre != "" {
			if rol, ok := s.roles.Obtener(nombre); ok {
				_ = json.NewEncoder(w).Encode(rol)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "rol no encontrado"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"roles": s.roles.ListarDetallado()})
	case http.MethodPost:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		var rol core.Rol
		if err := json.NewDecoder(r.Body).Decode(&rol); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
			return
		}
		if err := s.roles.CrearOActualizar(rol); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "rol guardado", "nombre": rol.Nombre})
	case http.MethodDelete:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		nombre := r.URL.Query().Get("nombre")
		if err := s.roles.Eliminar(nombre); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "rol eliminado"})
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

// manejarEmpresa expone el perfil de la empresa: GET lo consulta y POST lo
// reemplaza (solo Admin para la escritura).
func (s *ServidorWeb) manejarEmpresa(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.empresa == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin gestor de empresa"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"perfil":  s.empresa.Obtener(),
			"resumen": s.empresa.Resumen(),
		})
	case http.MethodPut, http.MethodPost:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		var perfil core.PerfilEmpresa
		if err := json.NewDecoder(r.Body).Decode(&perfil); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
			return
		}
		if err := s.empresa.Reemplazar(perfil); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "perfil de empresa guardado"})
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

// manejarPerfil expone quién opera JarvisOS: GET consulta usuarios y perfil
// activo; POST selecciona el activo o registra/elimina usuarios (Admin).
func (s *ServidorWeb) manejarPerfil(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.perfil == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin gestor de perfil"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"usuarios": s.perfil.Usuarios(),
			"activo":   s.perfil.Activo(),
			"rol":      s.perfil.ActivoRol(),
		})
	case http.MethodPut, http.MethodPost:
		if !s.exigePermiso(r, w, security.PermisoAprobar) {
			return
		}
		var req struct {
			Seleccionar string `json:"seleccionar"`
			Nombre       string `json:"nombre"`
			Area         string `json:"area"`
			Rol          string `json:"rol"`
			Eliminar     string `json:"eliminar"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request inválido"})
			return
		}
		switch {
		case req.Eliminar != "":
			if s.perfil.Eliminar(req.Eliminar) {
				_ = json.NewEncoder(w).Encode(map[string]string{"ok": "usuario eliminado"})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "usuario no encontrado"})
			}
		case req.Nombre != "":
			s.perfil.AgregarUsuario(req.Nombre, req.Area, req.Rol)
			_ = json.NewEncoder(w).Encode(map[string]string{"ok": "usuario guardado"})
		case req.Seleccionar != "":
			if s.perfil.Seleccionar(req.Seleccionar) {
				_ = json.NewEncoder(w).Encode(map[string]string{"ok": "perfil activado"})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "perfil no encontrado"})
			}
		default:
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "falta acción"})
		}
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

// PanelResumen es el payload agregado del panel del dueño: junta qué está
// en juego (órdenes), qué hizo hoy (auditoría) y el estado de la PC, para
// que el dashboard no haga varias llamadas.
type PanelResumen struct {
	Hora              string           `json:"hora"`
	Fecha             string           `json:"fecha"`
	OrdenesActivas    int              `json:"ordenes_activas"`
	EsperandoAprob    int              `json:"esperando_aprobacion"`
	Bloqueadas        int              `json:"bloqueadas"`
	AuditoriaHoy      int              `json:"auditoria_hoy"`
	Estados           map[string]int   `json:"estados"`
	Ordenes           []OrdenPanel     `json:"ordenes"`
	ActividadReciente []audit.Entrada  `json:"actividad_reciente"`
	Estado            core.EstadoPanel `json:"estado"`
}

// manejarDashboard entrega el material del panel del dueño. Es de solo
// lectura y no exige rol: el dueño ve la radiografía sin iniciar sesión,
// igual que /api/estado.
func (s *ServidorWeb) manejarDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.aprobador == nil && s.auditor == nil {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sin datos de panel"})
		return
	}

	res := PanelResumen{
		Hora:    time.Now().Format("15:04:05"),
		Fecha:   time.Now().Format("02/01/2006"),
		Estados: make(map[string]int),
	}

	if s.estado != nil {
		res.Estado = s.estado.EstadoPanel()
	}

	ordenes := []OrdenPanel{}
	if s.aprobador != nil {
		for _, o := range s.aprobador.OrdenesParaPanel() {
			res.OrdenesActivas++
			res.Estados[o.Estado]++
			switch o.Estado {
			case core.OrdenEsperandoAprobacion:
				res.EsperandoAprob++
			case core.OrdenBloqueada:
				res.Bloqueadas++
			}
			ordenes = append(ordenes, OrdenPanel{
				ID:                   o.ID,
				Objetivo:             o.Objetivo,
				Estado:               o.Estado,
				PendienteAccion:      o.PendienteAccion,
				PendienteDescripcion: o.PendienteDescripcion,
			})
		}
	}
	res.Ordenes = ordenes

	if s.auditor != nil {
		hoy := time.Now().Format("2006-01-02")
		entradas := s.auditor.AuditoriaPanel()
		for _, e := range entradas {
			if strings.HasPrefix(e.Momento, hoy) {
				res.AuditoriaHoy++
			}
		}
		if n := len(entradas); n > 12 {
			entradas = entradas[n-12:]
		}
		res.ActividadReciente = entradas
	}

	_ = json.NewEncoder(w).Encode(res)
}

// rolActual resuelve el rol del pedido: Admin si tiene una sesión válida (o
func (s *ServidorWeb) rolActual(r *http.Request) security.Rol {
	if s.contrasenaHash == "" {
		return security.RolAdmin
	}
	c, err := r.Cookie(nombreCookieSesion)
	if err != nil {
		return security.RolOperador
	}
	s.sesionesMu.Lock()
	exp, ok := s.sesiones[c.Value]
	s.sesionesMu.Unlock()
	if !ok || time.Now().After(exp) {
		return security.RolOperador
	}
	return security.RolAdmin
}

// exigePermiso responde 403 si el rol actual no tiene el permiso y devuelve
// false (el handler debe retornar); true si puede seguir.
func (s *ServidorWeb) exigePermiso(r *http.Request, w http.ResponseWriter, permiso security.Permiso) bool {
	if security.TienePermiso(s.rolActual(r), permiso) {
		return true
	}
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "acceso denegado: solo administrador"})
	return false
}

func nuevoTokenSesion() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func normalizarClave(clave string) string {
	return strings.ToLower(strings.TrimSpace(clave))
}

func (s *ServidorWeb) manejarHistorial(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.historial)
}

func (s *ServidorWeb) manejarLimpiar(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.historial = nil
	s.mu.Unlock()
	s.guardarHistorial()
	w.WriteHeader(http.StatusOK)
}

func (s *ServidorWeb) cargarHistorial() {
	if s.rutaHist == "" {
		return
	}
	datos, err := os.ReadFile(s.rutaHist)
	if err != nil {
		return
	}
	var historial []HistorialEntry
	if err := json.Unmarshal(datos, &historial); err != nil {
		return
	}
	s.mu.Lock()
	s.historial = historial
	s.mu.Unlock()
}

func (s *ServidorWeb) guardarHistorial() {
	if s.rutaHist == "" {
		return
	}
	s.mu.Lock()
	datos, _ := json.MarshalIndent(s.historial, "", "  ")
	s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.rutaHist), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(s.rutaHist, datos, 0o600)
}

//go:embed static
var archivosEmbeber embed.FS

var archivosEstaticos fs.FS

func init() {
	var err error
	archivosEstaticos, err = fs.Sub(archivosEmbeber, "static")
	if err != nil {
		panic(err)
	}
}
