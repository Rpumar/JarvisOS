package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"JarvisOS/core"
)

// fakeEmpresaHook captura el perfil guardado por el onboarding.
type fakeEmpresaHook struct {
	guardado core.PerfilEmpresa
}

func (f *fakeEmpresaHook) Obtener() core.PerfilEmpresa { return f.guardado }
func (f *fakeEmpresaHook) Resumen() string             { return "resumen" }
func (f *fakeEmpresaHook) Reemplazar(p core.PerfilEmpresa) error {
	f.guardado = p
	return nil
}

func TestEstadoOnboarding_PrimerArranque(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa: &fakeEmpresaHook{},
		Perfil:  &fakePerfil{},
	})
	st := s.estadoOnboarding()
	if st["primer_arranque"] != true {
		t.Fatalf("esperaba primer_arranque true, got %v", st["primer_arranque"])
	}
	if st["completo"] != false {
		t.Fatalf("esperaba completo false, got %v", st["completo"])
	}
}

func TestOnboarding_Get(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa: &fakeEmpresaHook{},
		Perfil:  &fakePerfil{},
	})
	rr := httptest.NewRecorder()
	s.manejarOnboarding(rr, httptest.NewRequest(http.MethodGet, "/api/onboarding", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET debería ser 200, fue %d", rr.Code)
	}
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["primer_arranque"] != true {
		t.Errorf("primer_arranque inesperado: %v", res["primer_arranque"])
	}
}

func TestOnboarding_PostCompleto(t *testing.T) {
	fe := &fakeEmpresaHook{}
	fp := &fakePerfil{}
	var pinSet, claveSet string
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa:          fe,
		Perfil:           fp,
		PINSetter:        func(h string) bool { pinSet = h; return true },
		ContrasenaSetter: func(h string) bool { claveSet = h; return true },
	})
	body := `{"empresa":"ACME SRL","rubro":"logística","dueno":"Ana","pin":"1234","contrasena":"secreto123"}`
	rr := httptest.NewRecorder()
	s.manejarOnboarding(rr, httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewBufferString(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("POST debería ser 200, fue %d (%s)", rr.Code, rr.Body.String())
	}
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["ok"] != true {
		t.Fatalf("ok inesperado: %v", res)
	}
	if pinSet == "" || claveSet == "" {
		t.Fatalf("no se fijaron hashes pin=%q clave=%q", pinSet, claveSet)
	}
	if fe.guardado.Nombre != "ACME SRL" {
		t.Errorf("perfil de empresa no guardado: %+v", fe.guardado)
	}
	st := s.estadoOnboarding()
	if st["completo"] != true {
		t.Fatalf("onboarding no completo: %v", st)
	}
}

func TestOnboarding_PostPINInvalido(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa:          &fakeEmpresaHook{},
		Perfil:           &fakePerfil{},
		PINSetter:        func(string) bool { return true },
		ContrasenaSetter: func(string) bool { return true },
	})
	body := `{"empresa":"X","pin":"1a"}`
	rr := httptest.NewRecorder()
	s.manejarOnboarding(rr, httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewBufferString(body)))
	var res map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&res)
	if res["ok"] == true {
		t.Fatalf("con PIN inválido no debería ser ok: %v", res)
	}
	errs, _ := res["errores"].([]interface{})
	found := false
	for _, e := range errs {
		if str, isStr := e.(string); isStr && strings.Contains(str, "PIN") {
			found = true
		}
	}
	if !found {
		t.Fatalf("debería reportar error de PIN: %v", res)
	}
}

func TestOnboardingHashPIN(t *testing.T) {
	s := &ServidorWeb{}
	hash, ok := s.hashPIN("1234")
	if !ok || len(hash) == 0 {
		t.Fatalf("hashPIN 1234 devolvió ok=%v", ok)
	}
	if _, ok := s.hashPIN("12"); ok {
		t.Fatal("PIN corto no debía validar")
	}
	if _, ok := s.hashPIN("12a4"); ok {
		t.Fatal("PIN no numérico no debía validar")
	}
	if core.HashTexto("1234") != hash {
		t.Fatal("el hash del PIN debe coincidir con HashTexto")
	}
}

func TestOnboarding_PostPidePermisoYaConfigurado(t *testing.T) {
	s := NuevoServidor(&fakeBrain{}, 0, ServidorOpciones{
		Empresa:          &fakeEmpresaHook{},
		Perfil:           &fakePerfil{},
		ContrasenaHash:   "abc", // ya hay contraseña → el panel exige sesión
		ContrasenaSetter: func(string) bool { return true },
	})
	body := `{"empresa":"Y"}`
	req := httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	s.manejarOnboarding(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("sin sesión y ya configurado debería ser 403, fue %d", rr.Code)
	}
}