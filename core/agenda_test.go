package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGestorAgenda(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorAgenda(filepath.Join(dir, "agenda.json"))

	ahora := time.Date(2026, 8, 2, 10, 0, 0, 0, time.Local)
	mañana10 := time.Date(2026, 8, 3, 10, 0, 0, 0, time.Local)
	mañana18 := time.Date(2026, 8, 3, 18, 0, 0, 0, time.Local)

	id1, err := g.Agregar("reunión con el equipo", mañana10.Format(time.RFC3339), "")
	if err != nil {
		t.Fatalf("Agregar falló: %v", err)
	}
	id2, _ := g.Agregar("gimnasio", mañana18.Format(time.RFC3339), "")
	if id2 <= id1 {
		t.Fatalf("los IDs deben incrementarse: %d >= %d", id2, id1)
	}
	if len(g.Listar()) != 2 {
		t.Fatalf("esperaba 2 eventos, hay %d", len(g.Listar()))
	}

	delDia := g.EventosEntre(hoyMedianoche(ahora), hoyMedianoche(ahora).Add(24*time.Hour))
	if len(delDia) != 0 {
		t.Fatalf("hoy no debería haber eventos, hay %d", len(delDia))
	}

	delManana := g.EventosEntre(hoyMedianoche(mañana10), hoyMedianoche(mañana10).Add(24*time.Hour))
	if len(delManana) != 2 {
		t.Fatalf("mañana debería haber 2 eventos, hay %d", len(delManana))
	}

	proximos := g.Proximos(5, ahora)
	if len(proximos) != 2 || proximos[0].Titulo != "reunión con el equipo" {
		t.Fatalf("Proximos falló: %+v", proximos)
	}

	n, borrados := g.Cancelar("gimnasio")
	if n != 1 || len(borrados) != 1 {
		t.Fatalf("Cancelar falló: n=%d borrados=%v", n, borrados)
	}
	if len(g.Listar()) != 1 {
		t.Fatalf("esperaba 1 evento tras cancelar, hay %d", len(g.Listar()))
	}

	g2 := NuevoGestorAgenda(filepath.Join(dir, "agenda.json"))
	if len(g2.Listar()) != 1 {
		t.Fatal("los eventos no persistieron entre instancias")
	}
}

func TestExtraerEvento(t *testing.T) {
	ahora := time.Date(2026, 8, 2, 10, 0, 0, 0, time.Local)

	titulo, inicio, _, ok := extraerEvento("agendá una reunión con el cliente mañana a las 15", ahora)
	if !ok {
		t.Fatal("extraerEvento no reconoció el comando")
	}
	if titulo != "reunión con el cliente" {
		t.Fatalf("título inesperado: %q", titulo)
	}
	inicioT, _ := time.Parse(time.RFC3339, inicio)
	want := time.Date(2026, 8, 3, 15, 0, 0, 0, ahora.Location())
	if !inicioT.Equal(want) {
		t.Fatalf("inicio inesperado: %v, quería %v", inicioT, want)
	}

	_, _, _, ok = extraerEvento("qué tengo hoy", ahora)
	if ok {
		t.Fatal("'qué tengo hoy' no debería agendar")
	}

	_, _, _, ok = extraerEvento("agendá un evento en el pasado a las 8", ahora)
	if ok {
		t.Fatal("un evento en el pasado no debería agendarse")
	}
}

func TestManejarAgenda(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{agenda: NuevoGestorAgenda(filepath.Join(dir, "agenda.json"))}

	got := h.manejarAgenda("agendá una cita con el médico mañana a las 9")
	if !strings.Contains(got, "cita con el médico") {
		t.Fatalf("agendar falló: %q", got)
	}
	if len(h.agenda.Listar()) != 1 {
		t.Fatalf("esperaba 1 evento, hay %d", len(h.agenda.Listar()))
	}

	got = h.manejarAgenda("qué tengo mañana")
	if !strings.Contains(got, "cita con el médico") {
		t.Fatalf("listar del día falló: %q", got)
	}

	got = h.manejarAgenda("cancelá el evento cita")
	if !strings.Contains(got, "Cancelé") {
		t.Fatalf("cancelar falló: %q", got)
	}
	if len(h.agenda.Listar()) != 0 {
		t.Fatalf("esperaba 0 eventos tras cancelar, hay %d", len(h.agenda.Listar()))
	}

	h2 := &Hands{}
	if got = h2.manejarAgenda("qué tengo hoy"); !strings.Contains(got, "No tengo calendario") {
		t.Fatalf("sin agenda debería avisar: %q", got)
	}
}
