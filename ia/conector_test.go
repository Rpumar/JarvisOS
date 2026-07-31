package ia

import (
	"net/http"
	"testing"
	"time"
)

func TestDisponible(t *testing.T) {
	t.Run("sin ollama", func(t *testing.T) {
		c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}}
		_ = c.Disponible()
		if c.Disponible() {
			t.Error("Disponible() deberia ser false sin Ollama")
		}
	})

	t.Run("con ollama activo", func(t *testing.T) {
		c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}, ollama: true}
		if !c.Disponible() {
			t.Error("Disponible() deberia ser true con Ollama activo")
		}
	})
}

func TestNuevoConector_TimeoutPorDefecto(t *testing.T) {
	c := NuevoConector("", 0)
	if c.httpClient.Timeout != 120*time.Second {
		t.Errorf("timeout por defecto = %v, esperaba 120s", c.httpClient.Timeout)
	}
}

func TestNuevoConector_RespetaTimeoutProvisto(t *testing.T) {
	c := NuevoConector("test-model", 7*time.Second)
	if c.httpClient.Timeout != 7*time.Second {
		t.Errorf("timeout = %v, esperaba 7s", c.httpClient.Timeout)
	}
	if c.modelo != "test-model" {
		t.Errorf("modelo = %v, esperaba test-model", c.modelo)
	}
}

func TestConsultar_SinDisponibilidad(t *testing.T) {
	c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}}

	if _, err := c.Consultar("algo", nil); err == nil {
		t.Error("se esperaba un error sin disponibilidad")
	}
}

func TestConsultarCodigo_SinDisponibilidad(t *testing.T) {
	c := &Conector{httpClient: &http.Client{Timeout: 5 * time.Second}}

	if _, _, err := c.ConsultarCodigo("algo"); err == nil {
		t.Error("se esperaba un error sin disponibilidad")
	}
}

func TestParsearRespuestaCodigo(t *testing.T) {
	casos := []struct {
		nombre              string
		entrada             string
		codigoEsperado      string
		explicacionEsperada string
	}{
		{
			nombre:              "formato bien formado",
			entrada:             "CODIGO:\nGet-Date\nEXPLICACION:\nMuestra la fecha actual.",
			codigoEsperado:      "Get-Date",
			explicacionEsperada: "Muestra la fecha actual.",
		},
		{
			nombre:              "codigo vacio por peticion riesgosa",
			entrada:             "CODIGO:\n\nEXPLICACION:\nNo genero scripts que borren carpetas del sistema.",
			codigoEsperado:      "",
			explicacionEsperada: "No genero scripts que borren carpetas del sistema.",
		},
		{
			nombre:              "sin marcadores: se descarta el codigo por seguridad",
			entrada:             "Claro, aca tenes un script: Get-Date",
			codigoEsperado:      "",
			explicacionEsperada: "Claro, aca tenes un script: Get-Date",
		},
		{
			nombre:         "marcadores en orden invertido: se descarta por seguridad",
			entrada:        "EXPLICACION:\nAlgo\nCODIGO:\nGet-Date",
			codigoEsperado: "",
		},
		{
			nombre:              "codigo multilinea",
			entrada:             "CODIGO:\nGet-ChildItem\nSort-Object Length\nEXPLICACION:\nLista archivos ordenados.",
			codigoEsperado:      "Get-ChildItem\nSort-Object Length",
			explicacionEsperada: "Lista archivos ordenados.",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			codigo, explicacion, err := parsearRespuestaCodigo(c.entrada)
			if err != nil {
				t.Fatalf("no se esperaba error: %v", err)
			}
			if codigo != c.codigoEsperado {
				t.Errorf("codigo = %q, esperaba %q", codigo, c.codigoEsperado)
			}
			if c.explicacionEsperada != "" && explicacion != c.explicacionEsperada {
				t.Errorf("explicacion = %q, esperaba %q", explicacion, c.explicacionEsperada)
			}
		})
	}
}