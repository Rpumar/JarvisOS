package core

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

// Skill es un bloque de instrucciones reutilizable que Jarvis inyecta en la
// IA cuando la petición coincide con sus palabras de activación. Es el
// equivalente a las skills de Claude Code / CLAUDE.md, pero por voz.
type Skill struct {
	Nombre        string
	Activar       []string
	Instrucciones string
}

// SkillsManager carga skills embebidas (defecto) y de la carpeta del usuario
// (USERPROFILE\JarvisOS-datos\skills\*.md) y las activa por coincidencia.
type SkillsManager struct {
	skills []Skill
}

//go:embed skills
var skillsFS embed.FS

func NuevoSkillsManager() *SkillsManager {
	m := &SkillsManager{}
	m.cargarDefaults()
	m.cargarUsuario()
	return m
}

func (m *SkillsManager) cargarDefaults() {
	entradas, err := skillsFS.ReadDir("skills")
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		datos, err := skillsFS.ReadFile("skills/" + e.Name())
		if err != nil {
			continue
		}
		if s, ok := parseSkill(string(datos)); ok {
			m.skills = append(m.skills, s)
		}
	}
}

func (m *SkillsManager) cargarUsuario() {
	dir := filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "skills")
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		datos, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		s, ok := parseSkill(string(datos))
		if !ok {
			continue
		}
		reemplazado := false
		for i := range m.skills {
			if m.skills[i].Nombre == s.Nombre {
				m.skills[i] = s
				reemplazado = true
				break
			}
		}
		if !reemplazado {
			m.skills = append(m.skills, s)
		}
	}
}

// parseSkill lee el front-matter: --- nombre: ... activar: [...] --- + cuerpo.
func parseSkill(contenido string) (Skill, bool) {
	lineas := strings.Split(contenido, "\n")
	if len(lineas) < 3 || strings.TrimSpace(lineas[0]) != "---" {
		return Skill{}, false
	}
	fin := -1
	for i := 1; i < len(lineas); i++ {
		if strings.TrimSpace(lineas[i]) == "---" {
			fin = i
			break
		}
	}
	if fin == -1 {
		return Skill{}, false
	}
	var s Skill
	for _, l := range lineas[1:fin] {
		trim := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(trim, "nombre:"):
			s.Nombre = strings.TrimSpace(strings.TrimPrefix(trim, "nombre:"))
		case strings.HasPrefix(trim, "activar:"):
			lista := strings.TrimSpace(strings.TrimPrefix(trim, "activar:"))
			lista = strings.Trim(lista, "[]")
			for _, k := range strings.Split(lista, ",") {
				k = strings.Trim(strings.TrimSpace(k), `"'`)
				if k != "" {
					s.Activar = append(s.Activar, strings.ToLower(k))
				}
			}
		}
	}
	s.Instrucciones = strings.TrimSpace(strings.Join(lineas[fin+1:], "\n"))
	if s.Nombre == "" || len(s.Activar) == 0 || s.Instrucciones == "" {
		return Skill{}, false
	}
	return s, true
}

// Buscar devuelve las skills cuyas palabras de activación aparecen en la
// petición. Normaliza la entrada (quita tildes y puntuación) para que el
// voseo y los acentos no rompan la activación.
func (m *SkillsManager) Buscar(entrada string) []Skill {
	entrada = strings.ToLower(simplificar(entrada))
	var res []Skill
	for _, s := range m.skills {
		for _, k := range s.Activar {
			if strings.Contains(entrada, k) {
				res = append(res, s)
				break
			}
		}
	}
	return res
}

// TextoParaIA arma el bloque de instrucciones para inyectar en el prompt de la
// IA. Vacío si ninguna skill aplica.
func (m *SkillsManager) TextoParaIA(entrada string) string {
	skills := m.Buscar(entrada)
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[INSTRUCCIONES DE SKILL]\n")
	for _, s := range skills {
		b.WriteString("## " + s.Nombre + "\n")
		b.WriteString(s.Instrucciones + "\n")
	}
	b.WriteString("[/INSTRUCCIONES DE SKILL]")
	return b.String()
}

func (m *SkillsManager) Listar() []string {
	nombres := make([]string, 0, len(m.skills))
	for _, s := range m.skills {
		nombres = append(nombres, s.Nombre)
	}
	return nombres
}
