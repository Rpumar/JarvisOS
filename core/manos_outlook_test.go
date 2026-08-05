package core

import (
	"strings"
	"testing"
	"time"
)

func TestBuildScriptSincronizarOutlook(t *testing.T) {
	inicio := time.Date(2026, 8, 4, 15, 0, 0, 0, time.Local).Format(time.RFC3339)
	eventos := []EventoAgenda{
		{ID: 1, Titulo: "reunión con el cliente", Inicio: inicio},
		{ID: 2, Titulo: "cita del doctor O'Brien", Inicio: inicio},
	}
	script := buildScriptSincronizarOutlook(eventos)

	if !strings.Contains(script, "Outlook.Application") {
		t.Fatalf("el script debe crear el objeto COM Outlook")
	}
	if !strings.Contains(script, "GetDefaultFolder(") {
		t.Fatalf("el script debe abrir el calendario")
	}
	if !strings.Contains(script, "reunión con el cliente") {
		t.Fatalf("el script no incluye el título del primer evento")
	}
	if !strings.Contains(script, "O''Brien") {
		t.Fatalf("el script debe escapar comillas simples: %q", script)
	}
	if !strings.Contains(script, "[datetime]'2026-08-04T15:00:00'") {
		t.Fatalf("el script debe usar la fecha de inicio ISO")
	}
}

func TestParseTurnosOutlook(t *testing.T) {
	salida := "Reunión quincenal|2026-08-04 15:00\n\nJuntada dueños|2026-08-04 16:00\n"
	got := parseTurnosOutlook(salida)
	if len(got) != 2 {
		t.Fatalf("esperaba 2 turnos, obtuve %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "Reunión quincenal") {
		t.Fatalf("turno 1 mal: %q", got[0])
	}
	if !strings.Contains(got[0], "04/08") {
		t.Fatalf("turno 1 sin fecha: %q", got[0])
	}
}

func TestTipoOutlookComando(t *testing.T) {
	c := NuevoClasificador()

	casos := []struct {
		entrada   string
		esperado  string
	}{
		{"sincronizá la agenda con outlook", "outlook"},
		{"exportá la agenda a outlook", "outlook"},
		{"leé mis próximos eventos de outlook", "outlook"},
		{"quÉ tengo en outlook", "outlook"},
		{"abrir outlook", ""},
		{"qué tengo hoy", "agenda"},
	}
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}