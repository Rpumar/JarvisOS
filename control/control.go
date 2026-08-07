// Package control implementa el plano de control en la nube (decisión 1,
// fase F5+): un servidor HTTP que emite y gestiona licencias, registra las
// instalaciones del agente (heartbeat) y controla los puestos en uso. Los
// datos viven en archivos JSON con escritura atómica, sin dependencias
// externas (solo librería estándar), coherente con el resto del proyecto.
package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"JarvisOS/core"
)

// Estados de una licencia emitida.
const (
	LicenciaActiva     = "activa"
	LicenciaSuspendida = "suspendida"
)

// Errores de negocio del plano de control.
var (
	ErrClaveNoExiste   = errors.New("la clave de licencia no existe")
	ErrLicenciaInactiva = errors.New("la licencia está suspendida")
	ErrSinPuestosLibres = errors.New("la licencia no tiene puestos libres")
)

// Licencia es una clave emitida por el servidor con su estado y dueño.
type Licencia struct {
	Clave   string `json:"clave"`
	Plan    string `json:"plan"`
	Puestos int    `json:"puestos"`
	Cliente string `json:"cliente"`
	Estado  string `json:"estado"`
	Creada  string `json:"creada"`
}

// Cliente es una instalación del agente registrada contra una licencia.
type Cliente struct {
	IDInstalacion string `json:"id_instalacion"`
	Nombre        string `json:"nombre"`
	Clave         string `json:"clave"`
	Version       string `json:"version"`
	PuestosUsados int    `json:"puestos_usados"`
	UltimoSeen    string `json:"ultimo_seen"`
}

// Gestor almacena y opera licencias y clientes en archivos JSON.
type Gestor struct {
	mu        sync.Mutex
	dir       string
	licencias map[string]*Licencia
	clientes  map[string]*Cliente
	latestVersion string
}

// NuevoGestorControl carga el estado persistido desde dir.
func NuevoGestorControl(dir string) *Gestor {
	g := &Gestor{
		dir:       dir,
		licencias: make(map[string]*Licencia),
		clientes:  make(map[string]*Cliente),
	}
	g.cargar()
	return g
}

func (g *Gestor) cargar() {
	var l map[string]*Licencia
	if leerJSON(filepath.Join(g.dir, "licencias.json"), &l) == nil {
		for k, v := range l {
			g.licencias[k] = v
		}
	}
	var c map[string]*Cliente
	if leerJSON(filepath.Join(g.dir, "clientes.json"), &c) == nil {
		for k, v := range c {
			g.clientes[k] = v
		}
	}
	var v struct {
		LatestVersion string `json:"latest_version"`
	}
	if leerJSON(filepath.Join(g.dir, "version.json"), &v) == nil {
		g.latestVersion = v.LatestVersion
	}
}

// PublicarVersion registra la última versión disponible del agente.
func (g *Gestor) PublicarVersion(version string) {
	version = strings.TrimSpace(version)
	if version == "" {
		return
	}
	g.mu.Lock()
	g.latestVersion = version
	g.mu.Unlock()
	g.guardarVersion()
}

// UltimaVersion devuelve la última versión publicada (vacía si ninguna).
func (g *Gestor) UltimaVersion() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.latestVersion
}

func (g *Gestor) guardarVersion() {
	_ = os.MkdirAll(g.dir, 0o700)
	escribirJSON(filepath.Join(g.dir, "version.json"), map[string]string{"latest_version": g.latestVersion})
}

// Emitir crea una clave nueva para plan/puestos y la registra como activa.
// Devuelve la clave generada (mismo formato local JARVIS-PLAN-PUESTOS-NONCE-
// FIRMA). El cliente es la empresa/comprador al que se le factura.
func (g *Gestor) Emitir(plan string, puestos int, cliente string) (string, error) {
	plan = strings.ToLower(strings.TrimSpace(plan))
	if !core.PlanValido(plan) {
		return "", fmt.Errorf("plan de licencia inválido: %q", plan)
	}
	if puestos <= 0 {
		puestos = core.PuestosPorPlan(plan)
	}
	if cliente == "" {
		cliente = "sin especificar"
	}
	clave, err := core.GenerarLicencia(plan, puestos)
	if err != nil {
		return "", err
	}
	g.mu.Lock()
	g.licencias[clave] = &Licencia{
		Clave:   clave,
		Plan:    plan,
		Puestos: puestos,
		Cliente: cliente,
		Estado:  LicenciaActiva,
		Creada:  time.Now().Format("2006-01-02 15:04"),
	}
	g.guardarLocked()
	g.mu.Unlock()
	return clave, nil
}

// Suspender revoca una licencia: el agente deja de poder activarse y el
// heartbeat devuelve estado suspendida. Devuelve false si la clave no existe.
func (g *Gestor) Suspender(clave string) bool {
	clave = strings.TrimSpace(clave)
	g.mu.Lock()
	l, ok := g.licencias[clave]
	if ok {
		l.Estado = LicenciaSuspendida
		g.guardarLocked()
	}
	g.mu.Unlock()
	return ok
}

