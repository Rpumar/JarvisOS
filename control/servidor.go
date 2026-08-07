package control

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Opciones del servidor HTTP del plano de control.
type Opciones struct {
	// Puerto es el puerto de escucha (por defecto 8443).
	Puerto int
	// Token es la clave maestra de administración para emitir/suspender/
	// consultar licencias. Si está vacía, esos endpoints responden 401.
	Token string
	// Dir es el directorio donde persistir licencias.json y clientes.json.
	Dir string
}

// Servidor expone la API HTTP del plano de control.
type Servidor struct {
	gestor *Gestor
	port   int
	token  string
}

// NuevoServidor crea el servidor sobre un gestor existente.
func NuevoServidor(gestor *Gestor, opciones Opciones) *Servidor {
	if opciones.Puerto <= 0 {
		opciones.Puerto = 8443
	}
	return &Servidor{gestor: gestor, port: opciones.Puerto, token: opciones.Token}
}

// Iniciar atiende requests hasta que el proceso termine (bloqueante).
func (s *Servidor) Iniciar() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/emitir", s.manejarEmitir)
	mux.HandleFunc("/api/v1/suspender", s.manejarSuspender)
	mux.HandleFunc("/api/v1/reactivar", s.manejarReactivar)
	mux.HandleFunc("/api/v1/activar", s.manejarActivar)
	mux.HandleFunc("/api/v1/heartbeat", s.manejarHeartbeat)
	mux.HandleFunc("/api/v1/estado", s.manejarEstado)
	mux.HandleFunc("/api/v1/version", s.manejarVersion)
	mux.HandleFunc("/salud", s.manejarSalud)

	var listener net.Listener
	var err error
	for puerto := s.port; puerto < s.port+10; puerto++ {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", puerto))
		if err == nil {
			s.port = puerto
			break
		}
	}
	if listener == nil {
		return fmt.Errorf("no hay puertos libres (%d-%d): %v", s.port, s.port+9, err)
	}

	fmt.Printf("[CONTROL] Plano de control escuchando en http://0.0.0.0:%d\n", s.port)
	return http.Serve(listener, s.conRecuperacion(mux))
}

