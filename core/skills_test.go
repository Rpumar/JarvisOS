package core

import (
	"strings"
	"testing"
)

func TestParseSkill(t *testing.T) {
	contenido := `---
nombre: prueba
prioridad: 7
descripcion: Una skill de prueba.
activar: [boton, api, endpoin]
---
Instrucciones de prueba.
Segunda línea.`
	s, ok := parseSkill(contenido)
	if !ok {
		t.Fatal("parseSkill falló con skill válida")
	}
	if s.Nombre != "prueba" {
		t.Errorf("nombre = %q", s.Nombre)
	}
	if s.Prioridad != 7 {
		t.Errorf("prioridad = %d, esperaba 7", s.Prioridad)
	}
	if s.Descripcion != "Una skill de prueba." {
		t.Errorf("descripcion = %q", s.Descripcion)
	}
	if len(s.Activar) != 3 || s.Activar[1] != "api" {
		t.Errorf("activar = %v", s.Activar)
	}
	if !strings.Contains(s.Instrucciones, "Segunda línea") {
		t.Errorf("instrucciones = %q", s.Instrucciones)
	}

	if _, ok := parseSkill("sin front matter"); ok {
		t.Error("debería rechazar texto sin front-matter")
	}
	if _, ok := parseSkill("---\nactivar: [x]\n---\n"); ok {
		t.Error("debería rechazar skill sin nombre")
	}
	if _, ok := parseSkill("---\nnombre: x\n---\n"); ok {
		t.Error("debería rechazar skill sin activar")
	}
}

func TestBuscarSkills(t *testing.T) {
	m := NuevoSkillsManager()
	if len(m.Listar()) == 0 {
		t.Fatal("no se cargaron skills (ni embebidas ni del usuario)")
	}
	// La skill de seguridad debe activarse con una petición peligrosa.
	encontrada := false
	for _, s := range m.Buscar("borrá el archivo que no compila") {
		if s.Nombre == "seguridad" {
			encontrada = true
		}
	}
	if !encontrada {
		t.Error("la skill 'seguridad' no se activó con 'borrá'")
	}

	if len(m.Buscar("decime la hora")) != 0 {
		t.Error("no debería activar ninguna skill con 'decime la hora'")
	}
}

func TestBuscarPuntajeYPrioridad(t *testing.T) {
	m := NuevoSkillsManager()
	// Una petición que toque varias palabras de la misma skill suma puntaje.
	skills := m.Buscar("borrá el firewall y reiniciá, es peligroso")
	if len(skills) == 0 {
		t.Fatal("debería activar al menos la skill de seguridad")
	}
	if skills[0].Nombre != "seguridad" {
		t.Errorf("con varias coincidencias la más puntuada debería ir primero, obtuve %v", skills[0].Nombre)
	}
	// Orden por prioridad cuando el puntaje empata: seguridad (100) > rest-api (50).
	porPalabras := m.Buscar("borrá el registro")
	if len(porPalabras) == 0 || porPalabras[0].Nombre != "seguridad" {
		t.Errorf("seguridad debería ganar por prioridad, obtuve %v", porPalabras)
	}
}

func TestTextoParaIA(t *testing.T) {
	m := NuevoSkillsManager()
	texto := m.TextoParaIA("agregá un endpoint nuevo a la api")
	if !strings.Contains(texto, "[INSTRUCCIONES DE SKILL]") {
		t.Error("debería envolver las instrucciones en el bloque esperado")
	}
	if !strings.Contains(texto, "## rest-api") {
		t.Error("debería incluir la skill rest-api")
	}
	if m.TextoParaIA("hola señor") != "" {
		t.Error("debería devolver vacío cuando ninguna skill aplica")
	}
}

func TestNuevoSkillsManagerDefaults(t *testing.T) {
	m := NuevoSkillsManager()
	names := m.Listar()
	deseadas := map[string]bool{"seguridad": false, "estilo-go": false, "rest-api": false, "frontend-panel": false, "proyecto-nuevo": false, "asistente-corporativo": false}
	for _, n := range names {
		if _, ok := deseadas[n]; ok {
			deseadas[n] = true
		}
	}
	for nombre, encontrada := range deseadas {
		if !encontrada {
			t.Errorf("falta la skill embebida %q (cargadas: %v)", nombre, names)
		}
	}
}

