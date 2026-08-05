package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Rol es una "mente" especializada que Jarvis puede activar: un perfil
// profesional con su propio prompt de sistema, palabras de activación y un
// contexto opcional (ej: el perfil de la empresa para el CEO).
type Rol struct {
	Nombre      string
	Etiqueta    string
	Descripcion string
	Activar     []string
	Contexto    string
	Prompt      string
}

// RolesManager carga los roles embebidos (defecto) y los de la carpeta del
// usuario (USERPROFILE\JarvisOS-datos\roles\*.md, que reemplazan por nombre).
// Un rol puede estar "activo" (modo persistente: "modo ceo") o activarse solo
// para un turno cuando la petición coincide con sus palabras clave.
type RolesManager struct {
	roles    []Rol
	activo   *Rol
	datosDir string
	mu       sync.RWMutex
}

//go:embed roles
var rolesFS embed.FS

func NuevoRolesManager() *RolesManager {
	return nuevoRolesManagerConDir(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos"))
}

func nuevoRolesManagerConDir(dir string) *RolesManager {
	m := &RolesManager{datosDir: dir}
	m.cargarDefaults()
	m.cargarUsuario()
	return m
}

func (m *RolesManager) cargarDefaults() {
	entradas, err := rolesFS.ReadDir("roles")
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		datos, err := rolesFS.ReadFile("roles/" + e.Name())
		if err != nil {
			continue
		}
		if r, ok := parseRol(string(datos)); ok {
			m.roles = append(m.roles, r)
		}
	}
}

func (m *RolesManager) cargarUsuario() {
	dir := filepath.Join(m.datosDir, "roles")
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
		r, ok := parseRol(string(datos))
		if !ok {
			continue
		}
		reemplazado := false
		for i := range m.roles {
			if m.roles[i].Nombre == r.Nombre {
				m.roles[i] = r
				reemplazado = true
				break
			}
		}
		if !reemplazado {
			m.roles = append(m.roles, r)
		}
	}
}

// parseRol lee el front-matter: --- nombre / etiqueta / descripcion / activar
// / contexto + el cuerpo del prompt.
func parseRol(contenido string) (Rol, bool) {
	lineas := strings.Split(contenido, "\n")
	if len(lineas) < 3 || strings.TrimSpace(lineas[0]) != "---" {
		return Rol{}, false
	}
	fin := -1
	for i := 1; i < len(lineas); i++ {
		if strings.TrimSpace(lineas[i]) == "---" {
			fin = i
			break
		}
	}
	if fin == -1 {
		return Rol{}, false
	}
	var r Rol
	for _, l := range lineas[1:fin] {
		trim := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(trim, "nombre:"):
			r.Nombre = strings.TrimSpace(strings.TrimPrefix(trim, "nombre:"))
		case strings.HasPrefix(trim, "etiqueta:"):
			r.Etiqueta = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trim, "etiqueta:")), `"'`)
		case strings.HasPrefix(trim, "descripcion:"):
			r.Descripcion = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trim, "descripcion:")), `"'`)
		case strings.HasPrefix(trim, "activar:"):
			lista := strings.TrimSpace(strings.TrimPrefix(trim, "activar:"))
			lista = strings.Trim(lista, "[]")
			for _, k := range strings.Split(lista, ",") {
				k = strings.Trim(strings.TrimSpace(k), `"'`)
				if k != "" {
					r.Activar = append(r.Activar, strings.ToLower(k))
				}
			}
		case strings.HasPrefix(trim, "contexto:"):
			r.Contexto = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trim, "contexto:")), `"'`)
		}
	}
	r.Prompt = strings.TrimSpace(strings.Join(lineas[fin+1:], "\n"))
	if r.Nombre == "" || r.Etiqueta == "" || r.Prompt == "" {
		return Rol{}, false
	}
	return r, true
}

// Activar activa el modo persistente de un rol. Acepta el nombre interno, la
// etiqueta o una palabra de activación ("ceo", "marketing"...).
func (m *RolesManager) Activar(texto string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.buscarRol(texto)
	if r == nil {
		return false
	}
	m.activo = r
	return true
}

