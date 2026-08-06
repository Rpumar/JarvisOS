package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"JarvisOS/core"
	"JarvisOS/core/audit"
	"JarvisOS/core/security"
)

type fakeBrain struct {
	respuesta string
	recibido  string
}

func (f *fakeBrain) Process(input string) string {
	f.recibido = input
	return f.respuesta
}

type fakeEstado struct {
	e core.EstadoPanel
}

func (f *fakeEstado) EstadoPanel() core.EstadoPanel { return f.e }

type fakeDiagnostico struct {
	d   *core.Diagnostico
	err error
}

func (f *fakeDiagnostico) Diagnostico() (*core.Diagnostico, error) { return f.d, f.err }

type fakeAprobadorConDatos struct {
	ordenes []core.Orden
}

func (f *fakeAprobadorConDatos) AprobarOrden(id int, pin string) string {
	return "aprobada con " + pin
}
func (f *fakeAprobadorConDatos) DenegarOrden(id int) string { return "denegada" }
func (f *fakeAprobadorConDatos) OrdenesParaPanel() []core.Orden {
	return f.ordenes
}

func TestManejarChatRespuesta(t *testing.T) {
	brain := &fakeBrain{respuesta: "hola señor"}
	s := NuevoServidor(brain, 0)
	body := `{"message":"hola"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	s.manejarChat(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("chat: status %d", rr.Code)
	}
	var res ChatResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Response != "hola señor" {
		t.Errorf("respuesta = %q", res.Response)
	}
	if brain.recibido != "hola" {
		t.Errorf("el mensaje no llegó al cerebro: %q", brain.recibido)
	}
	if len(s.historial) != 1 {
		t.Errorf("chat debería guardar el turno en el historial, hay %d", len(s.historial))
	}
}

func TestManejarChatMetodoIncorrecto(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	rr := httptest.NewRecorder()
	s.manejarChat(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/chat debería ser 405, fue %d", rr.Code)
	}
}

func TestManejarChatCuerpoInvalido(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString("{no json"))
	rr := httptest.NewRecorder()
	s.manejarChat(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("cuerpo inválido debería ser 400, fue %d", rr.Code)
	}
}

func TestManejarChatMensajeVacio(t *testing.T) {
	s := NuevoServidor(&fakeBrain{respuesta: "no debería llamarse"}, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":""}`))
	rr := httptest.NewRecorder()
	s.manejarChat(rr, req)
	var res ChatResponse
	json.NewDecoder(rr.Body).Decode(&res)
	if res.Response != "" {
		t.Errorf("mensaje vacío debería responder vacío, fue %q", res.Response)
	}
}

func TestManejarHistorialYLimpiar(t *testing.T) {
	s := NuevoServidor(&fakeBrain{respuesta: "ok"}, 0)
	s.manejarChat(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"a"}`)))
	s.manejarChat(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"b"}`)))

	req := httptest.NewRequest(http.MethodGet, "/api/historial", nil)
	rr := httptest.NewRecorder()
	s.manejarHistorial(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("historial: status %d", rr.Code)
	}
	var entries []HistorialEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("historial = %d entradas, esperaba 2", len(entries))
	}
	if entries[0].Usuario != "a" || entries[0].Jarvis != "ok" {
		t.Errorf("entrada 0 incorrecta: %+v", entries[0])
	}
	if entries[0].Timestamp == "" {
		t.Error("las entradas deberían tener timestamp")
	}

	lim := httptest.NewRequest(http.MethodPost, "/api/limpiar", nil)
	rr2 := httptest.NewRecorder()
	s.manejarLimpiar(rr2, lim)
	if rr2.Code != http.StatusOK {
		t.Fatalf("limpiar: status %d", rr2.Code)
	}
	if len(s.historial) != 0 {
		t.Error("tras limpiar el historial debería estar vacío")
	}
}

