package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAgregarFrasesNoRepetidas(t *testing.T) {
	got := agregarFrasesNoRepetidas([]string{"a", "b"}, []string{"b", "c", "a"})
	esperado := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, esperado) {
		t.Errorf("agregarFrasesNoRepetidas = %v, esperaba %v", got, esperado)
	}
}

func TestAgregarFrasesNoRepetidasVacio(t *testing.T) {
	got := agregarFrasesNoRepetidas(nil, []string{"x"})
	if len(got) != 1 || got[0] != "x" {
		t.Errorf("esperaba [x], obtuve %v", got)
	}
}

func TestGenerarFrasesCorta(t *testing.T) {
	got := generarFrases("hola mundo")
	if len(got) != 1 || got[0] != "hola mundo" {
		t.Errorf("frase corta debería quedar igual: %v", got)
	}
}

func TestGenerarFrasesLarga(t *testing.T) {
	frases := generarFrases("abre la calculadora y anota el total de la compra de hoy")
	if len(frases) < 5 {
		t.Errorf("esperaba varias frases, obtuve %d: %v", len(frases), frases)
	}
	visto := map[string]bool{}
	for _, f := range frases {
		if f == "" {
			t.Errorf("frase vacía generada")
		}
		visto[f] = true
	}
	if len(visto) < 5 {
		t.Errorf("esperaba al menos 5 frases únicas, obtuve %d", len(visto))
	}
}

func TestRegistroAprendizajePersistencia(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "aprendizaje.json")
	r := NuevoRegistroAprendizaje(ruta)
	r.Aprender([]string{"abrir spotify", "reproduce musica"}, "abrirApp(spotify)")

	// Recargar desde disco: el archivo debe existir y tener contenido.
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("no se guardó el archivo: %v", err)
	}
	r2 := NuevoRegistroAprendizaje(ruta)
	lista := r2.Listar()
	if len(lista) != 1 || lista[0].Comando != "abrirApp(spotify)" {
		t.Errorf("persistencia falló: %+v", lista)
	}
}

func TestRegistroAprendizajeAprenderDuplicado(t *testing.T) {
	r := NuevoRegistroAprendizaje(filepath.Join(t.TempDir(), "a.json"))
	r.Aprender([]string{"frase uno"}, "cmd1")
	r.Aprender([]string{"frase dos"}, "cmd1")

	lista := r.Listar()
	if len(lista) != 1 {
		t.Fatalf("duplicado no debería crear otro registro: %d", len(lista))
	}
	if len(lista[0].Frases) != 2 {
		t.Errorf("esperaba 2 frases, obtuve %d: %v", len(lista[0].Frases), lista[0].Frases)
	}
	if lista[0].Usos != 2 {
		t.Errorf("esperaba Usos=2, obtuve %d", lista[0].Usos)
	}
}

func TestRegistroAprendizajeBuscar(t *testing.T) {
	r := NuevoRegistroAprendizaje(filepath.Join(t.TempDir(), "b.json"))
	r.Aprender([]string{"abrir la calculadora"}, "abrirApp(calc)")

	casos := []struct {
		entrada  string
		esperado string
		ok       bool
	}{
		{"podés abrir la calculadora", "abrirApp(calc)", true},
		{"la calculadora", "abrirApp(calc)", true},
		{"contame un chiste", "", false},
	}
	for _, c := range casos {
		got, ok := r.Buscar(c.entrada)
		if ok != c.ok || (c.ok && got != c.esperado) {
			t.Errorf("Buscar(%q) = (%q, %v), esperaba (%q, %v)", c.entrada, got, ok, c.esperado, c.ok)
		}
	}
}

func TestRegistroAprendizajeOlvidar(t *testing.T) {
	r := NuevoRegistroAprendizaje(filepath.Join(t.TempDir(), "c.json"))
	r.Aprender([]string{"uno"}, "cmd1")
	r.Aprender([]string{"dos"}, "cmd2")

	if !r.Olvidar("cmd1") {
		t.Error("Olvidar debería devolver true para comando existente")
	}
	if r.Olvidar("cmd-no-existe") {
		t.Error("Olvidar debería devolver false para comando inexistente")
	}
	if len(r.Listar()) != 1 {
		t.Errorf("esperaba 1 registro, obtuve %d", len(r.Listar()))
	}
}

func TestRegistroAprendizajeArchivoCorrupto(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "corrupto.json")
	os.WriteFile(ruta, []byte("{esto-no-es-json"), 0o600)
	r := NuevoRegistroAprendizaje(ruta)
	if len(r.Listar()) != 0 {
		t.Errorf("archivo corrupto debería cargar vacío, obtuve %d", len(r.Listar()))
	}
}
