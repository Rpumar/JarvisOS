package ia

import (
	"net/http"
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
		if !c.Disponible() {
			t.Error("Disponible() deberia ser true con backend activo")
		}
	})
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