// Desactivar apaga el modo persistente. Devuelve la etiqueta del rol que
// estaba activo, o vacío si no había ninguno.
func (m *RolesManager) Desactivar() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activo == nil {
		return ""
	}
	etiqueta := m.activo.Etiqueta
	m.activo = nil
	return etiqueta
}

func (m *RolesManager) RolActivo() *Rol {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activo
}

// BuscarRol encuentra un rol por nombre, etiqueta o palabra de activación.
func (m *RolesManager) BuscarRol(texto string) *Rol {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buscarRol(texto)
}

func (m *RolesManager) buscarRol(texto string) *Rol {
	texto = simplificar(strings.ToLower(texto))
	for i := range m.roles {
		r := &m.roles[i]
		ids := append([]string{r.Nombre, r.Etiqueta}, r.Activar...)
		for _, k := range ids {
			k = simplificar(strings.ToLower(k))
			if k != "" && (strings.Contains(texto, k) || strings.Contains(k, texto)) {
				return r
			}
		}
	}
	return nil
}

// Sugerir devuelve los roles cuyas palabras de activación aparecen en la
// petición (normalizada, sin tildes), para un uso de un solo turno.
func (m *RolesManager) Sugerir(entrada string) []Rol {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sugerir(entrada)
}

func (m *RolesManager) sugerir(entrada string) []Rol {
	entrada = simplificar(strings.ToLower(entrada))
	var res []Rol
	for _, r := range m.roles {
		for _, k := range r.Activar {
			if strings.Contains(entrada, simplificar(strings.ToLower(k))) {
				res = append(res, r)
				break
			}
		}
	}
	return res
}

// TextoParaIA arma el bloque de instrucciones para inyectar en el prompt de la
// IA: el rol activo (modo persistente) o los roles sugeridos por el turno. Si
// el rol tiene contexto, se carga del archivo correspondiente (y se crea una
// plantilla si no existe). Vacío si ningún rol aplica.
func (m *RolesManager) TextoParaIA(entrada string) string {
	m.mu.RLock()
	var roles []*Rol
	if m.activo != nil {
		roles = append(roles, m.activo)
	} else {
		for _, s := range m.sugerir(entrada) {
			for i := range m.roles {
				if m.roles[i].Nombre == s.Nombre {
					roles = append(roles, &m.roles[i])
				}
			}
		}
	}
	m.mu.RUnlock()

	if len(roles) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[INSTRUCCIONES DE ROL]\n")
	for _, r := range roles {
		modo := ""
		if m.activo != nil && m.activo.Nombre == r.Nombre {
			modo = " (modo activo)"
		}
		b.WriteString("## " + r.Etiqueta + modo + "\n")
		b.WriteString(r.Prompt + "\n")
		if ctx := m.leerContexto(r); ctx != "" {
			b.WriteString("CONTEXTO:\n" + ctx + "\n")
		}
	}
	b.WriteString("[/INSTRUCCIONES DE ROL]")
	return b.String()
}

func (m *RolesManager) leerContexto(r *Rol) string {
	if r.Contexto == "" {
		return ""
	}
	ruta := filepath.Join(m.datosDir, r.Contexto)
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		m.crearPlantillaContexto(r)
		return ""
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(datos))
}

func (m *RolesManager) crearPlantillaContexto(r *Rol) {
	var plantilla string
	if r.Contexto == "empresa.md" {
		plantilla = plantillaEmpresa
	} else {
		plantilla = fmt.Sprintf("# Contexto de %s\nEscribí acá la información que %s debe conocer.\n", r.Etiqueta, r.Etiqueta)
	}
	_ = os.MkdirAll(m.datosDir, 0o700)
	_ = os.WriteFile(filepath.Join(m.datosDir, r.Contexto), []byte(plantilla), 0o600)
}

const plantillaEmpresa = `# Perfil de la empresa
# Completá estos datos para que Jarvis te asesore como CEO y como especialista en marketing.
- Nombre de la empresa:
- Rubro / sector:
- Tamaño (cantidad de empleados):
- Productos o servicios principales:
- Mercado / clientes típicos:
- Facturación aproximada por mes:
- Principales costos:
- Objetivos actuales:
- Equipo / roles:
- Competidores:
- Presencia en redes:
`

