package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizarNombreProyecto(t *testing.T) {
	casos := map[string]string{
		"Mi App Web":  "mi-app-web",
		"Panel  2026": "panel--2026",
		"Stock":       "stock",
		"  Hola_123 ": "hola_123",
		"Panel!":      "panel",
	}
	for entrada, esperado := range casos {
		if got := sanitizarNombreProyecto(entrada); got != esperado {
			t.Errorf("sanitizarNombreProyecto(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}

func TestPlantillasProyecto(t *testing.T) {
	files := plantillasProyecto("panel")
	esperados := []string{"go.mod", "main.go", "frontend/index.html", "frontend/style.css", "frontend/app.js", "README.md"}
	for _, f := range esperados {
		if _, ok := files[f]; !ok {
			t.Errorf("falta archivo plantilla %q", f)
		}
	}
	for _, f := range esperados {
		if strings.Contains(files[f], "NOMBRE") {
			t.Errorf("%s todavía contiene el placeholder NOMBRE", f)
		}
	}
	if !strings.Contains(files["main.go"], "package main") {
		t.Errorf("main.go no parece código Go válido")
	}
	if !strings.Contains(files["frontend/app.js"], "'/estado'") {
		t.Errorf("app.js no parece JS funcional")
	}
}

func TestExtraerArchivoDesarrollo(t *testing.T) {
	respuesta := "ARCHIVO: frontend/app.js\nCONTENIDO:\nvar x = 1;\nEXPLICACION:\nAgrega un contador."
	archivo, contenido := extraerArchivoDesarrollo(respuesta)
	if archivo != "frontend/app.js" {
		t.Errorf("archivo = %q, esperado frontend/app.js", archivo)
	}
	if !strings.Contains(contenido, "var x = 1") {
		t.Errorf("contenido = %q", contenido)
	}

	_, raro := extraerArchivoDesarrollo("respuesta sin formato")
	if !strings.Contains(raro, "sin formato") {
		t.Errorf("fallback = %q", raro)
	}
}

func TestExtraerNombreYFeatureProyecto(t *testing.T) {
	n, f := extraerNombreYFeatureProyecto("agregá un contador al proyecto panel")
	if n != "panel" {
		t.Errorf("nombre = %q, esperado panel", n)
	}
	if !strings.Contains(f, "contador") {
		t.Errorf("feature = %q", f)
	}

	n2, f2 := extraerNombreYFeatureProyecto("mejorar el proyecto stock")
	if n2 != "stock" {
		t.Errorf("nombre2 = %q, esperado stock", n2)
	}
	if f2 == "" {
		t.Errorf("feature2 no debería estar vacía: %q", f2)
	}
}

func TestCrearProyectoWebEndToEnd(t *testing.T) {
	dir := t.TempDir()
	h := NewHands(HandsOpciones{WorkspaceRoot: dir})
	resp := h.crearProyectoWeb("crear proyecto web prueba")
	if !strings.Contains(resp, "creado y compilando") {
		t.Fatalf("respuesta inesperada: %s", resp)
	}
	ruta := filepath.Join(dir, "prueba")
	for _, f := range []string{"main.go", "go.mod", "frontend/index.html", "frontend/app.js", "prueba.exe"} {
		if _, err := os.Stat(filepath.Join(ruta, f)); err != nil {
			t.Errorf("falta %s: %v", f, err)
		}
	}
	resp2 := h.crearProyectoWeb("crear proyecto web prueba")
	if !strings.Contains(resp2, "Ya existe") {
		t.Errorf("segunda creación debería rechazarse: %s", resp2)
	}
	resp3 := h.listarProyectos()
	if !strings.Contains(resp3, "prueba") {
		t.Errorf("listarProyectos no muestra el proyecto: %s", resp3)
	}
}
