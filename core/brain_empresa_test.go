package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nuevoBrainEmpresa(t *testing.T) (*Brain, *GestorEmpresa) {
	t.Helper()
	dir := t.TempDir()
	g := NuevoGestorEmpresa(filepath.Join(dir, "empresa.json"))
	b := NewBrain(&manosFalsas{}, BrainOpciones{Empresa: g})
	return b, g
}

func TestManejarEmpresa_ConsultaVacia(t *testing.T) {
	b, _ := nuevoBrainEmpresa(t)
	got := b.Process("qué sabés de mi empresa")
	if !strings.Contains(got, "Todavía no tengo cargado") {
		t.Errorf("esperaba guía de perfil vacío, obtuve %q", got)
	}
}

func TestManejarEmpresa_CargarCampos(t *testing.T) {
	b, g := nuevoBrainEmpresa(t)
	casos := []struct {
		cmd  string
		ver  func(p PerfilEmpresa) bool
	}{
		{"mi empresa se llama Panadería La Espiga", func(p PerfilEmpresa) bool { return p.Nombre == "Panadería La Espiga" }},
		{"mi rubro es Alimentos", func(p PerfilEmpresa) bool { return p.Rubro == "Alimentos" }},
		{"somos 5 empleados", func(p PerfilEmpresa) bool { return p.Tamano == "5 empleados" }},
		{"mi email es ventas@espiga.com", func(p PerfilEmpresa) bool { return p.ContactoMail == "ventas@espiga.com" }},
		{"el dueño se llama Juan", func(p PerfilEmpresa) bool { return p.ContactoDueno == "Juan" }},
	}
	for _, c := range casos {
		got := b.Process(c.cmd)
		if got == "" {
			t.Fatalf("comando %q no dio respuesta", c.cmd)
		}
		if !c.ver(g.Obtener()) {
			t.Errorf("comando %q no actualizó el perfil: %+v", c.cmd, g.Obtener())
		}
	}
}

func TestManejarEmpresa_AgregarYListar(t *testing.T) {
	b, g := nuevoBrainEmpresa(t)
	b.Process("mi producto principal es pan integral")
	b.Process("agregá un objetivo abrir un segundo local")
	b.Process("mis clientes son familias y comercios")
	p := g.Obtener()
	if len(p.Productos) != 1 || p.Productos[0] != "pan integral" {
		t.Errorf("productos = %v", p.Productos)
	}
	if len(p.Objetivos) != 1 || p.Objetivos[0] != "abrir un segundo local" {
		t.Errorf("objetivos = %v", p.Objetivos)
	}
	if len(p.Clientes) != 1 || p.Clientes[0] != "familias" {
		t.Errorf("clientes = %v (se corta en 'y')", p.Clientes)
	}
	got := b.Process("qué sabés de mi empresa")
	if !strings.Contains(got, "pan integral") || !strings.Contains(got, "abrir un segundo local") {
		t.Errorf("resumen incompleto: %q", got)
	}
}

func TestManejarEmpresa_BorrarItem(t *testing.T) {
	b, g := nuevoBrainEmpresa(t)
	b.Process("mi producto principal es pan integral")
	got := b.Process("borrá mi producto pan integral")
	if !strings.Contains(got, "Saqué") {
		t.Errorf("al borrar debería confirmar, obtuve %q", got)
	}
	if len(g.Obtener().Productos) != 0 {
		t.Errorf("deberían quedar 0 productos, hay %v", g.Obtener().Productos)
	}
}

func TestManejarEmpresa_NoInterfiereOtrosComandos(t *testing.T) {
	b, _ := nuevoBrainEmpresa(t)
	// El objetivo real es que empresa no se coma comandos ajenos a nivel de frase.
	if got, _ := b.manejarEmpresa("qué hora es"); got != "" {
		t.Errorf("'qué hora es' no debería atenderse como empresa, obtuve %q", got)
	}
	if got, _ := b.manejarEmpresa("hola señor"); got != "" {
		t.Errorf("frase sin contexto de empresa no debería atenderse, obtuve %q", got)
	}
}

func TestExtraerEmpleados(t *testing.T) {
	casos := map[string]string{
		"somos 5 empleados":         "5 empleados",
		"mi empresa tiene 12 empleados": "12 empleados",
		"trabajamos 3 empleados":     "3 empleados",
		"querés un café":            "",
	}
	for in, esperado := range casos {
		got, ok := extraerEmpleados(in, in)
		if esperado == "" {
			if ok {
				t.Errorf("extraerEmpleados(%q) = %q, esperaba no-ok", in, got)
			}
			continue
		}
		if !ok || got != esperado {
			t.Errorf("extraerEmpleados(%q) = %q/%v, esperaba %q", in, got, ok, esperado)
		}
	}
}

func TestGestorEmpresa_ArchivoReal(t *testing.T) {
	ruta := filepath.Join(os.TempDir(), "jarvisos_empresa_test.json")
	_ = os.Remove(ruta)
	defer os.Remove(ruta)
	g := NuevoGestorEmpresa(ruta)
	_ = g.SetCampo("nombre", "Test SRL")
	if g.Obtener().Nombre != "Test SRL" {
		t.Fatalf("no se guardó el nombre: %+v", g.Obtener())
	}
	g2 := NuevoGestorEmpresa(ruta)
	if g2.Obtener().Nombre != "Test SRL" {
		t.Errorf("no persistió entre instancias: %+v", g2.Obtener())
	}
}