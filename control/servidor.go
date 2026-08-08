package control

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed panel.html
var panelFS embed.FS

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
	mux.HandleFunc("/api/v1/publicar", s.manejarPublicar)
	mux.HandleFunc("/api/v1/descargar", s.manejarDescargar)
	mux.HandleFunc("/api/v1/release", s.manejarRelease)
	mux.HandleFunc("/panel", s.manejarPanel)
	mux.HandleFunc("/", s.manejarRaiz)
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

// peticionClienteValida autentica al agente por su licencia: comprueba que el
// par clave/id_instalacion esté registrado y que la licencia siga activa.
func (s *Servidor) peticionClienteValida(r *http.Request) bool {
	clave := strings.TrimSpace(r.Header.Get("X-Jarvis-Clave"))
	instalacion := strings.TrimSpace(r.Header.Get("X-Jarvis-Instalacion"))
	if clave == "" || instalacion == "" {
		return false
	}
	activa, _, _, _, err := s.gestor.Heartbeat(clave, instalacion, "", 0)
	return err == nil && activa
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
		Plan    string `json:"plan"`
		Puestos int    `json:"puestos"`
		Cliente string `json:"cliente"`
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
		Clave         string `json:"clave"`
		IDInstalacion string `json:"id_instalacion"`
		Nombre        string `json:"nombre"`
		Version       string `json:"version"`
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
// El GET también informa el release publicado (versión + sha256) para que el
// panel del dueño lo muestre.
func (s *Servidor) manejarVersion(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rv, rsha := s.gestor.ReleaseInfo()
		release := map[string]interface{}{}
		if rv != "" {
			release = map[string]interface{}{"version": rv, "sha256": rsha}
		}
		responder(w, http.StatusOK, map[string]interface{}{
			"latest_version": s.gestor.UltimaVersion(),
			"release":        release,
		})
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

// manejarPublicar recibe el binario del agente (""raw body"") para una versión
// y lo publica como release. Solo admin (token maestra). El binario se sirve
// luego por /api/v1/descargar a los agentes autenticados.
func (s *Servidor) manejarPublicar(w http.ResponseWriter, r *http.Request) {
	if !s.peticionAdminValida(r) {
		escribirError(w, http.StatusUnauthorized, "token de administración inválido")
		return
	}
	if r.Method != http.MethodPost {
		escribirError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	datos, err := io.ReadAll(r.Body)
	if err != nil {
		escribirError(w, http.StatusBadRequest, "no se pudo leer el binario")
		return
	}
	if version == "" {
		escribirError(w, http.StatusBadRequest, "falta el parámetro version")
		return
	}
	sha, err := s.gestor.PublicarRelease(version, datos)
	if err != nil {
		escribirError(w, http.StatusBadRequest, err.Error())
		return
	}
	responder(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "version": version, "sha256": sha, "bytes": len(datos),
	})
}

// manejarRelease informa (GET, autenticado por licencia) la versión publicada
// y su SHA-256, para que el agente decida si descarga. El binario en sí va
// por /api/v1/descargar.
func (s *Servidor) manejarRelease(w http.ResponseWriter, r *http.Request) {
	if !s.peticionClienteValida(r) {
		escribirError(w, http.StatusUnauthorized, "licencia inválida o inactiva")
		return
	}
	version, sha := s.gestor.ReleaseInfo()
	responder(w, http.StatusOK, map[string]interface{}{"version": version, "sha256": sha})
}

// manejarDescargar sirve el binario publicado (GET, autenticado por licencia).
func (s *Servidor) manejarDescargar(w http.ResponseWriter, r *http.Request) {
	if !s.peticionClienteValida(r) {
		escribirError(w, http.StatusUnauthorized, "licencia inválida o inactiva")
		return
	}
	if r.Method != http.MethodGet {
		escribirError(w, http.StatusMethodNotAllowed, "use GET")
		return
	}
	version, sha := s.gestor.ReleaseInfo()
	datos := s.gestor.ReleaseDatos()
	if len(datos) == 0 {
		escribirError(w, http.StatusNotFound, "no hay binario publicado")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=JarvisOS-%s.exe", version))
	w.Header().Set("X-Jarvis-Version", version)
	w.Header().Set("X-Jarvis-SHA256", sha)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(datos)
}

func (s *Servidor) manejarSalud(w http.ResponseWriter, r *http.Request) {
	responder(w, http.StatusOK, map[string]interface{}{"ok": true, "servicio": "plano de control JarvisOS"})
}

// manejarPanel sirve el dashboard del dueño (web embebida en el binario). La
// página es pública; todas sus llamadas a /api/v1/* exigen el token maestra.
func (s *Servidor) manejarPanel(w http.ResponseWriter, r *http.Request) {
	datos, err := panelFS.ReadFile("panel.html")
	if err != nil {
		escribirError(w, http.StatusInternalServerError, "no se pudo leer el panel")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(datos)
}

// manejarRaiz redirige la raíz al panel del dueño.
func (s *Servidor) manejarRaiz(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		escribirError(w, http.StatusNotFound, "ruta desconocida")
		return
	}
	http.Redirect(w, r, "/panel", http.StatusFound)
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