func TestManejarHistorialPersistidoEnDisco(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "historial.json")
	s := NuevoServidor(&fakeBrain{respuesta: "ok"}, 0, ServidorOpciones{RutaHistorial: ruta})
	s.manejarChat(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"hola"}`)))

	s2 := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{RutaHistorial: ruta})
	if len(s2.historial) != 1 || s2.historial[0].Usuario != "hola" {
		t.Errorf("historial no se recuperó del disco: %+v", s2.historial)
	}
}

func TestManejarEstado(t *testing.T) {
	s := NuevoServidor(nil, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/estado", nil)
	rr := httptest.NewRecorder()
	s.manejarEstado(rr, req)
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["error"] != "sin estado" {
		t.Errorf("sin proveedor de estado debería responder 'sin estado', fue %v", res)
	}

	conDatos := NuevoServidor(nil, 0, ServidorOpciones{Estado: &fakeEstado{e: core.EstadoPanel{Hora: "12:00", RAMTotal: "16GB"}}})
	rr2 := httptest.NewRecorder()
	conDatos.manejarEstado(rr2, req)
	var estado core.EstadoPanel
	if err := json.NewDecoder(rr2.Body).Decode(&estado); err != nil {
		t.Fatal(err)
	}
	if estado.Hora != "12:00" || estado.RAMTotal != "16GB" {
		t.Errorf("estado = %+v", estado)
	}
}

func TestManejarDiagnostico(t *testing.T) {
	s := NuevoServidor(nil, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/diagnostico", nil)
	rr := httptest.NewRecorder()
	s.manejarDiagnostico(rr, req)

	conDatos := NuevoServidor(nil, 0, ServidorOpciones{Diagnostico: &fakeDiagnostico{d: &core.Diagnostico{OS: "Windows", Version: "11"}}})
	rr2 := httptest.NewRecorder()
	conDatos.manejarDiagnostico(rr2, req)
	var diag core.Diagnostico
	if err := json.NewDecoder(rr2.Body).Decode(&diag); err != nil {
		t.Fatal(err)
	}
	if diag.OS != "Windows" || diag.Version != "11" {
		t.Errorf("diagnostico = %+v", diag)
	}

	conError := NuevoServidor(nil, 0, ServidorOpciones{Diagnostico: &fakeDiagnostico{err: os.ErrNotExist}})
	rr3 := httptest.NewRecorder()
	conError.manejarDiagnostico(rr3, req)
	var res map[string]string
	json.NewDecoder(rr3.Body).Decode(&res)
	if res["error"] == "" {
		t.Error("un error del proveedor debería reportarse en el JSON")
	}
}

func TestManejarSalud(t *testing.T) {
	s := NuevoServidor(nil, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/salud", nil)
	rr := httptest.NewRecorder()
	s.manejarSalud(rr, req)

	conDatos := NuevoServidor(nil, 0, ServidorOpciones{Diagnostico: &fakeDiagnostico{d: &core.Diagnostico{OS: "Windows", RAMPorcentaje: 20, CPUPorcentaje: 10}}})
	rr2 := httptest.NewRecorder()
	conDatos.manejarSalud(rr2, req)
	var salud map[string]interface{}
	if err := json.NewDecoder(rr2.Body).Decode(&salud); err != nil {
		t.Fatal(err)
	}
	if salud["puntaje"] == nil {
		t.Error("salud debería incluir puntaje")
	}
}

func TestManejarOrdenes(t *testing.T) {
	s := NuevoServidor(nil, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/ordenes", nil)
	rr := httptest.NewRecorder()
	s.manejarOrdenes(rr, req)

	aprob := &fakeAprobadorConDatos{ordenes: []core.Orden{
		{ID: 3, Objetivo: "limpiar temp", Estado: "esperando_aprobacion", PendienteAccion: "rm", PendienteDescripcion: "limpiar"},
	}}
	conDatos := NuevoServidor(nil, 0, ServidorOpciones{Aprobador: aprob})
	rr2 := httptest.NewRecorder()
	conDatos.manejarOrdenes(rr2, req)
	var res struct {
		Ordenes []OrdenPanel `json:"ordenes"`
	}
	if err := json.NewDecoder(rr2.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res.Ordenes) != 1 {
		t.Fatalf("ordenes = %d, esperaba 1", len(res.Ordenes))
	}
	o := res.Ordenes[0]
	if o.ID != 3 || o.Objetivo != "limpiar temp" || o.Estado != "esperando_aprobacion" ||
		o.PendienteAccion != "rm" || o.PendienteDescripcion != "limpiar" {
		t.Errorf("orden del panel incorrecta: %+v", o)
	}
}

func TestManejarDashboard(t *testing.T) {
	aprob := &fakeAprobadorConDatos{ordenes: []core.Orden{
		{ID: 1, Objetivo: "preparar informe", Estado: "pendiente"},
		{ID: 2, Objetivo: "enviar correo", Estado: "esperando_aprobacion", PendienteDescripcion: "enviar correo al cliente"},
		{ID: 3, Objetivo: "auditar servidores", Estado: "bloqueada"},
	}}
	aud := &fakeAuditor{entradas: []audit.Entrada{
		{Momento: "hoy 10:00", Comando: "abrir word", Rol: "dueño"},
		{Momento: "ayer 15:00", Comando: "listar procesos", Rol: "operador"},
	}}
	s := NuevoServidor(nil, 0, ServidorOpciones{
		Aprobador: aprob,
		Auditor:   aud,
		Estado:    &fakeEstado{e: core.EstadoPanel{Hora: "10:00:00"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rr := httptest.NewRecorder()
	s.manejarDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dashboard: status %d", rr.Code)
	}

	var res PanelResumen
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.OrdenesActivas != 3 {
		t.Errorf("ordenes_activas = %d, esperaba 3", res.OrdenesActivas)
	}
	if res.EsperandoAprob != 1 {
		t.Errorf("esperando_aprobacion = %d, esperaba 1", res.EsperandoAprob)
	}
	if res.Bloqueadas != 1 {
		t.Errorf("bloqueadas = %d, esperaba 1", res.Bloqueadas)
	}
	if len(res.Ordenes) != 3 {
		t.Errorf("ordenes = %d, esperaba 3", len(res.Ordenes))
	}
	if len(res.ActividadReciente) != 2 {
		t.Errorf("actividad_reciente = %d, esperaba 2", len(res.ActividadReciente))
	}
	if res.Estado.Hora == "" {
		t.Error("estado del sistema debería venir poblado")
	}
}

func TestAprobarDenegarAdminOK(t *testing.T) {
	aprob := &fakeAprobadorConDatos{}
	s := NuevoServidor(nil, 0, ServidorOpciones{Aprobador: aprob})

	req := httptest.NewRequest(http.MethodPost, "/api/aprobar", bytes.NewBufferString(`{"orden_id":1,"pin":"1234"}`))
	rr := httptest.NewRecorder()
	s.manejarAprobar(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("aprobar sin contraseña (admin) debería ser 200, fue %d", rr.Code)
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["response"] != "aprobada con 1234" {
		t.Errorf("respuesta de aprobación = %q", res["response"])
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/denegar", bytes.NewBufferString(`{"orden_id":1}`))
	rr2 := httptest.NewRecorder()
	s.manejarDenegar(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("denegar debería ser 200, fue %d", rr2.Code)
	}
}

func TestAprobarDenegarSinAprobador(t *testing.T) {
	s := NuevoServidor(nil, 0)
	req := httptest.NewRequest(http.MethodPost, "/api/aprobar", bytes.NewBufferString(`{"orden_id":1,"pin":"1234"}`))
	rr := httptest.NewRecorder()
	s.manejarAprobar(rr, req)
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["error"] != "sin aprobador" {
		t.Errorf("sin aprobador debería responder 'sin aprobador', fue %v", res)
	}
}

func TestAprobarMetodoIncorrecto(t *testing.T) {
	s := nuevoServidorCon("")
	req := httptest.NewRequest(http.MethodGet, "/api/aprobar", nil)
	rr := httptest.NewRecorder()
	s.manejarAprobar(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/aprobar debería ser 405, fue %d", rr.Code)
	}
}

func TestAprobarCuerpoInvalido(t *testing.T) {
	aprob := &fakeAprobadorConDatos{}
	s := NuevoServidor(nil, 0, ServidorOpciones{Aprobador: aprob})
	req := httptest.NewRequest(http.MethodPost, "/api/aprobar", bytes.NewBufferString("{no"))
	rr := httptest.NewRecorder()
	s.manejarAprobar(rr, req)
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["error"] != "request inválido" {
		t.Errorf("cuerpo inválido debería responder 'request inválido', fue %v", res)
	}
}

func TestSesionConContrasenaRequiereLogin(t *testing.T) {
	s := nuevoServidorCon("clave123")
	req := httptest.NewRequest(http.MethodGet, "/api/sesion", nil)
	rr := httptest.NewRecorder()
	s.manejarSesion(rr, req)
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["requiere_login"] != true {
		t.Errorf("con contraseña debería requerir login, fue %v", res["requiere_login"])
	}
}

func TestRolActual(t *testing.T) {
	s := nuevoServidorCon("")
	if r := s.rolActual(httptest.NewRequest(http.MethodGet, "/", nil)); r != security.RolAdmin {
		t.Errorf("sin contraseña rol = %v, esperaba admin", r)
	}

	s2 := nuevoServidorCon("clave")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if r := s2.rolActual(req); r != security.RolOperador {
		t.Errorf("sin cookie rol = %v, esperaba operador", r)
	}

	// sesión válida → admin
	body := bytes.NewBufferString(`{"clave":"clave"}`)
	login := httptest.NewRequest(http.MethodPost, "/api/login", body)
	rr := httptest.NewRecorder()
	s2.manejarLogin(rr, login)
	cookies := rr.Result().Cookies()
	reqConSesion := httptest.NewRequest(http.MethodGet, "/", nil)
	reqConSesion.AddCookie(cookies[0])
	if r := s2.rolActual(reqConSesion); r != security.RolAdmin {
		t.Errorf("con sesión válida rol = %v, esperaba admin", r)
	}

	// cookie de sesión desconocida → operador
	reqFalsa := httptest.NewRequest(http.MethodGet, "/", nil)
	reqFalsa.AddCookie(&http.Cookie{Name: nombreCookieSesion, Value: "no-existe"})
	if r := s2.rolActual(reqFalsa); r != security.RolOperador {
		t.Errorf("cookie desconocida rol = %v, esperaba operador", r)
	}
}

func TestExigePermisoDeniegaOperador(t *testing.T) {
	s := nuevoServidorCon("clave")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	if s.exigePermiso(req, rr, security.PermisoAuditoria) {
		t.Error("operador no debería pasar exigePermiso")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("debería responder 403, fue %d", rr.Code)
	}

	// admin pasa
	body := bytes.NewBufferString(`{"clave":"clave"}`)
	login := httptest.NewRequest(http.MethodPost, "/api/login", body)
	rrLogin := httptest.NewRecorder()
	s.manejarLogin(rrLogin, login)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(rrLogin.Result().Cookies()[0])
	rr2 := httptest.NewRecorder()
	if !s.exigePermiso(req2, rr2, security.PermisoAuditoria) {
		t.Error("admin debería pasar exigePermiso")
	}
}

func TestNormalizarClave(t *testing.T) {
	if normalizarClave("  AbC123  ") != "abc123" {
		t.Errorf("normalizarClave = %q, esperaba abc123", normalizarClave("  AbC123  "))
	}
	tok1 := nuevoTokenSesion()
	tok2 := nuevoTokenSesion()
	if tok1 == tok2 {
		t.Error("dos tokens de sesión no deberían ser iguales")
	}
}

func TestConRecuperacionConviertePanicoEn500(t *testing.T) {
	s := NuevoServidor(nil, 0)
	panico := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	rr := httptest.NewRecorder()
	s.conRecuperacion(panico).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("pánico debería traducirse en 500, fue %d", rr.Code)
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if res["error"] != "error interno del servidor" {
		t.Errorf("cuerpo inesperado: %v", res)
	}
}

func TestCargarHistorialArchivoCorruptoNoRompe(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "historial.json")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(ruta, []byte("{no json"), 0o600)

	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{RutaHistorial: ruta})
	if len(s.historial) != 0 {
		t.Errorf("historial corrupto debería dejarlo vacío, hay %d", len(s.historial))
	}
}

func TestChatAcotaHistorialACien(t *testing.T) {
	brain := &fakeBrain{respuesta: "ok"}
	s := NuevoServidor(brain, 0)
	for i := 0; i < 120; i++ {
		s.manejarChat(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"m"}`)))
	}
	if len(s.historial) != 100 {
		t.Errorf("historial debería acotarse a 100, quedaron %d", len(s.historial))
	}
	if !strings.Contains(s.historial[0].Usuario, "m") {
		t.Errorf("tras acotar, las entradas deberían seguir siendo válidas: %+v", s.historial[0])
	}
}