func (s *Servidor) conRecuperacion(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("[CONTROL] Error interno en %s: %v\n", r.URL.Path, rec)
				escribirError(w, http.StatusInternalServerError, "error interno del servidor")
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// peticionAdminVerifica comprueba el token maestra en Authorization: Bearer.
func (s *Servidor) peticionAdminValida(r *http.Request) bool {
	if s.token == "" {
		return false
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+s.token
}

func (s *Servidor) manejarEmitir(w http.ResponseWriter, r *http.Request) {
	if !s.peticionAdminValida(r) {
		escribirError(w, http.StatusUnauthorized, "token de administración inválido")
		return
	}
	if r.Method != http.MethodPost {
		escribirError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var cuerpo struct {
		Plan     string `json:"plan"`
		Puestos  int    `json:"puestos"`
		Cliente  string `json:"cliente"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
		escribirError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	clave, err := s.gestor.Emitir(cuerpo.Plan, cuerpo.Puestos, cuerpo.Cliente)
	if err != nil {
		escribirError(w, http.StatusBadRequest, err.Error())
		return
	}
	responder(w, http.StatusCreated, map[string]interface{}{"clave": clave})
}

func (s *Servidor) manejarSuspender(w http.ResponseWriter, r *http.Request) {
	s.manejarCambioEstado(w, r, false)
}

func (s *Servidor) manejarReactivar(w http.ResponseWriter, r *http.Request) {
	s.manejarCambioEstado(w, r, true)
}

func (s *Servidor) manejarCambioEstado(w http.ResponseWriter, r *http.Request, reactivar bool) {
	if !s.peticionAdminValida(r) {
		escribirError(w, http.StatusUnauthorized, "token de administración inválido")
		return
	}
	if r.Method != http.MethodPost {
		escribirError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var cuerpo struct {
		Clave string `json:"clave"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
		escribirError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	var ok bool
	if reactivar {
		ok = s.gestor.Reactivar(cuerpo.Clave)
	} else {
		ok = s.gestor.Suspender(cuerpo.Clave)
	}
	if !ok {
		escribirError(w, http.StatusNotFound, "la clave de licencia no existe")
		return
	}
	responder(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (s *Servidor) manejarActivar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		escribirError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var cuerpo struct {
		Clave        string `json:"clave"`
		IDInstalacion string `json:"id_instalacion"`
		Nombre       string `json:"nombre"`
		Version      string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
		escribirError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	err := s.gestor.Activar(cuerpo.Clave, cuerpo.IDInstalacion, cuerpo.Nombre, cuerpo.Version)
	if err != nil {
		status := http.StatusBadRequest
		if err == ErrSinPuestosLibres {
			status = http.StatusConflict
		}
		escribirError(w, status, err.Error())
		return
	}
	plan, puestos := s.planPuestos(cuerpo.Clave)
	responder(w, http.StatusOK, map[string]interface{}{"ok": true, "plan": plan, "puestos": puestos})
}

func (s *Servidor) manejarHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		escribirError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var cuerpo struct {
		Clave         string `json:"clave"`
		IDInstalacion string `json:"id_instalacion"`
		Version       string `json:"version"`
		PuestosUsados int    `json:"puestos_usados"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
		escribirError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	activa, plan, tope, latest, err := s.gestor.Heartbeat(cuerpo.Clave, cuerpo.IDInstalacion, cuerpo.Version, cuerpo.PuestosUsados)
	if err != nil {
		status := http.StatusBadRequest
		if err == ErrLicenciaInactiva {
			status = http.StatusForbidden
		}
		escribirError(w, status, err.Error())
		return
	}
	responder(w, http.StatusOK, map[string]interface{}{"ok": true, "activa": activa, "plan": plan, "puestos": tope, "latest_version": latest})
}

// manejarVersion publica la última versión (POST, admin) o la consulta (GET).
func (s *Servidor) manejarVersion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		responder(w, http.StatusOK, map[string]interface{}{"latest_version": s.gestor.UltimaVersion()})
	case http.MethodPost:
		if !s.peticionAdminValida(r) {
			escribirError(w, http.StatusUnauthorized, "token de administración inválido")
			return
		}
		var cuerpo struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&cuerpo); err != nil {
			escribirError(w, http.StatusBadRequest, "cuerpo JSON inválido")
			return
		}
		s.gestor.PublicarVersion(cuerpo.Version)
		responder(w, http.StatusOK, map[string]interface{}{"ok": true, "latest_version": s.gestor.UltimaVersion()})
	default:
		escribirError(w, http.StatusMethodNotAllowed, "use GET o POST")
	}
}

func (s *Servidor) manejarEstado(w http.ResponseWriter, r *http.Request) {
	if !s.peticionAdminValida(r) {
		escribirError(w, http.StatusUnauthorized, "token de administración inválido")
		return
	}
	responder(w, http.StatusOK, map[string]interface{}{
		"licencias": s.gestor.Licencias(),
		"clientes":  s.gestor.Clientes(),
		"tiempo":    time.Now().Format(time.RFC3339),
	})
}

func (s *Servidor) manejarSalud(w http.ResponseWriter, r *http.Request) {
	responder(w, http.StatusOK, map[string]interface{}{"ok": true, "servicio": "plano de control JarvisOS"})
}

func (s *Servidor) planPuestos(clave string) (string, int) {
	for _, l := range s.gestor.Licencias() {
		if l.Clave == clave {
			return l.Plan, l.Puestos
		}
	}
	return "", 0
}

func responder(w http.ResponseWriter, status int, valor interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(valor)
}

func escribirError(w http.ResponseWriter, status int, mensaje string) {
	responder(w, status, map[string]string{"error": mensaje})
}

// ArchivoEnvControlToken es el nombre de variable de entorno para el token
// maestra del plano de control (evita guardar el secreto en config.json).
const ArchivoEnvControlToken = "JARVISOS_CONTROL_TOKEN"

// TokenDesdeEntorno lee el token maestra desde la variable de entorno. Es la
// forma recomendada de arrancar el servidor.
func TokenDesdeEntorno() string {
	return strings.TrimSpace(os.Getenv(ArchivoEnvControlToken))
}
