package core

import (
	"net"
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

// iaDesarrolloFake simula la IA de desarrollo devolviendo respuestas en orden
// y contando las llamadas, para probar el ciclo iterativo de verificación.
type iaDesarrolloFake struct {
	llamadas   int
	respuestas []string
	disponible bool
}

func (f *iaDesarrolloFake) Disponible() bool { return f.disponible }

func (f *iaDesarrolloFake) ConsultarDesarrollo(peticion string) (string, string, error) {
	if len(f.respuestas) == 0 {
		return "", "sin respuesta", nil
	}
	r := f.respuestas[0]
	f.respuestas = f.respuestas[1:]
	f.llamadas++
	return r, "explicacion de la IA", nil
}

func mainGoRoto() string {
	return "ARCHIVO: main.go\nCONTENIDO:\npackage main\n\nfunc main( {\nEXPLICACION:\nrompe a proposito"
}

func mainGoValido() string {
	return "ARCHIVO: main.go\nCONTENIDO:\npackage main\n\nfunc main() {}\nEXPLICACION:\nversion corregida"
}

func TestMejorarProyectoIteraHastaCorregir(t *testing.T) {
	dir := t.TempDir()
	fake := &iaDesarrolloFake{disponible: true}
	h := NewHands(HandsOpciones{WorkspaceRoot: dir, DesarrolladorIA: fake})
	resp := h.crearProyectoWeb("crear proyecto web prueba")
	if !strings.Contains(resp, "creado y compilando") {
		t.Fatalf("no se creó el proyecto: %s", resp)
	}

	// Primero devuelve código roto, después la corrección válida.
	fake.respuestas = []string{mainGoRoto(), mainGoValido()}
	resp = h.mejorarProyecto("mejorar el proyecto prueba agregá un contador")
	if !strings.Contains(resp, "tras 2 intentos") {
		t.Errorf("se esperaba éxito tras 2 intentos, obtuve: %s", resp)
	}
	if fake.llamadas != 2 {
		t.Errorf("la IA debió ser llamada 2 veces, fue %d", fake.llamadas)
	}
	if _, err := os.Stat(filepath.Join(dir, "prueba", ".git")); err != nil {
		t.Errorf("se esperaba un checkpoint de git tras el éxito: %v", err)
	}
}

func TestMejorarProyectoRindeTrasMaxIntentos(t *testing.T) {
	dir := t.TempDir()
	fake := &iaDesarrolloFake{disponible: true}
	h := NewHands(HandsOpciones{WorkspaceRoot: dir, DesarrolladorIA: fake})
	resp := h.crearProyectoWeb("crear proyecto web prueba")
	if !strings.Contains(resp, "creado y compilando") {
		t.Fatalf("no se creó el proyecto: %s", resp)
	}

	fake.respuestas = []string{mainGoRoto(), mainGoRoto(), mainGoRoto()}
	resp = h.mejorarProyecto("mejorar el proyecto prueba agregá un contador")
	if !strings.Contains(resp, "no compila tras 3 intentos") {
		t.Errorf("se esperaba rendición tras 3 intentos, obtuve: %s", resp)
	}
	if fake.llamadas != 3 {
		t.Errorf("la IA debió ser llamada 3 veces, fue %d", fake.llamadas)
	}
}

func TestMejorarProyectoBloqueaFueraDelProyecto(t *testing.T) {
	dir := t.TempDir()
	fake := &iaDesarrolloFake{disponible: true}
	h := NewHands(HandsOpciones{WorkspaceRoot: dir, DesarrolladorIA: fake})
	resp := h.crearProyectoWeb("crear proyecto web prueba")
	if !strings.Contains(resp, "creado y compilando") {
		t.Fatalf("no se creó el proyecto: %s", resp)
	}

	fake.respuestas = []string{"ARCHIVO: ../afuera.txt\nCONTENIDO:\nevil\nEXPLICACION:\ntest"}
	resp = h.mejorarProyecto("mejorar el proyecto prueba")
	if !strings.Contains(resp, "bloqueé") {
		t.Errorf("se esperaba bloqueo de escritura fuera del proyecto: %s", resp)
	}
}

func TestEscribirMejoraValidaExtensiones(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "proyecto")

	msg := escribirMejora(ruta, "archivo.exe", "binario")
	if msg == "" {
		t.Error("escribirMejora debería bloquear .exe")
	}
	msg = escribirMejora(ruta, "app.js", "console.log('ok');")
	if msg != "" {
		t.Errorf("escribirMejora debería aceptar .js: %s", msg)
	}
	datos, err := os.ReadFile(filepath.Join(ruta, "app.js"))
	if err != nil || string(datos) != "console.log('ok');" {
		t.Errorf("no se escribió app.js: %v", err)
	}
}

func TestEscribirMejoraBloqueaPathTraversal(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "proyecto")
	msg := escribirMejora(ruta, "../fuera.go", "package main")
	if msg == "" {
		t.Error("escribirMejora debería bloquear path traversal")
	}
	if _, err := os.Stat(filepath.Join(dir, "fuera.go")); err == nil {
		t.Error("el archivo no debería existir fuera del proyecto")
	}
}

func TestPuertoOcupado(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no pude abrir listener: %v", err)
	}
	defer ln.Close()
	puerto := ln.Addr().(*net.TCPAddr).Port

	if !puertoOcupado(puerto) {
		t.Errorf("puerto %d debería estar ocupado", puerto)
	}
	if puertoOcupado(65400) {
		t.Errorf("puerto alto probablemente libre reportó ocupado")
	}
}

func TestContextoProyecto(t *testing.T) {
	ruta := t.TempDir()
	os.MkdirAll(filepath.Join(ruta, "frontend"), 0o755)
	os.WriteFile(filepath.Join(ruta, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(ruta, "frontend", "index.html"), []byte("<h1>Hola</h1>"), 0o644)
	os.MkdirAll(filepath.Join(ruta, ".git"), 0o755)
	os.WriteFile(filepath.Join(ruta, ".git", "config"), []byte("ignorado"), 0o644)

	ctx := contextoProyecto(ruta)
	if !strings.Contains(ctx, "main.go") {
		t.Errorf("contextoProyecto debería listar main.go")
	}
	if !strings.Contains(ctx, "index.html") {
		t.Errorf("contextoProyecto debería listar index.html")
	}
	if strings.Contains(ctx, "=== .git") || strings.Contains(ctx, "ignorado") {
		t.Errorf("contextoProyecto no debería incluir .git")
	}
	if !strings.Contains(ctx, "package main") {
		t.Errorf("contextoProyecto debería incluir el contenido de main.go")
	}
}
