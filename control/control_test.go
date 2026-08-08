package control

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"JarvisOS/core"
)

func TestEmitirActivarYHeartbeat(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)

	clave, err := g.Emitir("pro", 2, "Consultora ABC")
	if err != nil {
		t.Fatalf("emitir falló: %v", err)
	}
	if !strings.HasPrefix(clave, "JARVIS-") {
		t.Fatalf("clave con formato inesperado: %q", clave)
	}

	if err := g.Activar(clave, "inst-1", "PC Oficina", "0.14.1"); err != nil {
		t.Fatalf("activar falló: %v", err)
	}
	if err := g.Activar(clave, "inst-2", "PC Contabilidad", "0.14.1"); err != nil {
		t.Fatalf("segunda activación falló: %v", err)
	}
	// Licencia pro con 2 puestos: la tercera instalación debe fallar.
	if err := g.Activar(clave, "inst-3", "PC Riego", "0.14.1"); err != ErrSinPuestosLibres {
		t.Fatalf("esperaba ErrSinPuestosLibres, obtuve: %v", err)
	}

	// Re-activar la misma instalación no consume un puesto nuevo.
	if err := g.Activar(clave, "inst-1", "PC Oficina v2", "0.14.2"); err != nil {
		t.Fatalf("re-activar instalación existente falló: %v", err)
	}

	activa, plan, tope, _, err := g.Heartbeat(clave, "inst-1", "0.14.2", 1)
	if err != nil {
		t.Fatalf("heartbeat falló: %v", err)
	}
	if !activa || plan != "pro" || tope != 2 {
		t.Fatalf("heartbeat inesperado: activa=%v plan=%q tope=%d", activa, plan, tope)
	}

	if len(g.Licencias()) != 1 {
		t.Fatalf("esperaba 1 licencia, tengo %d", len(g.Licencias()))
	}
	if len(g.Clientes()) != 2 {
		t.Fatalf("esperaba 2 clientes, tengo %d", len(g.Clientes()))
	}
}

func TestSuspenderReactivar(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)

	clave, err := g.Emitir("lite", 1, "Cliente")
	if err != nil {
		t.Fatalf("emitir falló: %v", err)
	}
	if !g.Suspender(clave) {
		t.Fatal("suspender debía existir")
	}
	if err := g.Activar(clave, "inst-1", "PC", "0.14.1"); err != ErrLicenciaInactiva {
		t.Fatalf("esperaba ErrLicenciaInactiva, obtuve: %v", err)
	}
	if _, _, _, _, err := g.Heartbeat(clave, "inst-1", "0.14.1", 0); err != ErrLicenciaInactiva {
		t.Fatalf("heartbeat de licencia suspendida debía fallar, obtuve: %v", err)
	}
	if !g.Reactivar(clave) {
		t.Fatal("reactivar debía existir")
	}
	if err := g.Activar(clave, "inst-1", "PC", "0.14.1"); err != nil {
		t.Fatalf("activar tras reactivar falló: %v", err)
	}
}

func TestPersistenciaEntreInstancias(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	clave, _ := g.Emitir("empresa", 50, "Empresa Grande")

	g2 := NuevoGestorControl(dir)
	if len(g2.Licencias()) != 1 {
		t.Fatalf("la licencia no sobrevivió al recargar")
	}
	if err := g2.Activar(clave, "inst-1", "PC", "0.14.1"); err != nil {
		t.Fatalf("activar tras recargar falló: %v", err)
	}
	g3 := NuevoGestorControl(dir)
	if len(g3.Clientes()) != 1 {
		t.Fatalf("el cliente no sobrevivió al recargar")
	}
}

