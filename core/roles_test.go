package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRol(t *testing.T) {
	contenido := `---
nombre: ceo_empresarial
etiqueta: CEO empresarial
descripcion: Asesor ejecutivo
activar: [ceo, empresa, estrategia]
contexto: empresa.md
---
Sos un CEO experto en negocios.
`
	r, ok := parseRol(contenido)
	if !ok {
		t.Fatal("se esperaba parsear el rol")
	}
	if r.Nombre != "ceo_empresarial" {
		t.Errorf("nombre = %q, esperaba ceo_empresarial", r.Nombre)
	}
	if r.Etiqueta != "CEO empresarial" {
		t.Errorf("etiqueta = %q, esperaba CEO empresarial", r.Etiqueta)
	}
	if r.Descripcion != "Asesor ejecutivo" {
		t.Errorf("descripcion = %q, esperaba Asesor ejecutivo", r.Descripcion)
	}
	if len(r.Activar) != 3 || r.Activar[0] != "ceo" {
		t.Errorf("activar = %v, esperaba [ceo empresa estrategia]", r.Activar)
	}
	if r.Contexto != "empresa.md" {
		t.Errorf("contexto = %q, esperaba empresa.md", r.Contexto)
	}
	if !strings.Contains(r.Prompt, "CEO experto") {
		t.Errorf("prompt = %q, esperaba que contenga el cuerpo", r.Prompt)
	}
}

func TestNuevoRolesManagerDefaults(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	etiquetas := m.Listar()
	if len(etiquetas) != 5 {
		t.Fatalf("se esperaban 5 roles embebidos, obtuve %d: %v", len(etiquetas), etiquetas)
	}
	esperados := []string{
		"Ingeniero en sistemas",
		"Desarrollador fullstack senior",
		"CEO empresarial",
		"Licenciado en marketing",
		"Modo humano",
	}
	lista := strings.Join(etiquetas, ",")
	lista = strings.ToLower(lista)
	for _, e := range esperados {
		if !strings.Contains(lista, strings.ToLower(e)) {
			t.Errorf("faltó el rol %q en: %v", e, etiquetas)
		}
	}
}

func TestBuscarRolPorModo(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"ceo", "ceo_empresarial"},
		{"CEO", "ceo_empresarial"},
		{"modo ceo", "ceo_empresarial"},
		{"modo ceo empresarial", "ceo_empresarial"},
		{"ingeniero", "ingeniero_sistemas"},
		{"marketing", "licenciado_marketing"},
		{"humano", "humano"},
		{"desarrollador", "desarrollador_fullstack"},
		{"algo raro", ""},
	}
	for _, c := range casos {
		r := m.BuscarRol(c.entrada)
		if c.esperado == "" {
			if r != nil {
				t.Errorf("BuscarRol(%q) = %v, esperaba nil", c.entrada, r)
			}
			continue
		}
		if r == nil || r.Nombre != c.esperado {
			t.Errorf("BuscarRol(%q) = %v, esperaba %s", c.entrada, r, c.esperado)
		}
	}
}

func TestSugerirRolesPorPalabras(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"la pc anda muy lenta", "ingeniero_sistemas"},
		{"escribime un post para instagram", "licenciado_marketing"},
		{"cómo hago crecer mi negocio", "ceo_empresarial"},
		{"quiero conversar un rato", "humano"},
		{"desarrollame una función en go", "desarrollador_fullstack"},
		{"qué hora es", ""},
	}
	for _, c := range casos {
		sugeridos := m.Sugerir(c.entrada)
		encontrado := c.esperado == ""
		for _, s := range sugeridos {
			if s.Nombre == c.esperado {
				encontrado = true
			}
		}
		if !encontrado {
			t.Errorf("Sugerir(%q) no incluyó %s: %+v", c.entrada, c.esperado, sugeridos)
		}
	}
}

func TestActivarDesactivarRol(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	if !m.Activar("ceo") {
		t.Fatal("debería activar el rol ceo")
	}
	if m.RolActivo() == nil || m.RolActivo().Nombre != "ceo_empresarial" {
		t.Fatalf("rol activo = %+v, esperaba ceo_empresarial", m.RolActivo())
	}
	if et := m.Desactivar(); et != "CEO empresarial" {
		t.Errorf("Desactivar() = %q, esperaba CEO empresarial", et)
	}
	if m.RolActivo() != nil {
		t.Error("no debería quedar rol activo tras desactivar")
	}
	if m.Activar("no existe") {
		t.Error("no debería activar un rol inexistente")
	}
}

func TestTextoParaIA_RolActivoConContexto(t *testing.T) {
	dir := t.TempDir()
	m := nuevoRolesManagerConDir(dir)
	if !m.Activar("ceo") {
		t.Fatal("debería activar el rol ceo")
	}
	texto := m.TextoParaIA("qué hago con la empresa")
	if !strings.Contains(texto, "[INSTRUCCIONES DE ROL]") {
		t.Errorf("falta el bloque de rol: %q", texto)
	}
	if !strings.Contains(texto, "CEO empresarial") {
		t.Errorf("falta la etiqueta del rol: %q", texto)
	}
	if !strings.Contains(texto, "(modo activo)") {
		t.Errorf("debería marcar el modo activo: %q", texto)
	}
	if _, err := os.Stat(filepath.Join(dir, "empresa.md")); err != nil {
		t.Errorf("se esperaba la plantilla de empresa.md creada: %v", err)
	}
}

func TestTextoParaIA_RolSugeridoUnTurno(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	texto := m.TextoParaIA("escribime un post de instagram")
	if !strings.Contains(texto, "Licenciado en marketing") {
		t.Errorf("se esperaba el rol de marketing, obtuve: %q", texto)
	}
	if strings.Contains(texto, "(modo activo)") {
		t.Error("no debería marcar modo activo en un uso de un turno")
	}
}