// Reactivar vuelve a habilitar una licencia suspendida.
func (g *Gestor) Reactivar(clave string) bool {
	clave = strings.TrimSpace(clave)
	g.mu.Lock()
	l, ok := g.licencias[clave]
	if ok {
		l.Estado = LicenciaActiva
		g.guardarLocked()
	}
	g.mu.Unlock()
	return ok
}

// Activar registra una instalación contra su licencia. Valida que la clave
// exista y esté activa, y que la licencia tenga puestos libres (sin contar
// esta misma instalación). Si la instalación ya estaba registrada, se
// actualiza y no consume un puesto nuevo.
func (g *Gestor) Activar(clave, idInstalacion, nombre, version string) error {
	clave = strings.TrimSpace(clave)
	idInstalacion = strings.TrimSpace(idInstalacion)
	g.mu.Lock()
	defer g.mu.Unlock()

	l, ok := g.licencias[clave]
	if !ok {
		return ErrClaveNoExiste
	}
	if l.Estado != LicenciaActiva {
		return ErrLicenciaInactiva
	}

	if c, existe := g.clientes[idInstalacion]; existe && c.Clave == clave {
		c.Nombre = nombre
		c.Version = version
		c.UltimoSeen = time.Now().Format(time.RFC3339)
		g.guardarLocked()
		return nil
	}

	if !g.tienePuestoLibreLocked(clave, idInstalacion) {
		return ErrSinPuestosLibres
	}

	g.clientes[idInstalacion] = &Cliente{
		IDInstalacion: idInstalacion,
		Nombre:        nombre,
		Clave:         clave,
		Version:       version,
		UltimoSeen:    time.Now().Format(time.RFC3339),
	}
	g.guardarLocked()
	return nil
}

// Heartbeat actualiza el último aviso de una instalación y devuelve si la
// licencia sigue activa, el plan, el tope de puestos y la última versión
// publicada. El agente lo envía periódicamente.
func (g *Gestor) Heartbeat(clave, idInstalacion, version string, puestosUsados int) (activa bool, plan string, topePuestos int, latestVersion string, err error) {
	clave = strings.TrimSpace(clave)
	idInstalacion = strings.TrimSpace(idInstalacion)
	g.mu.Lock()
	defer g.mu.Unlock()

	l, ok := g.licencias[clave]
	if !ok {
		return false, "", 0, "", ErrClaveNoExiste
	}
	if c, existe := g.clientes[idInstalacion]; existe && c.Clave == clave {
		c.Version = version
		c.PuestosUsados = puestosUsados
		c.UltimoSeen = time.Now().Format(time.RFC3339)
		g.guardarLocked()
	}
	if l.Estado != LicenciaActiva {
		return false, l.Plan, l.Puestos, g.latestVersion, ErrLicenciaInactiva
	}
	return true, l.Plan, l.Puestos, g.latestVersion, nil
}

// tienePuestoLibreLocked indica si la licencia aún acepta instalaciones
// nuevas: puestos contratados menos clientes registrados (excluyendo la
// instalación que consulta, que podría estar re-activándose).
func (g *Gestor) tienePuestoLibreLocked(clave, idInstalacion string) bool {
	l := g.licencias[clave]
	if l == nil {
		return false
	}
	usados := 0
	for id, c := range g.clientes {
		if c.Clave == clave && id != idInstalacion {
			usados++
		}
	}
	return usados < l.Puestos
}

// Licencias devuelve la lista ordenada de licencias emitidas.
func (g *Gestor) Licencias() []Licencia {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Licencia, 0, len(g.licencias))
	for _, l := range g.licencias {
		out = append(out, *l)
	}
	return out
}

// Clientes devuelve la lista de instalaciones registradas.
func (g *Gestor) Clientes() []Cliente {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Cliente, 0, len(g.clientes))
	for _, c := range g.clientes {
		out = append(out, *c)
	}
	return out
}

// guardarLocked persiste el estado actual. Debe llamarse con g.mu tomada.
func (g *Gestor) guardarLocked() {
	_ = os.MkdirAll(g.dir, 0o700)
	escribirJSON(filepath.Join(g.dir, "licencias.json"), g.licencias)
	escribirJSON(filepath.Join(g.dir, "clientes.json"), g.clientes)
}

// leerJSON lee un archivo JSON (ausente = vacío sin error).
func leerJSON(ruta string, destino interface{}) error {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(datos, destino)
}

// escribirJSON guarda con escritura atómica (temp + rename) y permisos 0600.
func escribirJSON(ruta string, valor interface{}) {
	datos, err := json.MarshalIndent(valor, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(ruta), 0o700)
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, ruta)
}
