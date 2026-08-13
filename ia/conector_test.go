package ia

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"JarvisOS/core"
)

func TestDisponible(t *testing.T) {
	t.Run("sin backend", func(t *testing.T) {
		c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}}
		_ = c.Disponible()
		if c.Disponible() {
			t.Error("Disponible() deberia ser false sin backend")
		}
	})

	t.Run("con backend activo", func(t *testing.T) {
		c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}, disponible: true}
		// ProbeTime en el futuro evita que Disponible() re-probee en este test.
		c.mu.Lock()
		c.probeTime = time.Now().Add(time.Hour)
		c.mu.Unlock()
		if !c.Disponible() {
			t.Error("Disponible() deberia ser true con backend activo")
		}
	})
}

func TestDisponible_ReProbeLazy(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}

	// Sin probe previo, la primera llamada re-probea contra el backend; la
	// segunda se sirve del cache dentro del TTL.
	if !c.Disponible() {
		t.Fatal("Disponible() debia re-probear y estar disponible")
	}
	if !c.Disponible() {
		t.Fatal("con TTL fresco no debia perder disponibilidad")
	}
	if hits != 1 {
		t.Fatalf("llamadas al backend = %d, esperaba 1 (la segunda usó cache)", hits)
	}

	// Expirar el TTL fuerza un re-probe en la siguiente llamada.
	c.mu.Lock()
	c.probeTime = time.Now().Add(-2 * ttlProbe)
	c.mu.Unlock()
	if !c.Disponible() {
		t.Fatal("tras el re-probe debia seguir disponible")
	}
	if hits != 2 {
		t.Fatalf("llamadas al backend = %d, esperaba 2 (re-probe perezoso)", hits)
	}
}

func TestDisponible_BackendCaido(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}

	if c.Disponible() {
		t.Fatal("Disponible() debia ser false con /models en 500")
	}
	if _, err := c.Consultar("hola", nil); err == nil {
		t.Fatal("Consultar debia fallar si el backend responde 500")
	}
}

func TestConsultar_ConBackendHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"models":["llama3.2:3b"]}`)
		case "/chat/completions":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"Respuesta"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Conector{httpClient: srv.Client(), baseURL: srv.URL, modelo: "llama3.2:3b"}

	got, err := c.Consultar("decime algo", nil)
	if err != nil {
		t.Fatalf("Consultar falló: %v", err)
	}
	if got != "Respuesta" {
		t.Errorf("Consultar devolvió %q, esperaba %q", got, "Respuesta")
	}
}

func TestChat_ConBackendHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"models":[]}`)
		case "/chat/completions":
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"Respuesta"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Conector{httpClient: srv.Client(), baseURL: srv.URL}

	got, err := c.Chat("system", "user")
	if err != nil {
		t.Fatalf("Chat falló: %v", err)
	}
	if got != "Respuesta" {
		t.Errorf("Chat devolvió %q, esperaba %q", got, "Respuesta")
	}
}

func TestNuevoConector_TimeoutPorDefecto(t *testing.T) {
	c := NuevoConector("", 0, "", "")
	if c.httpClient.Timeout != 120*time.Second {
		t.Errorf("timeout por defecto = %v, esperaba 120s", c.httpClient.Timeout)
	}
	if c.baseURL != urlOllamaV1 {
		t.Errorf("baseURL por defecto = %q, esperaba %q", c.baseURL, urlOllamaV1)
	}
}

func TestNuevoConector_RespetaTimeoutProvisto(t *testing.T) {
	c := NuevoConector("test-model", 7*time.Second, "", "")
	if c.httpClient.Timeout != 7*time.Second {
		t.Errorf("timeout = %v, esperaba 7s", c.httpClient.Timeout)
	}
	if c.modelo != "test-model" {
		t.Errorf("modelo = %v, esperaba test-model", c.modelo)
	}
}

func TestNuevoConector_ConfiguraBaseURLYKey(t *testing.T) {
	c := NuevoConector("modelo-nube", 7*time.Second, "https://api.groq.com/openai/v1/", "mi-clave")
	if c.baseURL != "https://api.groq.com/openai/v1" {
		t.Errorf("baseURL = %q, esperaba sin barra final", c.baseURL)
	}
	if c.apiKey != "mi-clave" {
		t.Errorf("apiKey = %q, esperaba mi-clave", c.apiKey)
	}
}

func TestConsultar_SinDisponibilidad(t *testing.T) {
	c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}}

	if _, err := c.Consultar("algo", nil); err == nil {
		t.Error("se esperaba un error sin disponibilidad")
	}
}

func TestBuildMensajes(t *testing.T) {
	m := buildMensajes("decime la hora", nil)
	if len(m) != 2 {
		t.Fatalf("sin historial deberia haber system + user, hay %d", len(m))
	}
	if m[0].Role != "system" || !strings.Contains(m[0].Content, "JARVIS") {
		t.Errorf("primer mensaje deberia ser el system prompt de JARVIS")
	}
	if m[1].Role != "user" || m[1].Content != "decime la hora" {
		t.Errorf("ultimo mensaje deberia ser la peticion: %+v", m[1])
	}
}

func TestBuildMensajesConHistorial(t *testing.T) {
	historial := []core.TurnoConversacion{
		{Usuario: "hola", Asistente: "hola señor"},
		{Usuario: "como estas", Asistente: "muy bien"},
	}
	m := buildMensajes("gracias", historial)
	// system + 2 turnos (4) + user = 6
	if len(m) != 6 {
		t.Fatalf("esperaba 6 mensajes, hay %d", len(m))
	}
	if m[1].Content != "hola" || m[2].Content != "hola señor" {
		t.Errorf("historial mal intercalado: %+v %+v", m[1], m[2])
	}
	if m[len(m)-1].Content != "gracias" {
		t.Errorf("ultimo mensaje = %q", m[len(m)-1].Content)
	}
}