type fakeEmpresa struct {
	perf core.PerfilEmpresa
	err  error
}

func (f *fakeEmpresa) Obtener() core.PerfilEmpresa                    { return f.perf }
func (f *fakeEmpresa) Reemplazar(p core.PerfilEmpresa) error           { f.perf = p; return f.err }
func (f *fakeEmpresa) Resumen() string                                { return "resumen" }

func TestManejarEmpresa_Get(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa: &fakeEmpresa{perf: core.PerfilEmpresa{Nombre: "ABC"}},
	})
	rr := httptest.NewRecorder()
	s.manejarEmpresa(rr, httptest.NewRequest(http.MethodGet, "/api/empresa", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET debería ser 200, fue %d", rr.Code)
	}
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["resumen"] != "resumen" {
		t.Errorf("resumen inesperado: %v", res)
	}
}

func TestManejareEmpresaPutRequiereAdmin(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa:        &fakeEmpresa{},
		ContrasenaHash: "h",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/empresa", bytes.NewBufferString(`{"nombre":"X"}`))
	rr := httptest.NewRecorder()
	s.manejarEmpresa(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("sin sesión debería ser 403, fue %d", rr.Code)
	}
}

func TestManejareEmpresaPutAdmin(t *testing.T) {
	fe := &fakeEmpresa{}
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{Empresa: fe})
	// Sin contraseña configurada todos son Admin → la escritura pasa.
	req := httptest.NewRequest(http.MethodPost, "/api/empresa", bytes.NewBufferString(`{"nombre":"Nueva SRL"}`))
	rr := httptest.NewRecorder()
	s.manejarEmpresa(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("guardar perfil debería ser 200, fue %d (%s)", rr.Code, rr.Body.String())
	}
	if fe.perf.Nombre != "Nueva SRL" {
		t.Errorf("el perfil no se guardó: %+v", fe.perf)
	}
}

