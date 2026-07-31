package core

import (
	"strings"
	"testing"
)

func TestParseSkill(t *testing.T) {
	contenido := `---
nombre: prueba
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
	deseadas := map[string]bool{"seguridad": false, "estilo-go": false, "rest-api": false, "frontend-panel": false, "proyecto-nuevo": false}
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
