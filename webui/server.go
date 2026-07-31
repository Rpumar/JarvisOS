package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"JarvisOS/core"
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

type ServidorWeb struct {
	brain      ProcesadorChat
	estado     EstadoProvider
	diagnostico DiagnosticoProvider
	historial  []HistorialEntry
	mu         sync.Mutex
	port       int
	rutaHist   string
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
	}
	for _, o := range opciones {
		if o.Estado != nil {
			s.estado = o.Estado
		}
		if o.Diagnostico != nil {
			s.diagnostico = o.Diagnostico
		}
		if o.RutaHistorial != "" {
			s.rutaHist = o.RutaHistorial
		}
	}
	s.cargarHistorial()
	return s
}

type ServidorOpciones struct {
	Estado        EstadoProvider
	Diagnostico   DiagnosticoProvider
	RutaHistorial string
}

func (s *ServidorWeb) Iniciar() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/chat", s.manejarChat)
	mux.HandleFunc("/api/historial", s.manejarHistorial)
	mux.HandleFunc("/api/limpiar", s.manejarLimpiar)
	mux.HandleFunc("/api/estado", s.manejarEstado)
	mux.HandleFunc("/api/diagnostico", s.manejarDiagnostico)
	mux.HandleFunc("/api/salud", s.manejarSalud)

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
				json.NewEncoder(w).Encode(map[string]string{"error": "error interno del servidor"})
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
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
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
		json.NewEncoder(w).Encode(ChatResponse{Response: ""})
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
	json.NewEncoder(w).Encode(ChatResponse{Response: respuesta})
}

func (s *ServidorWeb) manejarEstado(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.estado == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "sin estado"})
		return
	}
	json.NewEncoder(w).Encode(s.estado.EstadoPanel())
}

func (s *ServidorWeb) manejarDiagnostico(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.diagnostico == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "sin diagnostico"})
		return
	}
	d, err := s.diagnostico.Diagnostico()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(d)
}

func (s *ServidorWeb) manejarSalud(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.diagnostico == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "sin diagnostico"})
		return
	}
	d, err := s.diagnostico.Diagnostico()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	puntaje, problemas := core.AnalizarSalud(d)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"puntaje":   puntaje,
		"problemas": problemas,
	})
}

func (s *ServidorWeb) manejarHistorial(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.historial)
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
	os.MkdirAll(filepath.Dir(s.rutaHist), 0o700)
	os.WriteFile(s.rutaHist, datos, 0o600)
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
