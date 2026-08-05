package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PerfilEmpresa es el perfil estructurado de la empresa del dueño. Jarvis lo
// usa para asesorar como CEO, especialista en marketing y asistente
// corporativo, injertándolo en el contexto de la IA.
type PerfilEmpresa struct {
	Nombre          string   `json:"nombre"`            // Nombre de la empresa
	Rubro           string   `json:"rubro"`             // Rubro / sector
	Descripcion     string   `json:"descripcion"`       // Qué hace la empresa
	Tamano          string   `json:"tamano"`            // Cantidad de empleados
	Productos       []string `json:"productos"`         // Productos o servicios principales
	Clientes        []string `json:"clientes"`          // Mercado / clientes típicos
	Facturacion     string   `json:"facturacion"`       // Facturación aproximada por mes
	Costos          string   `json:"costos"`            // Principales costos
	Objetivos       []string `json:"objetivos"`         // Objetivos actuales
	Equipo          string   `json:"equipo"`            // Equipo / roles
	Competidores    []string `json:"competidores"`      // Competidores
	Redes           []string `json:"redes"`             // Presencia en redes
	ContactoDueno   string   `json:"contacto_dueno"`    // Nombre del dueño
	ContactoMail    string   `json:"contacto_mail"`     // Email de contacto con clientes
	ContactoTelefono string  `json:"contacto_telefono"` // Teléfono/WhatsApp
}

// EstaVacia informa si el perfil aún no se cargó (ningún campo sustancial).
func (p PerfilEmpresa) EstaVacia() bool {
	vacio := []string{p.Nombre, p.Rubro, p.Descripcion, p.Tamano, p.Facturacion, p.Costos, p.Equipo, p.ContactoDueno, p.ContactoMail, p.ContactoTelefono}
	for _, s := range vacio {
		if strings.TrimSpace(s) != "" {
			return false
		}
	}
	return len(p.Productos)+len(p.Clientes)+len(p.Objetivos)+len(p.Competidores)+len(p.Redes) == 0
}

// GestorEmpresa persiste el perfil de la empresa en la carpeta de datos del
// usuario. Usa escritura atómica (temp + rename) para no corromper el archivo.
type GestorEmpresa struct {
	mu   sync.Mutex
	ruta string
	perf PerfilEmpresa
}

func NuevoGestorEmpresa(ruta string) *GestorEmpresa {
	g := &GestorEmpresa{ruta: ruta}
	g.cargar()
	return g
}

func (g *GestorEmpresa) cargar() {
	g.mu.Lock()
	defer g.mu.Unlock()
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var p PerfilEmpresa
	if json.Unmarshal(datos, &p) != nil {
		return
	}
	g.perf = p
}

func (g *GestorEmpresa) guardar() error {
	g.perf.normalizar()
	datos, err := json.MarshalIndent(g.perf, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(g.ruta), 0o700); err != nil {
		return err
	}
	tmp := g.ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, g.ruta)
}

// Obtener devuelve una copia del perfil.
func (g *GestorEmpresa) Obtener() PerfilEmpresa {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.perf
}

// UltimaModificacion devuelve el mtime del archivo o cero si no existe.
func (g *GestorEmpresa) UltimaModificacion() (time.Time, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	st, err := os.Stat(g.ruta)
	if err != nil {
		return time.Time{}, false
	}
	return st.ModTime(), true
}

// Reemplazar guarda un perfil completo desde la WebUI.
func (g *GestorEmpresa) Reemplazar(p PerfilEmpresa) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.perf = p
	return g.guardar()
}

// SetCampo asigna un campo simple del perfil por su clave ("nombre", "rubro",
// "tamano", "facturacion", "costos", "equipo", "dueno", "email", "telefono",
// "descripcion"). Devuelve un error si la clave no existe.
func (g *GestorEmpresa) SetCampo(clave, valor string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch strings.ToLower(strings.TrimSpace(clave)) {
	case "nombre":
		g.perf.Nombre = strings.TrimSpace(valor)
	case "rubro":
		g.perf.Rubro = strings.TrimSpace(valor)
	case "descripcion":
		g.perf.Descripcion = strings.TrimSpace(valor)
	case "tamano":
		g.perf.Tamano = strings.TrimSpace(valor)
	case "facturacion":
		g.perf.Facturacion = strings.TrimSpace(valor)
	case "costos":
		g.perf.Costos = strings.TrimSpace(valor)
	case "equipo":
		g.perf.Equipo = strings.TrimSpace(valor)
	case "dueno":
		g.perf.ContactoDueno = strings.TrimSpace(valor)
	case "email":
		g.perf.ContactoMail = strings.TrimSpace(valor)
	case "telefono":
		g.perf.ContactoTelefono = strings.TrimSpace(valor)
	default:
		return fmt.Errorf("campo desconocido %q", clave)
	}
	return g.guardar()
}

