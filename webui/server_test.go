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

type fakeGestorSkills struct {
	skills []core.Skill
}

func (f *fakeGestorSkills) ListarDetallado() []core.SkillInfo {
	var res []core.SkillInfo
	for _, s := range f.skills {
		res = append(res, core.SkillInfo{Nombre: s.Nombre, Descripcion: s.Descripcion, Activar: s.Activar, Prioridad: s.Prioridad})
	}
	return res
}
func (f *fakeGestorSkills) Obtener(nombre string) (core.Skill, bool) {
	for _, s := range f.skills {
		if s.Nombre == nombre {
			return s, true
		}
	}
	return core.Skill{}, false
}
func (f *fakeGestorSkills) CrearOActualizar(s core.Skill) error {
	for i := range f.skills {
		if f.skills[i].Nombre == s.Nombre {
			f.skills[i] = s
			return nil
		}
	}
	f.skills = append(f.skills, s)
	return nil
}
func (f *fakeGestorSkills) Eliminar(nombre string) error {
	for i, s := range f.skills {
		if s.Nombre == nombre {
			f.skills = append(f.skills[:i], f.skills[i+1:]...)
			return nil
		}
	}
	return nil
}

type fakeGestorRoles struct {
	roles []core.Rol
}

func (f *fakeGestorRoles) ListarDetallado() []core.RolInfo {
	var res []core.RolInfo
	for _, r := range f.roles {
		res = append(res, core.RolInfo{Nombre: r.Nombre, Etiqueta: r.Etiqueta, Descripcion: r.Descripcion, Activar: r.Activar, Contexto: r.Contexto})
	}
	return res
}
func (f *fakeGestorRoles) Obtener(nombre string) (core.Rol, bool) {
	for _, r := range f.roles {
		if r.Nombre == nombre {
			return r, true
		}
	}
	return core.Rol{}, false
}
func (f *fakeGestorRoles) CrearOActualizar(r core.Rol) error {
	for i := range f.roles {
		if f.roles[i].Nombre == r.Nombre {
			f.roles[i] = r
			return nil
		}
	}
	f.roles = append(f.roles, r)
	return nil
}
func (f *fakeGestorRoles) Eliminar(nombre string) error {
	for i, r := range f.roles {
		if r.Nombre == nombre {
			f.roles = append(f.roles[:i], f.roles[i+1:]...)
			return nil
		}
	}
	return nil
}

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

func TestSkillsAdminCreaYLista(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	sk := &fakeGestorSkills{}
	s.skills = sk

	// Sin sesión, crear una skill devuelve 403.
	body := bytes.NewBufferString(`{"nombre":"mi-skill","activar":["x"],"instrucciones":"y"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/skills", body)
	rr := httptest.NewRecorder()
	s.manejarSkills(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operador debería recibir 403 al crear skill, recibió %d", rr.Code)
	}
}

func TestSkillsCRUDFlujo(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	sk := &fakeGestorSkills{}
	s.skills = sk

	// Login para tener sesión de admin.
	var lb bytes.Buffer
	json.NewEncoder(&lb).Encode(map[string]string{"clave": "abcd1234"})
	lr := httptest.NewRecorder()
	s.manejarLogin(lr, httptest.NewRequest(http.MethodPost, "/api/login", &lb))
	cookie := lr.Result().Cookies()[0]

	// Crear.
	body := bytes.NewBufferString(`{"nombre":"mi-skill","activar":["x","y"],"instrucciones":"z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/skills", body)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	s.manejarSkills(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin debería poder crear skill, recibió %d", rr.Code)
	}

	// Listar.
	lr2 := httptest.NewRecorder()
	s.manejarSkills(lr2, httptest.NewRequest(http.MethodGet, "/api/skills", nil))
	var lista map[string][]core.SkillInfo
	json.NewDecoder(lr2.Body).Decode(&lista)
	if len(lista["skills"]) != 1 || lista["skills"][0].Nombre != "mi-skill" {
		t.Fatalf("la skill creada no aparece: %+v", lista)
	}

	// Obtener detalle (con instrucciones).
	or := httptest.NewRecorder()
	s.manejarSkills(or, httptest.NewRequest(http.MethodGet, "/api/skills?nombre=mi-skill", nil))
	var detalle core.Skill
	json.NewDecoder(or.Body).Decode(&detalle)
	if detalle.Instrucciones != "z" || len(detalle.Activar) != 2 {
		t.Fatalf("detalle incorrecto: %+v", detalle)
	}

	// Borrar.
	dr := httptest.NewRecorder()
	del := httptest.NewRequest(http.MethodDelete, "/api/skills?nombre=mi-skill", nil)
	del.AddCookie(cookie)
	s.manejarSkills(dr, del)
	if dr.Code != http.StatusOK {
		t.Fatalf("admin debería poder borrar skill, recibió %d", dr.Code)
	}
	if len(sk.skills) != 0 {
		t.Fatal("la skill no se borró")
	}
}

func TestRolesAdminCreaYLista(t *testing.T) {
	s := nuevoServidorCon("abcd1234")
	rs := &fakeGestorRoles{}
	s.roles = rs

	// Operador no puede crear.
	body := bytes.NewBufferString(`{"nombre":"analista","etiqueta":"Analista","prompt":"p"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/roles", body)
	rr := httptest.NewRecorder()
	s.manejarRoles(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operador debería recibir 403 al crear rol, recibió %d", rr.Code)
	}

	// Admin crea y lista.
	var lb bytes.Buffer
	json.NewEncoder(&lb).Encode(map[string]string{"clave": "abcd1234"})
	lr := httptest.NewRecorder()
	s.manejarLogin(lr, httptest.NewRequest(http.MethodPost, "/api/login", &lb))
	cookie := lr.Result().Cookies()[0]

	body2 := bytes.NewBufferString(`{"nombre":"analista","etiqueta":"Analista de datos","activar":["datos"],"prompt":"p"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/roles", body2)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	s.manejarRoles(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("admin debería poder crear rol, recibió %d", rr2.Code)
	}

	lr3 := httptest.NewRecorder()
	s.manejarRoles(lr3, httptest.NewRequest(http.MethodGet, "/api/roles", nil))
	var lista map[string][]core.RolInfo
	json.NewDecoder(lr3.Body).Decode(&lista)
	if len(lista["roles"]) != 1 || lista["roles"][0].Nombre != "analista" {
		t.Fatalf("el rol creado no aparece: %+v", lista)
	}
}