type fakePerfil struct {
	us    []core.PerfilUsuario
	activo string
}

func (f *fakePerfil) Usuarios() []core.PerfilUsuario    { return f.us }
func (f *fakePerfil) Activo() string                    { return f.activo }
func (f *fakePerfil) ActivoRol() string                 { return core.PerfilDueno }
func (f *fakePerfil) Seleccionar(texto string) bool     { f.activo = texto; return true }
func (f *fakePerfil) AgregarUsuario(n, a, r string) bool { f.us = append(f.us, core.PerfilUsuario{Nombre: n, Area: a, Rol: r}); return true }
func (f *fakePerfil) Eliminar(nombre string) bool       { return true }

func TestManejarPerfil_Get(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Perfil: &fakePerfil{activo: core.PerfilDueno},
	})
	rr := httptest.NewRecorder()
	s.manejarPerfil(rr, httptest.NewRequest(http.MethodGet, "/api/perfil", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET debería ser 200, fue %d", rr.Code)
	}
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["activo"] != core.PerfilDueno {
		t.Errorf("activo inesperado: %v", res)
	}
}

func TestManejarPerfil_PostRequiereAdmin(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Perfil:         &fakePerfil{},
		ContrasenaHash: "h",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/perfil", bytes.NewBufferString(`{"seleccionar":"admin"}`))
	rr := httptest.NewRecorder()
	s.manejarPerfil(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("sin sesión debería ser 403, fue %d", rr.Code)
	}
}

func TestManejarPerfil_Seleccionar(t *testing.T) {
	fp := &fakePerfil{}
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{Perfil: fp})
	req := httptest.NewRequest(http.MethodPost, "/api/perfil", bytes.NewBufferString(`{"seleccionar":"admin"}`))
	rr := httptest.NewRecorder()
	s.manejarPerfil(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("seleccionar debería ser 200, fue %d (%s)", rr.Code, rr.Body.String())
	}
	if fp.activo != "admin" {
		t.Errorf("perfil activo = %q, esperaba admin", fp.activo)
	}
}
