package core

import (
	"path/filepath"
	"testing"
)

func TestGestorPerfil_SeleccionYPersistencia(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "perfil.json")
	g := NuevoGestorPerfil(ruta)
	if g.Activo() != PerfilDueno {
		t.Fatalf("activo inicial = %s, esperaba dueno", g.Activo())
	}

	if !g.AgregarUsuario("Ana", "Ventas", "admin") {
		t.Fatal("no se pudo agregar a Ana")
	}
	if g.AgregarUsuario("Ana", "Ventas", "admin") {
		t.Fatal("Ana ya existía; debía devolver false")
	}
	if !g.AgregarUsuario("Pedro", "", "empleado") {
		t.Fatal("no se pudo agregar a Pedro")
	}
	if !g.Seleccionar("ana") {
		t.Fatal("no se pudo seleccionar a Ana (case-insensitive)")
	}
	if g.Activo() != "Ana" {
		t.Fatalf("activo = %s, esperaba Ana", g.Activo())
	}
	if g.ActivoRol() != PerfilAdmin {
		t.Fatalf("rol activo = %s, esperaba admin", g.ActivoRol())
	}

	g2 := NuevoGestorPerfil(ruta)
	if g2.Activo() != "Ana" {
		t.Fatalf("tras recargar activo = %s, esperaba Ana", g2.Activo())
	}
	if len(g2.Usuarios()) != 2 {
		t.Fatalf("usuarios = %d, esperaba 2", len(g2.Usuarios()))
	}
}

func TestGestorPerfil_NivelesDirectos(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))

	if !g.Seleccionar("admin") {
		t.Fatal("no se pudo seleccionar el nivel admin")
	}
	if g.Activo() != PerfilAdmin {
		t.Fatalf("activo = %s, esperaba admin", g.Activo())
	}
	if g.ActivoRol() != PerfilAdmin {
		t.Fatalf("rol = %s, esperaba admin", g.ActivoRol())
	}

	if !g.Seleccionar("el dueno") {
		t.Fatal("no se pudo seleccionar el nivel dueno")
	}
	if g.Activo() != PerfilDueno {
		t.Fatalf("activo = %s, esperaba dueno", g.Activo())
	}
}

func TestGestorPerfil_LimitePuestos(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	g.LimitePuestos = 2
	if !g.AgregarUsuario("Ana", "", "admin") {
		t.Fatal("no se pudo agregar a Ana (puesto 1)")
	}
	if !g.AgregarUsuario("Pedro", "", "empleado") {
		t.Fatal("no se pudo agregar a Pedro (puesto 2)")
	}
	if g.AgregarUsuario("Luis", "", "empleado") {
		t.Fatal("el puesto 3 debe rechazarse con límite de 2")
	}
	if !g.LimiteAlcanzado() {
		t.Fatal("LimiteAlcanzado debe ser true con 2/2")
	}
	if len(g.Usuarios()) != 2 {
		t.Fatalf("usuarios = %d, esperaba 2", len(g.Usuarios()))
	}
	// Actualizar a alguien existente no debe chocar con el límite.
	if g.AgregarUsuario("Ana", "Ventas", "admin") {
		t.Fatal("Ana ya existía; debía devolver false (actualización)")
	}
	if len(g.Usuarios()) != 2 {
		t.Fatalf("usuarios tras actualizar = %d, esperaba 2", len(g.Usuarios()))
	}
}

func TestGestorPerfil_SinLimitePermiteTodos(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	g.LimitePuestos = 0
	if !g.AgregarUsuario("Ana", "", "admin") {
		t.Fatal("sin límite debe aceptar a Ana")
	}
	if g.LimiteAlcanzado() {
		t.Fatal("sin límite nunca debe reportar alcanzado")
	}
	if libres := g.PuestosLibres(); libres != -1 {
		t.Fatalf("PuestosLibres = %d, esperaba -1 (sin límite)", libres)
	}
}

func TestBrain_PerfilLimitePuestosPorVoz(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	g.LimitePuestos = 1
	b := NewBrain(&manosFalsas{}, BrainOpciones{Perfil: g})

	got := b.Process("agregá al usuario Ana como admin")
	if !contains(got, "Ana") {
		t.Fatalf("primer registro = %q, esperaba mención a Ana", got)
	}

	got = b.Process("agregá al usuario Pedro como empleado")
	if !contains(got, "licencia") && !contains(got, "puestos") {
		t.Fatalf("segundo registro con límite = %q, esperaba aviso de licencia/puestos", got)
	}
	if len(g.Usuarios()) != 1 {
		t.Fatalf("usuarios = %d, esperaba 1", len(g.Usuarios()))
	}
}

func TestGestorPerfil_EliminarVuelveADueno(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	g.AgregarUsuario("Ana", "", "admin")
	g.Seleccionar("Ana")
	if !g.Eliminar("Ana") {
		t.Fatal("no se pudo eliminar a Ana")
	}
	if g.Activo() != PerfilDueno {
		t.Fatalf("activo tras eliminar = %s, esperaba dueno", g.Activo())
	}
	if len(g.Usuarios()) != 0 {
		t.Fatalf("usuarios = %d, esperaba 0", len(g.Usuarios()))
	}
}

func TestBrain_PerfilPorVoz(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	b := NewBrain(&manosFalsas{}, BrainOpciones{Perfil: g})

	got := b.Process("qué perfil está activo")
	if !contains(got, "dueno") {
		t.Errorf("respuesta de perfil activo = %q, esperaba mención a dueno", got)
	}

	if got = b.Process("operá como admin"); !contains(got, "admin") {
		t.Errorf("selección de admin = %q, esperaba mención a admin", got)
	}

	if got = b.Process("qué perfil está activo"); !contains(got, "admin") {
		t.Errorf("tras seleccionar admin = %q, esperaba mención a admin", got)
	}
}

func TestBrain_PerfilRegistrarYListar(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	b := NewBrain(&manosFalsas{}, BrainOpciones{Perfil: g})

	got := b.Process("agregá al usuario Ana como admin del área Ventas")
	if !contains(got, "Ana") || !contains(got, "admin") {
		t.Errorf("registro = %q, esperaba mención a Ana y admin", got)
	}
	if len(g.Usuarios()) != 1 {
		t.Fatalf("usuarios = %d, esperaba 1", len(g.Usuarios()))
	}

	if got = b.Process("qué perfiles hay"); !contains(got, "Ana") {
		t.Errorf("listado = %q, esperaba mención a Ana", got)
	}

	if got = b.Process("borrá al usuario Ana"); !contains(got, "Ana") {
		t.Errorf("borrado = %q, esperaba mención a Ana", got)
	}
	if len(g.Usuarios()) != 0 {
		t.Fatalf("usuarios tras borrar = %d, esperaba 0", len(g.Usuarios()))
	}
}