func TestListarDetalladoOrdenado(t *testing.T) {
	m := NuevoSkillsManager()
	detalles := m.ListarDetallado()
	if len(detalles) != 6 {
		t.Fatalf("se esperaban 6 skills detalladas, obtuve %d", len(detalles))
	}
	// Ordenada por prioridad descendente: seguridad primero.
	if detalles[0].Nombre != "seguridad" {
		t.Errorf("la primera debería ser seguridad (prioridad 100), obtuve %v", detalles[0].Nombre)
	}
	// La skill de seguridad debe traer descripción y palabras clave.
	if detalles[0].Descripcion == "" {
		t.Error("la skill de seguridad debería tener descripción")
	}
	if len(detalles[0].Activar) == 0 {
		t.Error("la skill de seguridad debería listar sus palabras de activación")
	}
}

func TestCRUDSkill(t *testing.T) {
	dir := t.TempDir()
	m := nuevoSkillsManagerConDir(dir)
	antes := len(m.Listar())

	nueva := Skill{Nombre: "mi-skill", Descripcion: "Desc", Activar: []string{"mi", "skill"}, Instrucciones: "Instrucciones.", Prioridad: 3}
	if err := m.CrearOActualizar(nueva); err != nil {
		t.Fatalf("CrearOActualizar: %v", err)
	}
	if len(m.Listar()) != antes+1 {
		t.Errorf("debería haber una skill más, obtuve %d (antes %d)", len(m.Listar()), antes)
	}
	// Round-trip: al recargar sigue existiendo y con instrucciones.
	m.Recargar()
	obtenida, ok := m.Obtener("mi-skill")
	if !ok {
		t.Fatal("no se encontró la skill creada tras recargar")
	}
	if obtenida.Instrucciones != "Instrucciones." || obtenida.Prioridad != 3 || len(obtenida.Activar) != 2 {
		t.Errorf("round-trip incorrecto: %+v", obtenida)
	}
	// Actualizar (sobrescribe) y verificar.
	nueva.Prioridad = 9
	if err := m.CrearOActualizar(nueva); err != nil {
		t.Fatalf("actualizar: %v", err)
	}
	obtenida, _ = m.Obtener("mi-skill")
	if obtenida.Prioridad != 9 {
		t.Errorf("prioridad no actualizada: %d", obtenida.Prioridad)
	}
	// Eliminar.
	if err := m.Eliminar("mi-skill"); err != nil {
		t.Fatalf("Eliminar: %v", err)
	}
	if _, ok := m.Obtener("mi-skill"); ok {
		t.Error("la skill eliminada sigue apareciendo")
	}
	if len(m.Listar()) != antes {
		t.Errorf("tras eliminar debería volver a %d, obtuve %d", antes, len(m.Listar()))
	}
}

func TestCRUDSkillValidaciones(t *testing.T) {
	dir := t.TempDir()
	m := nuevoSkillsManagerConDir(dir)
	casos := []struct {
		nombre string
		s      Skill
	}{
		{"sin nombre", Skill{Activar: []string{"x"}, Instrucciones: "y"}},
		{"sin activar", Skill{Nombre: "a", Instrucciones: "y"}},
		{"sin instrucciones", Skill{Nombre: "a", Activar: []string{"x"}}},
		{"nombre inválido", Skill{Nombre: "a/b", Activar: []string{"x"}, Instrucciones: "y"}},
	}
	for _, c := range casos {
		if err := m.CrearOActualizar(c.s); err == nil {
			t.Errorf("%s: debería devolver error", c.nombre)
		}
	}
}

func TestSerializarSkillParseable(t *testing.T) {
	original := Skill{Nombre: "s", Descripcion: "Desc con \"comillas\"", Activar: []string{"a", "b"}, Instrucciones: "Cuerpo", Prioridad: 4}
	parseada, ok := parseSkill(serializarSkill(original))
	if !ok {
		t.Fatal("serializarSkill no produce un front-matter parseable")
	}
	if parseada.Nombre != "s" || parseada.Prioridad != 4 || len(parseada.Activar) != 2 {
		t.Errorf("round-trip de serialización incorrecto: %+v", parseada)
	}
}

func TestAsistenteCorporativoSeActivaConPedidoVago(t *testing.T) {
	m := NuevoSkillsManager()
	skills := m.Buscar("un cliente pidió coordinar una reunión para el martes")
	encontrada := false
	for _, s := range skills {
		if s.Nombre == "asistente-corporativo" {
			encontrada = true
		}
	}
	if !encontrada {
		t.Errorf("la skill asistente-corporativo no se activó con un pedido de cliente: %v", skills)
	}
	// Un pedido simple no debería activarla.
	if len(m.Buscar("decime la hora")) != 0 {
		t.Error("un pedido simple no debería activar la skill corporativa")
	}
}
