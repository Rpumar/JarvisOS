package core

import (
	"strings"
	"testing"
	"time"
)

func TestFormatearHora(t *testing.T) {
	momento := time.Date(2026, time.July, 20, 14, 5, 0, 0, time.UTC)
	got := formatearHora(momento)
	esperado := "Son las 14:05, señor."
	if got != esperado {
		t.Errorf("formatearHora() = %q, esperaba %q", got, esperado)
	}
}

func TestFormatearFecha(t *testing.T) {
	casos := []struct {
		momento  time.Time
		esperado string
	}{
		{time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC), "Hoy es 20 de julio de 2026, señor."},
		{time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), "Hoy es 1 de enero de 2026, señor."},
		{time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC), "Hoy es 31 de diciembre de 2026, señor."},
	}
	for _, c := range casos {
		if got := formatearFecha(c.momento); got != c.esperado {
			t.Errorf("formatearFecha(%v) = %q, esperaba %q", c.momento, got, c.esperado)
		}
	}
}

// NOTA HONESTA: el resto de hands.go (abrirApp, volumen, capturarPantalla,
// buscarEnGoogle, etc.) llama directamente a exec.Command/PowerShell y NO
// se testea acá. Son wrappers finos sobre el sistema operativo — probarlos
// de verdad requiere Windows real, no un test unitario. formatearHora/
// formatearFecha se separaron de sus métodos justamente para que esa
// porción de lógica, que sí es pura, quedara cubierta. Lo mismo aplica a
// esProcesoProtegido (Fase 4): es lógica pura de decisión, separable de la
// llamada real a taskkill, así que sí se testea.

func TestEsProcesoProtegido(t *testing.T) {
	protegidos := []string{
		"winlogon.exe", "csrss.exe", "services.exe", "lsass.exe",
		"smss.exe", "wininit.exe", "svchost.exe", "explorer.exe",
	}
	for _, p := range protegidos {
		if !esProcesoProtegido(p) {
			t.Errorf("esProcesoProtegido(%q) = false, esperaba true (proceso crítico)", p)
		}
	}

	noProtegidos := []string{"chrome.exe", "spotify.exe", "code.exe", "notepad.exe", "calc.exe"}
	for _, p := range noProtegidos {
		if esProcesoProtegido(p) {
			t.Errorf("esProcesoProtegido(%q) = true, esperaba false (no es un proceso crítico)", p)
		}
	}
}

func TestEsProcesoProtegido_NoDistingueMayusculas(t *testing.T) {
	// La entrada viene de voz reconocida y pasa por strings.ToLower en
	// RunCommand antes de llegar acá, pero se prueba explícito para no
	// depender de ese orden como suposición implícita.
	if !esProcesoProtegido("EXPLORER.EXE") {
		t.Error("esProcesoProtegido debería ser insensible a mayúsculas/minúsculas")
	}
}

func TestEjecutarConTimeout_Aborta(t *testing.T) {
	viejo := TiempoLimiteComando
	TiempoLimiteComando = 300 * time.Millisecond
	defer func() { TiempoLimiteComando = viejo }()

	_, err := ejecutarConTimeout("powershell", "-NoProfile", "-Command", "Start-Sleep -Seconds 5")
	if err == nil || !strings.Contains(err.Error(), "límite") {
		t.Fatalf("un comando que no termina debe abortarse con error de tiempo límite, obtuve: %v", err)
	}
}

func TestEjecutarConTimeout_ComandoRapidoOk(t *testing.T) {
	viejo := TiempoLimiteComando
	TiempoLimiteComando = 30 * time.Second
	defer func() { TiempoLimiteComando = viejo }()

	salida, err := ejecutarConTimeout("cmd", "/C", "echo hola")
	if err != nil {
		t.Fatalf("un comando rápido no debe fallar: %v", err)
	}
	if !strings.Contains(string(salida), "hola") {
		t.Fatalf("salida inesperada: %q", string(salida))
	}
}
