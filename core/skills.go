package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Skill es un bloque de instrucciones reutilizable que Jarvis inyecta en la
// IA cuando la petición coincide con sus palabras de activación. Es el
// equivalente a las skills de Claude Code / CLAUDE.md, pero por voz.
type Skill struct {
	Nombre        string
	Activar       []string
	Instrucciones string
	Prioridad     int
	Descripcion   string
}

// SkillsManager carga skills embebidas (defecto) y de la carpeta del usuario
// (USERPROFILE\JarvisOS-datos\skills\*.md) y las activa por coincidencia.
// Es seguro para uso concurrente: los handlers de la WebUI y el brain
// comparten la misma instancia.
type SkillsManager struct {
	mu       sync.RWMutex
	skills   []Skill
	datosDir string
}

//go:embed skills
var skillsFS embed.FS

func NuevoSkillsManager() *SkillsManager {
	return nuevoSkillsManagerConDir(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos"))
}

func nuevoSkillsManagerConDir(dir string) *SkillsManager {
	m := &SkillsManager{datosDir: dir}
	m.cargarDefaults()
	m.cargarUsuario()
	return m
}

// Recargar vuelve a leer las skills del usuario desde disco, sin reiniciar.
// Se usa tras crear/editar/borrar desde la WebUI.
func (m *SkillsManager) Recargar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skills = m.skills[:0]
	m.cargarDefaults()
	m.cargarUsuario()
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
	dir := filepath.Join(m.datosDir, "skills")
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
		case strings.HasPrefix(trim, "descripcion:"):
			s.Descripcion = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trim, "descripcion:")), `"'`)
		case strings.HasPrefix(trim, "activar:"):
			lista := strings.TrimSpace(strings.TrimPrefix(trim, "activar:"))
			lista = strings.Trim(lista, "[]")
			for _, k := range strings.Split(lista, ",") {
				k = strings.Trim(strings.TrimSpace(k), `"'`)
				if k != "" {
					s.Activar = append(s.Activar, strings.ToLower(k))
				}
			}
		case strings.HasPrefix(trim, "prioridad:"):
			v := strings.TrimSpace(strings.TrimPrefix(trim, "prioridad:"))
			if n, err := strconv.Atoi(v); err == nil {
				s.Prioridad = n
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
// voseo y los acentos no rompan la activación. Ordena el resultado por
// puntaje (cantidad de palabras de activación presentes, lo que reduce falsos
// positivos) y luego por prioridad (mayor gana).
func (m *SkillsManager) Buscar(entrada string) []Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entrada = strings.ToLower(simplificar(entrada))
	type candidata struct {
		skill Skill
		punt  int
	}
	var res []candidata
	for _, s := range m.skills {
		punt := 0
		for _, k := range s.Activar {
			if strings.Contains(entrada, k) {
				punt++
			}
		}
		if punt > 0 {
			res = append(res, candidata{skill: s, punt: punt})
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].punt != res[j].punt {
			return res[i].punt > res[j].punt
		}
		return res[i].skill.Prioridad > res[j].skill.Prioridad
	})
	salida := make([]Skill, 0, len(res))
	for _, c := range res {
		salida = append(salida, c.skill)
	}
	return salida
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	nombres := make([]string, 0, len(m.skills))
	for _, s := range m.skills {
		nombres = append(nombres, s.Nombre)
	}
	return nombres
}

// SkillInfo es la vista pública de una skill para el listado del panel.
type SkillInfo struct {
	Nombre      string   `json:"nombre"`
	Descripcion string   `json:"descripcion,omitempty"`
	Activar     []string `json:"activar,omitempty"`
	Prioridad   int      `json:"prioridad"`
}

// ListarDetallado devuelve la información de todas las skills, ordenadas por
// prioridad descendente (mayor primero).
func (m *SkillsManager) ListarDetallado() []SkillInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	skills := make([]Skill, len(m.skills))
	copy(skills, m.skills)
	sort.SliceStable(skills, func(i, j int) bool {
		return skills[i].Prioridad > skills[j].Prioridad
	})
	res := make([]SkillInfo, 0, len(skills))
	for _, s := range skills {
		res = append(res, SkillInfo{
			Nombre:      s.Nombre,
			Descripcion: s.Descripcion,
			Activar:     s.Activar,
			Prioridad:   s.Prioridad,
		})
	}
	return res
}

// Obtener devuelve la skill completa (incluida las instrucciones) por nombre,
// o false si no existe. Se usa para editar desde la WebUI.
func (m *SkillsManager) Obtener(nombre string) (Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.skills {
		if s.Nombre == nombre {
			return s, true
		}
	}
	return Skill{}, false
}

// RutaUsuario devuelve la ruta del archivo .md del usuario para una skill, o
// vacío si el nombre no es válido.
func (m *SkillsManager) RutaUsuario(nombre string) string {
	if nombre == "" || strings.ContainsAny(nombre, `/\:*?"<>|`) {
		return ""
	}
	return filepath.Join(m.datosDir, "skills", nombre+".md")
}

// CrearOActualizar guarda (o sobrescribe) una skill del usuario y recarga la
// lista en caliente. Devuelve un error si los campos obligatorios están vacíos.
func (m *SkillsManager) CrearOActualizar(s Skill) error {
	s.Nombre = strings.TrimSpace(s.Nombre)
	s.Descripcion = strings.TrimSpace(s.Descripcion)
	s.Instrucciones = strings.TrimSpace(s.Instrucciones)
	if s.Nombre == "" {
		return fmt.Errorf("el nombre de la skill es obligatorio")
	}
	if len(s.Activar) == 0 {
		return fmt.Errorf("la skill debe tener al menos una palabra de activación")
	}
	if s.Instrucciones == "" {
		return fmt.Errorf("la skill debe tener instrucciones")
	}
	ruta := m.RutaUsuario(s.Nombre)
	if ruta == "" {
		return fmt.Errorf("nombre de skill inválido: %q", s.Nombre)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(ruta, []byte(serializarSkill(s)), 0o600); err != nil {
		return err
	}
	m.Recargar()
	return nil
}

// Eliminar borra una skill del usuario. Si la skill es embebida (no existe
// archivo de usuario), no hace nada y devuelve nil.
func (m *SkillsManager) Eliminar(nombre string) error {
	ruta := m.RutaUsuario(nombre)
	if ruta == "" {
		return fmt.Errorf("nombre de skill inválido: %q", nombre)
	}
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		return nil
	}
	if err := os.Remove(ruta); err != nil {
		return err
	}
	m.Recargar()
	return nil
}

// serializarSkill arma el archivo .md con front-matter de una skill.
func serializarSkill(s Skill) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("nombre: " + s.Nombre + "\n")
	if s.Descripcion != "" {
		b.WriteString("descripcion: \"" + s.Descripcion + "\"\n")
	}
	b.WriteString("prioridad: " + strconv.Itoa(s.Prioridad) + "\n")
	claves := make([]string, 0, len(s.Activar))
	for _, k := range s.Activar {
		claves = append(claves, strings.TrimSpace(k))
	}
	b.WriteString("activar: [" + strings.Join(claves, ", ") + "]\n")
	b.WriteString("---\n")
	b.WriteString(s.Instrucciones + "\n")
	return b.String()
}
