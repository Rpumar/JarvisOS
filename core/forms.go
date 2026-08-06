package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ============================================================
// FORMULARIOS WEB (F3.6)
//
// El dueño define "plantillas de formulario": {campo: valor} que se repiten
// (cargar una factura, pedir un presupuesto, llenar una planilla). Jarvis
// guarda la plantilla, y al pedirlo abre la URL y completa los campos en el
// navegador por autocompletado, y reporta con aprobación.
// ============================================================

// CampoFormulario es una entrada de la plantilla: selector del campo en la
// página y el valor a escribir.
type CampoFormulario struct {
	Nombre   string `json:"nombre"`             // etiqueta legible ("nombre", "email")
	Selector string `json:"selector,omitempty"` // id/name del campo si se conoce
	Valor    string `json:"valor"`
}

// Formulario es una plantilla completa para rellenar una página web.
type Formulario struct {
	Nombre   string             `json:"nombre"`
	URL      string             `json:"url"`
	Campos   []CampoFormulario  `json:"campos"`
	Creado   string             `json:"creado"`
}

// GestorFormularios administra las plantillas persistidas en JSON.
type GestorFormularios struct {
	mu   sync.Mutex
	ruta string
	list map[string]*Formulario
}

// NuevoGestorFormularios carga las plantillas persistidas.
func NuevoGestorFormularios(ruta string) *GestorFormularios {
	g := &GestorFormularios{ruta: ruta, list: make(map[string]*Formulario)}
	g.cargar()
	return g
}

func (g *GestorFormularios) cargar() {
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var lista []*Formulario
	if err := json.Unmarshal(datos, &lista); err != nil {
		return
	}
	for _, f := range lista {
		if f != nil && f.Nombre != "" {
			g.list[strings.ToLower(f.Nombre)] = f
		}
	}
}

func (g *GestorFormularios) guardar() {
	g.mu.Lock()
	defer g.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(g.ruta), 0o700)
	lista := make([]*Formulario, 0, len(g.list))
	for _, f := range g.list {
		lista = append(lista, f)
	}
	datos, _ := json.MarshalIndent(lista, "", "  ")
	_ = os.WriteFile(g.ruta, datos, 0o600)
}

// Obtener devuelve una plantilla por nombre (case-insensitive).
func (g *GestorFormularios) Obtener(nombre string) (*Formulario, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f, ok := g.list[strings.ToLower(strings.TrimSpace(nombre))]
	return f, ok
}

// Listar devuelve los nombres de todas las plantillas.
func (g *GestorFormularios) Listar() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	nombres := make([]string, 0, len(g.list))
	for _, f := range g.list {
		nombres = append(nombres, f.Nombre)
	}
	sortNombres(nombres)
	return nombres
}

// Agregar crea o reemplaza una plantilla por nombre.
func (g *GestorFormularios) Agregar(f Formulario) {
	f.Nombre = strings.TrimSpace(f.Nombre)
	if f.Nombre == "" {
		return
	}
	if f.Creado == "" {
		f.Creado = time.Now().Format("2006-01-02 15:04")
	}
	g.mu.Lock()
	g.list[strings.ToLower(f.Nombre)] = &f
	g.mu.Unlock()
	g.guardar()
}

// CambiarCampo agrega o actualiza un campo de la plantilla. Devuelve si la
// plantilla existía (falso si no se encontró).
func (g *GestorFormularios) CambiarCampo(nombre, campoNombre, valor string) (*Formulario, bool) {
	nombre = strings.TrimSpace(nombre)
	g.mu.Lock()
	f, ok := g.list[strings.ToLower(nombre)]
	if !ok {
		g.mu.Unlock()
		return nil, false
	}
	for i := range f.Campos {
		if strings.EqualFold(f.Campos[i].Nombre, campoNombre) {
			f.Campos[i].Valor = valor
			g.mu.Unlock()
			g.guardar()
			return f, true
		}
	}
	f.Campos = append(f.Campos, CampoFormulario{Nombre: campoNombre, Valor: valor})
	g.mu.Unlock()
	g.guardar()
	return f, true
}

// AgregarCampo agrega un campo nuevo (sin sobrescribir). Devuelve si la
// plantilla existe.
func (g *GestorFormularios) AgregarCampo(nombre, campo, valor string) (*Formulario, bool) {
	nombre = strings.TrimSpace(nombre)
	g.mu.Lock()
	f, ok := g.list[strings.ToLower(nombre)]
	if !ok {
		g.mu.Unlock()
		return nil, false
	}
	for i := range f.Campos {
		if strings.EqualFold(f.Campos[i].Nombre, campo) {
			g.mu.Unlock()
			return f, true // ya existía; no duplicar
		}
	}
	f.Campos = append(f.Campos, CampoFormulario{Nombre: campo, Valor: valor})
	g.mu.Unlock()
	g.guardar()
	return f, true
}

// Eliminar borra una plantilla por nombre.
func (g *GestorFormularios) Eliminar(nombre string) bool {
	nombre = strings.TrimSpace(nombre)
	g.mu.Lock()
	_, ok := g.list[strings.ToLower(nombre)]
	if !ok {
		g.mu.Unlock()
		return false
	}
	delete(g.list, strings.ToLower(nombre))
	g.mu.Unlock()
	g.guardar()
	return true
}

func sortNombres(nombres []string) {
	for i := 1; i < len(nombres); i++ {
		for j := i; j > 0 && nombres[j] < nombres[j-1]; j-- {
			nombres[j], nombres[j-1] = nombres[j-1], nombres[j]
		}
	}
}