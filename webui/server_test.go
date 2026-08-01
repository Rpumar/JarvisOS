package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"JarvisOS/core"
	"JarvisOS/core/audit"
)

type fakeAprobador struct {
	aprobadas int
	denegadas int
}

func (f *fakeAprobador) AprobarOrden(id int, pin string) string { f.aprobadas++; return "aprobada" }
func (f *fakeAprobador) DenegarOrden(id int) string             { f.denegadas++; return "denegada" }
func (f *fakeAprobador) OrdenesParaPanel() []core.Orden         { return nil }

type fakeAuditor struct {
	entradas []audit.Entrada
}

func (f *fakeAuditor) AuditoriaPanel() []audit.Entrada { return f.entradas }

func nuevoServidorCon(clave string) *ServidorWeb {
	hash := ""
	if clave != "" {
		hash = core.HashTexto(clave)
	}
	return NuevoServidor(nil, 0,
		ServidorOpciones{ContrasenaHash: hash, Aprobador: &fakeAprobador{}, Auditor: &fakeAuditor{}},
	)
}

func TestSinContrasenaElAccesoEsAdmin(t *testing.T) {
	s := nuevoServidorCon("")
	req := httptest.NewRequest(http.MethodGet, "/api/sesion", nil)
	rr := httptest.NewRecorder()
	s.manejarSesion(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sesion: status %d", rr.Code)
	}
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["rol"] != "admin" {
		t.Fatalf("sin contraseña el rol debería ser admin, fue %v", res["rol"])
	}
}

func TestConContrasenaElAnonimoEsOperador(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	req := httptest.NewRequest(http.MethodGet, "/api/sesion", nil)
	rr := httptest.NewRecorder()
	s.manejarSesion(rr, req)
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["rol"] != "operador" {
		t.Fatalf("el visitante debería ser operador, fue %v", res["rol"])
	}
	if res["autenticado"] != false {
		t.Fatalf("el visitante no debería estar autenticado, fue %v", res["autenticado"])
	}
}

func TestOperadorNoVeAuditoria(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	req := httptest.NewRequest(http.MethodGet, "/api/auditoria", nil)
	rr := httptest.NewRecorder()
	s.manejarAuditoria(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operador debería recibir 403, recibió %d", rr.Code)
	}
}

func TestOperadorNoAprueba(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(map[string]interface{}{"orden_id": 1, "pin": "1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/aprobar", &body)
	rr := httptest.NewRecorder()
	s.manejarAprobar(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operador debería recibir 403 al aprobar, recibió %d", rr.Code)
	}
}

func TestLoginCorrectoDaAdmin(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(map[string]string{"clave": "ABCD1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", &body)
	rr := httptest.NewRecorder()
	s.manejarLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login con clave válida debería ser 200, fue %d", rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login debería emitir cookie de sesión")
	}

	ses := httptest.NewRequest(http.MethodGet, "/api/sesion", nil)
	ses.AddCookie(cookies[0])
	rr2 := httptest.NewRecorder()
	s.manejarSesion(rr2, ses)
	var res map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&res)
	if res["rol"] != "admin" {
		t.Fatalf("con sesión el rol debería ser admin, fue %v", res["rol"])
	}

	aud := httptest.NewRequest(http.MethodGet, "/api/auditoria", nil)
	aud.AddCookie(cookies[0])
	rr3 := httptest.NewRecorder()
	s.manejarAuditoria(rr3, aud)
	if rr3.Code != http.StatusOK {
		t.Fatalf("admin debería ver auditoría, recibió %d", rr3.Code)
	}
}

func TestLoginIncorrectoRechazado(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(map[string]string{"clave": "incorrecta"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", &body)
	rr := httptest.NewRecorder()
	s.manejarLogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("login con clave incorrecta debería ser 401, fue %d", rr.Code)
	}
}

func TestLogoutRevocaLaSesion(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(map[string]string{"clave": "abcd1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", &body)
	rr := httptest.NewRecorder()
	s.manejarLogin(rr, req)
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login debería emitir cookie")
	}

	lo := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	lo.AddCookie(cookies[0])
	rr2 := httptest.NewRecorder()
	s.manejarLogout(rr2, lo)

	ses := httptest.NewRequest(http.MethodGet, "/api/sesion", nil)
	ses.AddCookie(cookies[0])
	rr3 := httptest.NewRecorder()
	s.manejarSesion(rr3, ses)
	var res map[string]interface{}
	json.NewDecoder(rr3.Body).Decode(&res)
	if res["rol"] != "operador" {
		t.Fatalf("tras logout el rol debería volver a operador, fue %v", res["rol"])
	}
}
