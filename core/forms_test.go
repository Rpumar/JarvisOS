package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGestorFormularios_CRUD(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "formularios.json")
	g := NuevoGestorFormularios(ruta)

	if len(g.Listar()) != 0 {
		t.Fatal("debería arrancar vacío")
	}

	g.Agregar(Formulario{Nombre: "Factura", URL: "https://sitio.com/factura"})
	if _, ok := g.Obtener("factura"); !ok {
		t.Fatal("no se encontró la plantilla Factura (case-insensitive)")
	}

	f, ok := g.AgregarCampo("Factura", "email", "pedidos@miempresa.com")
	if !ok {
		t.Fatal("no se pudo agregar el campo a Factura")
	}
	if len(f.Campos) != 1 || f.Campos[0].Nombre != "email" {
		t.Fatalf("campos inesperados: %+v", f.Campos)
	}

	// AgregarCampo no duplica.
	g.AgregarCampo("Factura", "email", "otro@x.com")
	f, _ = g.Obtener("Factura")
	if len(f.Campos) != 1 {
		t.Fatalf("AgregarCampo duplicó el campo: %+v", f.Campos)
	}

	// CambiarCampo actualiza.
	if _, ok := g.CambiarCampo("factura", "email", "nuevo@x.com"); !ok {
		t.Fatal("no se pudo cambiar el campo")
	}
	f, _ = g.Obtener("Factura")
	if f.Campos[0].Valor != "nuevo@x.com" {
		t.Fatalf("valor no actualizado: %+v", f.Campos[0])
	}

	// Persistencia.
	g2 := NuevoGestorFormularios(ruta)
	f2, ok := g2.Obtener("Factura")
	if !ok || len(f2.Campos) != 1 || f2.URL == "" {
		t.Fatalf("persistencia falló: %+v", f2)
	}

	// Eliminar.
	if !g2.Eliminar("factura") {
		t.Fatal("no se pudo eliminar Factura")
	}
	if _, ok := g2.Obtener("Factura"); ok {
		t.Fatal("Factura sigue existiendo tras eliminar")
	}
	if len(g2.Listar()) != 0 {
		t.Fatal("lista debería estar vacía tras eliminar")
	}
}

func TestGestorFormularios_AplicarSobreInexistente(t *testing.T) {
	g := NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))
	if _, ok := g.AgregarCampo("NoExiste", "x", "y"); ok {
		t.Fatal("AgregarCampo debería fallar con plantilla inexistente")
	}
	if g.Eliminar("NoExiste") {
		t.Fatal("Eliminar debería fallar con plantilla inexistente")
	}
}

func TestSepararNombreYURL(t *testing.T) {
	nombre, url := separarNombreYURL("factura para https://sitio.com/form")
	if nombre != "factura" || url != "https://sitio.com/form" {
		t.Fatalf("got (%q, %q)", nombre, url)
	}
	nombre, url = separarNombreYURL("factura")
	if nombre != "factura" || url != "" {
		t.Fatalf("got (%q, %q)", nombre, url)
	}
}

func TestSplitCampoValorFormulario(t *testing.T) {
	campo, valor, formulario := splitCampoValorFormulario("email con pedidos@miempresa.com a factura")
	if campo != "email" || valor != "pedidos@miempresa.com" || formulario != "factura" {
		t.Fatalf("got (%q, %q, %q)", campo, valor, formulario)
	}

	if _, _, f := splitCampoValorFormulario("sin formato"); f != "" {
		t.Fatal("debería fallar sin ' a '")
	}
}

func TestManejarCrearFormulario(t *testing.T) {
	h := &Hands{formularios: NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))}

	got := h.manejarCrearFormulario("creá un formulario factura para facturacion.com")
	if !contains(got, "factura") {
		t.Errorf("respuesta inesperada: %q", got)
	}
	f, ok := h.formularios.Obtener("factura")
	if !ok {
		t.Fatal("no se guardó la plantilla factura")
	}
	if f.URL != "https://facturacion.com" {
		t.Errorf("URL = %q, esperaba https://facturacion.com", f.URL)
	}
}

func TestManejarAgregarCampo(t *testing.T) {
	h := &Hands{formularios: NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))}
	h.formularios.Agregar(Formulario{Nombre: "Factura"})

	got := h.manejarAgregarCampo("agregá el campo email con pedidos@x.com a factura")
	if !contains(got, "email") {
		t.Errorf("respuesta inesperada: %q", got)
	}
	f, _ := h.formularios.Obtener("factura")
	if len(f.Campos) != 1 || f.Campos[0].Valor != "pedidos@x.com" {
		t.Fatalf("campo no cargado: %+v", f.Campos)
	}

	// Campo a plantilla inexistente.
	got = h.manejarAgregarCampo("agregá el campo x con y a noexiste")
	if !contains(got, "No encontré") {
		t.Errorf("debería reportar plantilla inexistente: %q", got)
	}
}

func TestManejarFormulario_Listar(t *testing.T) {
	h := &Hands{formularios: NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))}
	h.formularios.Agregar(Formulario{Nombre: "Factura"})
	h.formularios.Agregar(Formulario{Nombre: "Presupuesto"})

	got := h.manejarFormulario("qué formularios tengo")
	if !contains(got, "Factura") || !contains(got, "Presupuesto") {
		t.Errorf("listado inesperado: %q", got)
	}
}

func TestManejarFormulario_Borrar(t *testing.T) {
	h := &Hands{formularios: NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))}
	h.formularios.Agregar(Formulario{Nombre: "Factura"})

	got := h.manejarFormulario("borrá el formulario factura")
	if !contains(got, "eliminado") {
		t.Errorf("borrado inesperado: %q", got)
	}
	if len(h.formularios.Listar()) != 0 {
		t.Fatal("la plantilla sigue existiendo")
	}
}

func TestManejarFormulario_AutoNoEncontrado(t *testing.T) {
	h := &Hands{formularios: NuevoGestorFormularios(filepath.Join(t.TempDir(), "formularios.json"))}
	got := h.manejarFormulario("rellená el formulario noexiste")
	if !contains(got, "No encontré") {
		t.Errorf("respuesta inesperada: %q", got)
	}
}

func TestEscaparSendKeys(t *testing.T) {
	got := escaparSendKeys("a+b")
	if !strings.Contains(got, "{+}") {
		t.Errorf("el + debe escaparse: %q", got)
	}
	got = escaparSendKeys("100%")
	if !strings.Contains(got, "{%}") {
		t.Errorf("el %% debe escaparse: %q", got)
	}
	got = escaparSendKeys("hasta las 5")
	if got != "hasta las 5" {
		t.Errorf("texto plano no debe alterarse: %q", got)
	}
}

func TestNormalizarSitio(t *testing.T) {
	if got := normalizarSitio("facturacion.com"); got != "https://facturacion.com" {
		t.Errorf("got %q", got)
	}
	if got := normalizarSitio("https://ya.com/x"); got != "https://ya.com/x" {
		t.Errorf("got %q", got)
	}
	if got := normalizarSitio("sitio"); got != "https://sitio.com" {
		t.Errorf("got %q", got)
	}
}
