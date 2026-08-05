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
	if len(etiquetas) != 4 {
		t.Fatalf("se esperaban 4 roles embebidos, obtuve %d: %v", len(etiquetas), etiquetas)
	}
	esperados := []string{
		"CEO empresarial",
		"Licenciado en marketing",
		"Modo humano",
		"Asistente corporativo",
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
		{"marketing", "licenciado_marketing"},
		{"humano", "humano"},
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
		{"la pc anda muy lenta", ""},
		{"escribime un post para instagram", "licenciado_marketing"},
		{"cómo hago crecer mi negocio", "ceo_empresarial"},
		{"quiero conversar un rato", "humano"},
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

func TestCRUDRol(t *testing.T) {
	dir := t.TempDir()
	m := nuevoRolesManagerConDir(dir)
	antes := len(m.Listar())

	nuevo := Rol{
		Nombre:      "analista_datos",
		Etiqueta:    "Analista de datos",
		Descripcion: "Analiza datos y saca conclusiones.",
		Activar:     []string{"analista", "datos", "informe de datos"},
		Prompt:      "Sos un analista de datos senior.",
	}
	if err := m.CrearOActualizar(nuevo); err != nil {
		t.Fatalf("CrearOActualizar: %v", err)
	}
	if len(m.Listar()) != antes+1 {
		t.Errorf("debería haber un rol más, obtuve %d (antes %d)", len(m.Listar()), antes)
	}
	m.Recargar()
	obtenido, ok := m.Obtener("analista_datos")
	if !ok {
		t.Fatal("no se encontró el rol creado tras recargar")
	}
	if obtenido.Prompt != "Sos un analista de datos senior." || len(obtenido.Activar) != 3 {
		t.Errorf("round-trip incorrecto: %+v", obtenido)
	}
	// El rol creado debe poder activarse por su etiqueta.
	if !m.Activar("analista de datos") {
		t.Error("el rol nuevo no se activa por etiqueta")
	}
	m.Desactivar()
	// Actualizar y eliminar.
	nuevo.Prompt = "Prompt nuevo."
	if err := m.CrearOActualizar(nuevo); err != nil {
		t.Fatalf("actualizar: %v", err)
	}
	obtenido, _ = m.Obtener("analista_datos")
	if obtenido.Prompt != "Prompt nuevo." {
		t.Errorf("prompt no actualizado: %q", obtenido.Prompt)
	}
	if err := m.Eliminar("analista_datos"); err != nil {
		t.Fatalf("Eliminar: %v", err)
	}
	if _, ok := m.Obtener("analista_datos"); ok {
		t.Error("el rol eliminado sigue apareciendo")
	}
}

func TestCRUDRolValidaciones(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	casos := []struct {
		nombre string
		r      Rol
	}{
		{"sin nombre", Rol{Etiqueta: "e", Prompt: "p"}},
		{"sin etiqueta", Rol{Nombre: "n", Prompt: "p"}},
		{"sin prompt", Rol{Nombre: "n", Etiqueta: "e"}},
		{"nombre inválido", Rol{Nombre: "a/b", Etiqueta: "e", Prompt: "p"}},
	}
	for _, c := range casos {
		if err := m.CrearOActualizar(c.r); err == nil {
			t.Errorf("%s: debería devolver error", c.nombre)
		}
	}
}

func TestSerializarRolParseable(t *testing.T) {
	original := Rol{Nombre: "r", Etiqueta: "Rol", Descripcion: "D", Contexto: "ctx.md", Activar: []string{"x", "y"}, Prompt: "Cuerpo"}
	parseada, ok := parseRol(serializarRol(original))
	if !ok {
		t.Fatal("serializarRol no produce un front-matter parseable")
	}
	if parseada.Nombre != "r" || parseada.Etiqueta != "Rol" || parseada.Contexto != "ctx.md" || len(parseada.Activar) != 2 {
		t.Errorf("round-trip de serialización incorrecto: %+v", parseada)
	}
}

func TestAsistenteCorporativoSeActivaPorFrase(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	casos := []string{
		"un cliente me pidió una reunión",
		"coordiná el seguimiento del contrato",
		"el proveedor entregó el pedido",
		"gestioná la agenda del día",
	}
	for _, c := range casos {
		if !m.Activar(c) {
			t.Errorf("el rol corporativo debería activarse con %q", c)
		}
		m.Desactivar()
	}
	if r := m.BuscarRol("asistente corporativo"); r == nil || r.Nombre != "asistente_corporativo" {
		t.Errorf("BuscarRol('asistente corporativo') no devolvió el rol")
	}
}

func TestAsistenteCorporativoPromptDeElite(t *testing.T) {
	m := nuevoRolesManagerConDir(t.TempDir())
	if !m.Activar("asistente corporativo") {
		t.Fatal("no se pudo activar el rol")
	}
	texto := m.TextoParaIA("un cliente quiere coordinar")
	if !strings.Contains(texto, "Asistente corporativo") {
		t.Errorf("falta la etiqueta del rol: %q", texto)
	}
	if !strings.Contains(texto, "estructura ejecutiva") {
		t.Errorf("el prompt debería exigir estructura ejecutiva: %q", texto)
	}
}