// AgregarItem añade un elemento a una lista del perfil (sin duplicados).
func (g *GestorEmpresa) AgregarItem(lista, valor string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	val := strings.TrimSpace(valor)
	if val == "" {
		return fmt.Errorf("valor vacío")
	}
	switch strings.ToLower(strings.TrimSpace(lista)) {
	case "productos":
		g.perf.Productos = appendSinDuplicar(g.perf.Productos, val)
	case "clientes":
		g.perf.Clientes = appendSinDuplicar(g.perf.Clientes, val)
	case "objetivos":
		g.perf.Objetivos = appendSinDuplicar(g.perf.Objetivos, val)
	case "competidores":
		g.perf.Competidores = appendSinDuplicar(g.perf.Competidores, val)
	case "redes":
		g.perf.Redes = appendSinDuplicar(g.perf.Redes, val)
	default:
		return fmt.Errorf("no existe una lista llamada %q. Las listas son: productos, clientes, objetivos, competidores y redes", lista)
	}
	return g.guardar()
}

// BorrarItem elimina un elemento de una lista del perfil (por valor).
func (g *GestorEmpresa) BorrarItem(lista, valor string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	filter := func(items []string) []string {
		out := items[:0]
		for _, it := range items {
			if !normalizadasIgualesLista(it, valor) {
				out = append(out, it)
			}
		}
		return out
	}
	switch strings.ToLower(strings.TrimSpace(lista)) {
	case "productos":
		g.perf.Productos = filter(g.perf.Productos)
	case "clientes":
		g.perf.Clientes = filter(g.perf.Clientes)
	case "objetivos":
		g.perf.Objetivos = filter(g.perf.Objetivos)
	case "competidores":
		g.perf.Competidores = filter(g.perf.Competidores)
	case "redes":
		g.perf.Redes = filter(g.perf.Redes)
	default:
		return fmt.Errorf("no existe una lista llamada %q", lista)
	}
	return g.guardar()
}

func appendSinDuplicar(lista []string, valor string) []string {
	v := strings.ToLower(strings.TrimSpace(valor))
	for _, it := range lista {
		if strings.ToLower(strings.TrimSpace(it)) == v {
			return lista
		}
	}
	return append(lista, strings.TrimSpace(valor))
}

func normalizadasIgualesLista(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// normalizar asegura que las listas sean slices no-nil y limpia espacios.
func (p *PerfilEmpresa) normalizar() {
	if p.Productos == nil {
		p.Productos = []string{}
	}
	if p.Clientes == nil {
		p.Clientes = []string{}
	}
	if p.Objetivos == nil {
		p.Objetivos = []string{}
	}
	if p.Competidores == nil {
		p.Competidores = []string{}
	}
	if p.Redes == nil {
		p.Redes = []string{}
	}
}

// Resumen devuelve una vista textual corta del perfil. Si está vacío, explica
// cómo cargarlo.
func (g *GestorEmpresa) Resumen() string {
	p := g.Obtener()
	if p.EstaVacia() {
		return "Todavía no tengo cargado el perfil de tu empresa, señor. Puede decírmelo así: 'mi empresa se llama X', 'mi rubro es Y', 'agendá mi objetivo Z'. También puede completarlo en el panel web."
	}
	return p.Resumen()
}

// Resumen arma el texto legible del perfil.
func (p *PerfilEmpresa) Resumen() string {
	var b strings.Builder
	campos := p.campos()
	for i, c := range campos {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- " + c)
	}
	secciones := []struct {
		titulo string
		items  []string
	}{
		{"Productos: ", p.Productos},
		{"Clientes típicos: ", p.Clientes},
		{"Objetivos: ", p.Objetivos},
		{"Competidores: ", p.Competidores},
		{"Presencia en redes: ", p.Redes},
	}
	for _, s := range secciones {
		if len(s.items) > 0 {
			b.WriteString("\n- " + s.titulo + strings.Join(s.items, ", "))
		}
	}
	return b.String()
}

// TextoParaIA devuelve el perfil formateado para inyectar como contexto en la IA.
func (g *GestorEmpresa) TextoParaIA() string {
	p := g.Obtener()
	if p.EstaVacia() {
		return ""
	}
	return "[PERFIL DE LA EMPRESA]\n" + p.Resumen() + "\n[/PERFIL DE LA EMPRESA]"
}

// campos lista los campos escalares no vacíos como "Clave: valor".
func (p *PerfilEmpresa) campos() []string {
	var res []string
	add := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			res = append(res, k+": "+strings.TrimSpace(v))
		}
	}
	add("Nombre", p.Nombre)
	add("Rubro", p.Rubro)
	add("Descripción", p.Descripcion)
	add("Tamaño", p.Tamano)
	add("Facturación", p.Facturacion)
	add("Costos", p.Costos)
	add("Equipo", p.Equipo)
	add("Dueño", p.ContactoDueno)
	add("Email", p.ContactoMail)
	add("Teléfono", p.ContactoTelefono)
	return res
}
