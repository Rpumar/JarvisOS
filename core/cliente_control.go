package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ClienteControl es la cara del agente hacia el plano de control en la nube
// (decisión 1, F5+). Todo es best-effort: si el servidor no responde, el
// agente sigue funcionando con la licencia local; los errores se guardan
// para reportarlos por voz sin cortar la operación.
type ClienteControl struct {
	mu           sync.Mutex
	baseURL      string
	http         *http.Client
	clave        string
	idInst       string
	nombre       string
	version      string
	ultimaVers   string
	ultimoMsg    string
	ultimoOK     bool
	ultimoAt     time.Time
}

// NuevoClienteControl crea el cliente con un timeout corto de red (no debe
// colgar el arranque ni el loop del agente).
func NuevoClienteControl(baseURL, clave, idInstalacion, nombre, version string) *ClienteControl {
	return &ClienteControl{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
		clave:   clave,
		idInst:  idInstalacion,
		nombre:  nombre,
		version: version,
	}
}

// Configurado indica si hay un plano de control al cual reportar.
func (c *ClienteControl) Configurado() bool {
	return c != nil && c.baseURL != ""
}

// EstadoReporta devuelve un resumen legible del último contacto, para el
// comando de voz "estado del plano de control".
func (c *ClienteControl) EstadoReporta() string {	if c == nil || c.baseURL == "" {
		return "No hay un plano de control configurado, señor. Los datos siguen siendo 100% locales."
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ultimoAt.IsZero() {
		return fmt.Sprintf("Plano de control configurado en %s, pero aún no se pudo contactar, señor.", c.baseURL)
	}
	estado := "sin respuesta"
	if c.ultimoOK {
		estado = "operativo"
	}
	aviso := c.ultimoMsg
	if avisoActualizacion := c.avisoActualizacion(); avisoActualizacion != "" {
		aviso = aviso + " " + avisoActualizacion
	}
	return fmt.Sprintf("Plano de control en %s: %s (último contacto %s). %s", c.baseURL, estado, c.ultimoAt.Format("15:04:05"), aviso)
}

// avisoActualizacion compara la versión local con la última publicada por el
// servidor; devuelve un aviso si hay una actualización, o vacío.
func (c *ClienteControl) avisoActualizacion() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ultimaVers == "" || c.ultimaVers == c.version {
		return ""
	}
	return fmt.Sprintf("Hay una actualización disponible: versión %s (estás en %s).", c.ultimaVers, c.version)
}

// Activar registra esta instalación contra la licencia. Best-effort: devuelve
// el mensaje del servidor o un error, sin lanzar.
func (c *ClienteControl) Activar() error {
	if c == nil || c.baseURL == "" {
		return nil
	}
	cuerpo, err := json.Marshal(map[string]interface{}{
		"clave":          c.clave,
		"id_instalacion": c.idInst,
		"nombre":         c.nombre,
		"version":        c.version,
	})
	if err != nil {
		return err
	}
	var respuesta struct {
		Ok      bool   `json:"ok"`
		Plan    string `json:"plan"`
		Puestos int    `json:"puestos"`
		Error   string `json:"error"`
	}
	status, err := c.llamar("POST", "/api/v1/activar", cuerpo, &respuesta)
	c.registrar(resultado{err: err, status: status, msg: respuesta.Error, ok: respuesta.Ok})
	if err != nil {
		return fmt.Errorf("no se pudo activar en el plano de control: %w", err)
	}
	if !respuesta.Ok {
		return fmt.Errorf("el plano de control rechazó la activación: %s", respuesta.Error)
	}
	c.registrar(resultado{ok: true, status: status, msg: fmt.Sprintf("Licencia %s activa en el plano de control (%d puestos).", respuesta.Plan, respuesta.Puestos)})
	return nil
}

// Heartbeat reporta estado y puestos en uso. Devuelve si la licencia sigue
// activa según el servidor.
func (c *ClienteControl) Heartbeat(puestosUsados int) (bool, error) {
	if c == nil || c.baseURL == "" {
		return true, nil
	}
	cuerpo, err := json.Marshal(map[string]interface{}{
		"clave":          c.clave,
		"id_instalacion": c.idInst,
		"version":        c.version,
		"puestos_usados": puestosUsados,
	})
	if err != nil {
		return true, err
	}
	var respuesta struct {
		Ok           bool   `json:"ok"`
		Activa       bool   `json:"activa"`
		Plan         string `json:"plan"`
		Puestos      int    `json:"puestos"`
		LatestVersion string `json:"latest_version"`
		Error        string `json:"error"`
	}
	status, err := c.llamar("POST", "/api/v1/heartbeat", cuerpo, &respuesta)
	if err != nil {
		c.registrar(resultado{err: err, status: status, msg: respuesta.Error})
		return true, fmt.Errorf("heartbeat sin respuesta del plano de control: %w", err)
	}
	c.mu.Lock()
	c.ultimaVers = respuesta.LatestVersion
	c.mu.Unlock()
	c.registrar(resultado{ok: respuesta.Ok && respuesta.Activa, status: status, msg: fmt.Sprintf("Heartbeat ok (plan %s, %d puestos).", respuesta.Plan, respuesta.Puestos)})
	return respuesta.Activa, nil
}

type resultado struct {
	err    error
	status int
	msg    string
	ok     bool
}

func (c *ClienteControl) registrar(r resultado) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r.err != nil {
		c.ultimoOK = false
		c.ultimoMsg = "No responde el plano de control: " + r.err.Error()
	} else if r.msg != "" {
		c.ultimoOK = r.ok
		c.ultimoMsg = r.msg
	}
	c.ultimoAt = time.Now()
}

// llamar envía la petición JSON y decodifica la respuesta. Devuelve el status
// HTTP y el error de red (si lo hubo).
func (c *ClienteControl) llamar(metodo, ruta string, cuerpo []byte, destino interface{}) (int, error) {
	req, err := http.NewRequest(metodo, c.baseURL+ruta, bytes.NewReader(cuerpo))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(destino)
	return resp.StatusCode, nil
}

// estadoPlanoControl reporta por voz el estado del plano de control (comando
// "estado del plano de control"). Si no hay plano configurado, aclara que
// todo sigue siendo local.
func (h *Hands) estadoPlanoControl() string {
	if h.Control == nil || !h.Control.Configurado() {
		return "No hay un plano de control configurado, señor. Todo funciona en modo local."
	}
	return h.Control.EstadoReporta()
}
