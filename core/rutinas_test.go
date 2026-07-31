package core

import (
	"path/filepath"
	"testing"
)

func TestRutinaManager(t *testing.T) {
	dir := t.TempDir()
	m := NuevoRutinaManager(filepath.Join(dir, "rutinas.json"))
	m.Crear("trabajo", []string{"abrir chrome", "abrir spotify"})

	pasos, ok := m.Obtener("trabajo")
	if !ok || len(pasos) != 2 {
		t.Fatalf("esperaba rutina con 2 pasos, ok=%v pasos=%v", ok, pasos)
	}

	m2 := NuevoRutinaManager(filepath.Join(dir, "rutinas.json"))
	if _, ok := m2.Obtener("trabajo"); !ok {
		t.Fatal("la rutina no persistió entre instancias")
	}

	if !m.Borrar("trabajo") {
		t.Fatal("esperaba borrar la rutina")
	}
	if _, ok := m.Obtener("trabajo"); ok {
		t.Fatal("la rutina sigue existiendo tras borrarla")
	}
}

func TestDividirPasos(t *testing.T) {
	pasos := dividirPasos("abrir chrome y abrir spotify")
	if len(pasos) != 2 || pasos[0] != "abrir chrome" || pasos[1] != "abrir spotify" {
		t.Fatalf("pasos inesperados: %v", pasos)
	}

	pasos = dividirPasos("abrí chrome, abrí spotify y abrí youtube")
	if len(pasos) != 3 {
		t.Fatalf("esperaba 3 pasos: %v", pasos)
	}
	if pasos[0] != "abrir chrome" {
		t.Fatalf("el verbo no se normalizó: %v", pasos[0])
	}
}

func TestExtraerNombreRutina(t *testing.T) {
	if n := extraerNombreRutina("ejecutar rutina trabajo"); n != "trabajo" {
		t.Fatalf("nombre inesperado: %q", n)
	}
	if n := extraerNombreRutina("corré la rutina gaming"); n != "gaming" {
		t.Fatalf("nombre inesperado: %q", n)
	}
	if n := extraerNombreRutina("rutina"); n != "" {
		t.Fatalf("esperaba vacío: %q", n)
	}
}
