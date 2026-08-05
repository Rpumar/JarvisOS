package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNuevoGestorEmpresa_Vacia(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorEmpresa(filepath.Join(dir, "empresa.json"))
	if !g.Obtener().EstaVacia() {
		t.Fatal("un perfil recién creado debería estar vacío")
	}
	if got := g.Resumen(); !strings.Contains(got, "Todavía no tengo cargado") {
		t.Errorf("resumen vacío esperaba guía, obtuve: %q", got)
	}
	if got := g.TextoParaIA(); got != "" {
		t.Errorf("perfil vacío no debería generar contexto IA, obtuve: %q", got)
	}
}

func TestGestorEmpresa_ReemplazarYPersiste(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "empresa.json")
	g := NuevoGestorEmpresa(ruta)
	p := PerfilEmpresa{
		Nombre:        "Panadería La Espiga",
		Rubro:         "Alimentos",
		Tamano:        "5 empleados",
		Productos:     []string{"pan", "facturas"},
		Objetivos:     []string{"abrir un segundo local"},
		ContactoDueno: "Juan",
	}
	if err := g.Reemplazar(p); err != nil {
		t.Fatalf("Reemplazar: %v", err)
	}
	g2 := NuevoGestorEmpresa(ruta)
	ob := g2.Obtener()
	if ob.Nombre != "Panadería La Espiga" {
		t.Errorf("nombre no persistió: %q", ob.Nombre)
	}
	if len(ob.Productos) != 2 || ob.Productos[0] != "pan" {
		t.Errorf("productos no persistieron: %v", ob.Productos)
	}
}

func TestGestorEmpresa_SetCampo(t *testing.T) {
	g := NuevoGestorEmpresa(filepath.Join(t.TempDir(), "empresa.json"))
	casos := map[string]string{
		"nombre":     "Mi Empresa",
		"rubro":      "Tecnología",
		"tamano":     "10 empleados",
		"facturacion": "10000 usd",
		"email":      "ventas@mia.com",
		"dueno":      "Ana",
	}
	for clave, valor := range casos {
		if err := g.SetCampo(clave, valor); err != nil {
			t.Fatalf("SetCampo(%q): %v", clave, err)
		}
	}
	p := g.Obtener()
	if p.Nombre != "Mi Empresa" {
		t.Errorf("nombre = %q", p.Nombre)
	}
	if p.Rubro != "Tecnología" {
		t.Errorf("rubro = %q", p.Rubro)
	}
	if p.ContactoDueno != "Ana" {
		t.Errorf("dueño = %q", p.ContactoDueno)
	}
	if p.ContactoMail != "ventas@mia.com" {
		t.Errorf("email = %q", p.ContactoMail)
	}
	if p.Facturacion != "10000 usd" {
		t.Errorf("facturacion = %q", p.Facturacion)
	}
	if err := g.SetCampo("inexistente", "x"); err == nil {
		t.Error("campo inexistente debería dar error")
	}
}

func TestGestorEmpresa_AgregarYBorrarItem(t *testing.T) {
	g := NuevoGestorEmpresa(filepath.Join(t.TempDir(), "empresa.json"))
	_ = g.AgregarItem("productos", "pan")
	_ = g.AgregarItem("productos", "pan") // duplicado
	_ = g.AgregarItem("productos", "facturas")
	p := g.Obtener()
	if len(p.Productos) != 2 {
		t.Fatalf("deberían quedar 2 productos sin duplicados, hay %d: %v", len(p.Productos), p.Productos)
	}
	if err := g.BorrarItem("productos", "pan"); err != nil {
		t.Fatalf("BorrarItem: %v", err)
	}
	if len(g.Obtener().Productos) != 1 || g.Obtener().Productos[0] != "facturas" {
		t.Errorf("tras borrar quedó %v", g.Obtener().Productos)
	}
	if err := g.AgregarItem("lista_inexistente", "x"); err == nil {
		t.Error("lista inexistente debería dar error")
	}
}

func TestGestorEmpresa_TextoParaIA_YResumen(t *testing.T) {
	g := NuevoGestorEmpresa(filepath.Join(t.TempDir(), "empresa.json"))
	g.Reemplazar(PerfilEmpresa{Nombre: "Consultora ABC", Rubro: "Consultoría", Productos: []string{"informes"}, Objetivos: []string{"crecer 10%"}})
	ctx := g.TextoParaIA()
	if !strings.Contains(ctx, "[PERFIL DE LA EMPRESA]") || !strings.Contains(ctx, "Consultora ABC") || !strings.Contains(ctx, "crecer 10%") {
		t.Errorf("contexto IA incompleto: %q", ctx)
	}
	res := g.Resumen()
	if !strings.Contains(res, "Nombre: Consultora ABC") || !strings.Contains(res, "crecer 10%") {
		t.Errorf("resumen incompleto: %q", res)
	}
}

func TestGestorEmpresa_UltimaModificacion(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "empresa.json")
	g := NuevoGestorEmpresa(ruta)
	if _, ok := g.UltimaModificacion(); ok {
		t.Fatal("no debería haber mtime para un archivo inexistente")
	}
	g.Reemplazar(PerfilEmpresa{Nombre: "X"})
	st, ok := g.UltimaModificacion()
	if !ok {
		t.Fatal("debería haber mtime tras guardar")
	}
	if st.IsZero() {
		t.Error("mtime es cero")
	}
	if mod, _ := os.Stat(ruta); !mod.ModTime().After(time.Time{}) {
		t.Error("el archivo debería existir")
	}
}