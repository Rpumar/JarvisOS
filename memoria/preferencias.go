package memoria

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Preferencias struct {
	Nombre      string            `json:"nombre"`
	AppFrecuente map[string]int   `json:"app_frecuente"`
	UltimoProyecto string         `json:"ultimo_proyecto"`
	UltimoComando string          `json:"ultimo_comando"`
	Tema         string            `json:"tema"`
	VozActivada  bool              `json:"voz_activada"`
	Volumen      int               `json:"volumen"`
	Brillo       int               `json:"brillo"`
	UltimaSesion string            `json:"ultima_sesion"`
	VecesUsado   int               `json:"veces_usado"`
	ComandosFrecuentes map[string]int `json:"comandos_frecuentes"`
}

type GestorPreferencias struct {
	mu   sync.RWMutex
	ruta string
	pref Preferencias
}

func NuevoGestorPreferencias(ruta string) *GestorPreferencias {
	g := &GestorPreferencias{ruta: ruta}
	g.pref = Preferencias{
		AppFrecuente:       make(map[string]int),
		ComandosFrecuentes: make(map[string]int),
		VozActivada:        true,
		Volumen:            50,
		Brillo:             50,
	}
	g.cargar()
	return g
}

func (g *GestorPreferencias) cargar() {
	datos, err := os.ReadFile(g.ruta)
	if err != nil {
		return
	}
	var p Preferencias
	if err := json.Unmarshal(datos, &p); err != nil {
		return
	}
	if p.AppFrecuente == nil {
		p.AppFrecuente = make(map[string]int)
	}
	if p.ComandosFrecuentes == nil {
		p.ComandosFrecuentes = make(map[string]int)
	}
	g.pref = p
}

func (g *GestorPreferencias) guardar() {
	g.pref.UltimaSesion = time.Now().Format(time.RFC3339)
	g.pref.VecesUsado++
	if err := os.MkdirAll(filepath.Dir(g.ruta), 0o700); err != nil {
		return
	}
	datos, _ := json.MarshalIndent(g.pref, "", "  ")
	os.WriteFile(g.ruta, datos, 0o600)
}

func (g *GestorPreferencias) Get() Preferencias {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pref
}

func (g *GestorPreferencias) SetNombre(nombre string) {
	g.mu.Lock()
	g.pref.Nombre = nombre
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) RegistrarApp(nombre string) {
	g.mu.Lock()
	g.pref.AppFrecuente[nombre]++
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) AppMasUsada() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var maxApp string
	maxCount := 0
	for app, count := range g.pref.AppFrecuente {
		if count > maxCount {
			maxCount = count
			maxApp = app
		}
	}
	return maxApp
}

func (g *GestorPreferencias) SetUltimoProyecto(ruta string) {
	g.mu.Lock()
	g.pref.UltimoProyecto = ruta
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) RegistrarComando(comando string) {
	g.mu.Lock()
	g.pref.ComandosFrecuentes[comando]++
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) SetTema(tema string) {
	g.mu.Lock()
	g.pref.Tema = tema
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) SetVoz(activada bool) {
	g.mu.Lock()
	g.pref.VozActivada = activada
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) SetVolumen(nivel int) {
	g.mu.Lock()
	g.pref.Volumen = nivel
	g.mu.Unlock()
	g.guardar()
}

func (g *GestorPreferencias) String() string {
	p := g.Get()
	return fmt.Sprintf("Señor, le he visto %d veces. Su app favorita es '%s'. Último proyecto: %s",
		p.VecesUsado, g.AppMasUsada(), p.UltimoProyecto)
}