// Recargar vuelve a leer los roles del usuario desde disco, sin reiniciar.
// Se usa tras crear/editar/borrar desde la WebUI.
func (m *RolesManager) Recargar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roles = m.roles[:0]
	m.cargarDefaults()
	m.cargarUsuario()
}

// Listar devuelve las etiquetas de todos los roles cargados.
func (m *RolesManager) Listar() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]string, 0, len(m.roles))
	for _, r := range m.roles {
		nombre := r.Etiqueta
		if r.Contexto != "" {
			nombre += " (usa contexto)"
		}
		res = append(res, nombre)
	}
	return res
}

// RolInfo es la vista pública de un rol para el listado del panel.
type RolInfo struct {
	Nombre      string   `json:"nombre"`
	Etiqueta    string   `json:"etiqueta"`
	Descripcion string   `json:"descripcion,omitempty"`
	Activar     []string `json:"activar,omitempty"`
	Contexto    string   `json:"contexto,omitempty"`
}

// ListarDetallado devuelve la información de todos los roles.
func (m *RolesManager) ListarDetallado() []RolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]RolInfo, 0, len(m.roles))
	for _, r := range m.roles {
		res = append(res, RolInfo{
			Nombre:      r.Nombre,
			Etiqueta:    r.Etiqueta,
			Descripcion: r.Descripcion,
			Activar:     r.Activar,
			Contexto:    r.Contexto,
		})
	}
	return res
}

// Obtener devuelve el rol completo por nombre, o false si no existe. Se usa
// para editar desde la WebUI.
func (m *RolesManager) Obtener(nombre string) (Rol, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.roles {
		if r.Nombre == nombre {
			return r, true
		}
	}
	return Rol{}, false
}

// RutaUsuario devuelve la ruta del archivo .md del usuario para un rol, o
// vacío si el nombre no es válido.
func (m *RolesManager) RutaUsuario(nombre string) string {
	if nombre == "" || strings.ContainsAny(nombre, `/\:*?"<>|`) {
		return ""
	}
	return filepath.Join(m.datosDir, "roles", nombre+".md")
}

// CrearOActualizar guarda (o sobrescribe) un rol del usuario y recarga la
// lista en caliente.
func (m *RolesManager) CrearOActualizar(r Rol) error {
	r.Nombre = strings.TrimSpace(r.Nombre)
	r.Etiqueta = strings.TrimSpace(r.Etiqueta)
	r.Prompt = strings.TrimSpace(r.Prompt)
	if r.Nombre == "" {
		return fmt.Errorf("el nombre del rol es obligatorio")
	}
	if r.Etiqueta == "" {
		return fmt.Errorf("la etiqueta del rol es obligatoria")
	}
	if r.Prompt == "" {
		return fmt.Errorf("el rol debe tener un prompt")
	}
	ruta := m.RutaUsuario(r.Nombre)
	if ruta == "" {
		return fmt.Errorf("nombre de rol inválido: %q", r.Nombre)
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(ruta, []byte(serializarRol(r)), 0o600); err != nil {
		return err
	}
	m.Recargar()
	return nil
}

// Eliminar borra un rol del usuario. Si es embebido (sin archivo de usuario),
// no hace nada y devuelve nil.
func (m *RolesManager) Eliminar(nombre string) error {
	ruta := m.RutaUsuario(nombre)
	if ruta == "" {
		return fmt.Errorf("nombre de rol inválido: %q", nombre)
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

// serializarRol arma el archivo .md con front-matter de un rol.
func serializarRol(r Rol) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("nombre: " + r.Nombre + "\n")
	b.WriteString("etiqueta: \"" + r.Etiqueta + "\"\n")
	if r.Descripcion != "" {
		b.WriteString("descripcion: \"" + r.Descripcion + "\"\n")
	}
	if r.Contexto != "" {
		b.WriteString("contexto: " + r.Contexto + "\n")
	}
	claves := make([]string, 0, len(r.Activar))
	for _, k := range r.Activar {
		claves = append(claves, strings.TrimSpace(k))
	}
	b.WriteString("activar: [" + strings.Join(claves, ", ") + "]\n")
	b.WriteString("---\n")
	b.WriteString(r.Prompt + "\n")
	return b.String()
}