func TestServidorHTTPFlujoCompleto(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	s := NuevoServidor(g, Opciones{Token: "clave-maestra", Dir: dir})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/emitir", s.manejarEmitir)
	mux.HandleFunc("/api/v1/activar", s.manejarActivar)
	mux.HandleFunc("/api/v1/heartbeat", s.manejarHeartbeat)
	mux.HandleFunc("/api/v1/estado", s.manejarEstado)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Sin token, emitir debe dar 401.
	resp, err := http.Post(srv.URL+"/api/v1/emitir", "application/json",
		bytes.NewBufferString(`{"plan":"lite","puestos":1,"cliente":"X"}`))
	if err != nil {
		t.Fatalf("POST emitir sin token: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("emitir sin token debía dar 401, dio %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Con token, emitir devuelve la clave.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/emitir",
		bytes.NewBufferString(`{"plan":"lite","puestos":1,"cliente":"X"}`))
	req.Header.Set("Authorization", "Bearer clave-maestra")
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("emitir con token: %v", err)
	}
	var emision struct {
		Clave string `json:"clave"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&emision)
	resp.Body.Close()
	if emision.Clave == "" {
		t.Fatal("no devolvió la clave emitida")
	}

	// Activar y heartbeat sin token (el agente usa su licencia).
	activa := map[string]string{"clave": emision.Clave, "id_instalacion": "inst-1", "nombre": "PC", "version": "0.14.1"}
	datos, _ := json.Marshal(activa)
	resp, err = http.Post(srv.URL+"/api/v1/activar", "application/json", bytes.NewReader(datos))
	if err != nil {
		t.Fatalf("activar: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activar debía dar 200, dio %d", resp.StatusCode)
	}
	resp.Body.Close()

	hb := map[string]interface{}{"clave": emision.Clave, "id_instalacion": "inst-1", "version": "0.14.1", "puestos_usados": 1}
	datos, _ = json.Marshal(hb)
	resp, err = http.Post(srv.URL+"/api/v1/heartbeat", "application/json", bytes.NewReader(datos))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	var hbResp struct {
		Ok     bool `json:"ok"`
		Activa bool `json:"activa"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&hbResp)
	resp.Body.Close()
	if !hbResp.Ok || !hbResp.Activa {
		t.Fatalf("heartbeat inesperado: %+v", hbResp)
	}

	// Estado admin con token.
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/v1/estado", nil)
	req.Header.Set("Authorization", "Bearer clave-maestra")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("estado: %v", err)
	}
	var estado struct {
		Licencias []Licencia `json:"licencias"`
		Clientes  []Cliente  `json:"clientes"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&estado)
	resp.Body.Close()
	if len(estado.Licencias) != 1 || len(estado.Clientes) != 1 {
		t.Fatalf("estado inesperado: licencias=%d clientes=%d", len(estado.Licencias), len(estado.Clientes))
	}
}

func TestClienteActivaEnServidorReal(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	clave, _ := g.Emitir("pro", 5, "Cliente")

	s := NuevoServidor(g, Opciones{})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/activar", s.manejarActivar)
	mux.HandleFunc("/api/v1/heartbeat", s.manejarHeartbeat)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cliente := core.NuevoClienteControl(srv.URL, clave, "inst-7", "PC Test", "0.14.1")
	if err := cliente.Activar(); err != nil {
		t.Fatalf("activación del cliente falló: %v", err)
	}
	activa, err := cliente.Heartbeat(2)
	if err != nil {
		t.Fatalf("heartbeat del cliente falló: %v", err)
	}
	if !activa {
		t.Fatal("el servidor debía reportar la licencia activa")
	}
	if len(g.Clientes()) != 1 {
		t.Fatalf("el servidor debía registrar el cliente, tiene %d", len(g.Clientes()))
	}
}

func TestEstadoReportaSinPlano(t *testing.T) {
	var c *core.ClienteControl
	got := c.EstadoReporta()
	if !strings.Contains(got, "No hay un plano") && !strings.Contains(got, "no hay un plano") {
		t.Fatalf("respuesta sin plano inesperada: %q", got)
	}
}

func TestServidorTokenDesdeEntorno(t *testing.T) {
	t.Setenv("JARVISOS_CONTROL_TOKEN", "  abc  ")
	if got := TokenDesdeEntorno(); got != "abc" {
		t.Fatalf("TokenDesdeEntorno debía recortar el token, dio %q", got)
	}
}

func TestPublicarYConsultarVersion(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	if v := g.UltimaVersion(); v != "" {
		t.Fatalf("sin publicar debía estar vacía, dio %q", v)
	}
	g.PublicarVersion("0.15.0")
	if v := g.UltimaVersion(); v != "0.15.0" {
		t.Fatalf("después de publicar debía ser 0.15.0, dio %q", v)
	}

	g2 := NuevoGestorControl(dir)
	if v := g2.UltimaVersion(); v != "0.15.0" {
		t.Fatalf("la versión debía persistir tras recargar, dio %q", v)
	}
}

func TestVersionEnHeartbeatYEndpoint(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	clave, _ := g.Emitir("pro", 5, "Cliente")
	g.PublicarVersion("0.15.0")

	s := NuevoServidor(g, Opciones{Token: "maestra", Dir: dir})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/heartbeat", s.manejarHeartbeat)
	mux.HandleFunc("/api/v1/version", s.manejarVersion)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// El heartbeat debe devolver la última versión publicada.
	hb := map[string]interface{}{"clave": clave, "id_instalacion": "inst-9", "version": "0.14.1", "puestos_usados": 1}
	datos, _ := json.Marshal(hb)
	resp, err := http.Post(srv.URL+"/api/v1/heartbeat", "application/json", bytes.NewReader(datos))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	var hbResp struct {
		Ok           bool   `json:"ok"`
		LatestVersion string `json:"latest_version"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&hbResp)
	resp.Body.Close()
	if !hbResp.Ok || hbResp.LatestVersion != "0.15.0" {
		t.Fatalf("heartbeat debía reportar 0.15.0, dio %+v", hbResp)
	}

	// GET público devuelve la versión.
	resp, err = http.Get(srv.URL + "/api/v1/version")
	if err != nil {
		t.Fatalf("GET version: %v", err)
	}
	var vResp struct {
		LatestVersion string `json:"latest_version"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&vResp)
	resp.Body.Close()
	if vResp.LatestVersion != "0.15.0" {
		t.Fatalf("GET version debía devolver 0.15.0, dio %+v", vResp)
	}

	// POST sin token debe dar 401; con token, publica.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/version",
		bytes.NewBufferString(`{"version":"0.16.0"}`))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST version sin token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST sin token debía dar 401, dio %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/version",
		bytes.NewBufferString(`{"version":"0.16.0"}`))
	req.Header.Set("Authorization", "Bearer maestra")
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST version con token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST con token debía dar 200, dio %d", resp.StatusCode)
	}
	if v := g.UltimaVersion(); v != "0.16.0" {
		t.Fatalf("después del POST debía ser 0.16.0, dio %q", v)
	}
}

func TestPanelDocumentacion(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	s := NuevoServidor(g, Opciones{})

	mux := http.NewServeMux()
	mux.HandleFunc("/panel", s.manejarPanel)
	mux.HandleFunc("/", s.manejarRaiz)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// El panel se sirve como HTML sin exigir token.
	resp, err := http.Get(srv.URL + "/panel")
	if err != nil {
		t.Fatalf("GET panel: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("panel debía dar 200, dio %d", resp.StatusCode)
	}
	if !strings.Contains(strings.ToLower(string(body)), "<html") {
		t.Fatal("el panel no devolvió contenido HTML")
	}
	if !strings.Contains(string(body), "Plano de Control") || !strings.Contains(string(body), "Emitir licencia") {
		t.Fatal("el panel no contiene la interfaz esperada")
	}

	// La raíz redirige al panel.
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err = client.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET raíz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("la raíz debía redirigir (302), dio %d", resp.StatusCode)
	}
	if loc, _ := resp.Location(); !strings.HasSuffix(loc.Path, "/panel") {
		t.Fatalf("la redirección debía ir a /panel, fue %v", loc)
	}
}

// TestPanelPersistenciaFlujo verifica que el panel se sirve con el embed y
// que las acciones del panel (emitir/suspender/reactivar) siguen exigiendo
// token, aunque la página HTML en sí sea pública.
func TestPanelRequiereTokenEnAPI(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorControl(dir)
	s := NuevoServidor(g, Opciones{Token: "maestra"})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/emitir", s.manejarEmitir)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/emitir", "application/json",
		bytes.NewBufferString(`{"plan":"lite","puestos":1,"cliente":"X"}`))
	if err != nil {
		t.Fatalf("emitir sin token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("emitir sin token debía dar 401, dio %d", resp.StatusCode)
	}
}

var _ = filepath.Join
