package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerarYValidarLicencia(t *testing.T) {
	clave, err := GenerarLicencia(PlanPro, 5)
	if err != nil {
		t.Fatalf("GenerarLicencia: %v", err)
	}
	if !strings.HasPrefix(clave, "JARVIS-PRO-005-") {
		t.Errorf("formato inesperado: %q", clave)
	}
	k, ok := LicenciaValida(clave)
	if !ok {
		t.Fatal("la clave generada debe validar")
	}
	if k.Plan != PlanPro || k.Puestos != 5 {
		t.Errorf("plan/puestos = %s/%d, esperaba pro/5", k.Plan, k.Puestos)
	}
}

func TestGenerarLicenciaPlanInvalido(t *testing.T) {
	if _, err := GenerarLicencia("gratis", 5); err == nil {
		t.Fatal("plan inválido debe dar error")
	}
	if _, err := GenerarLicencia(PlanLite, 0); err == nil {
		t.Fatal("puestos 0 debe dar error")
	}
}

func TestLicenciaValidaTamper(t *testing.T) {
	clave, err := GenerarLicencia(PlanEmpresa, 50)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := LicenciaValida(clave + "X"); ok {
		t.Fatal("clave con carácter extra no debe validar")
	}
	cortada := clave[:len(clave)-1]
	if _, ok := LicenciaValida(cortada); ok {
		t.Fatal("clave cortada no debe validar")
	}
	// Cambiar un dígito de puestos invalida la firma.
	conPuestosDistintos := strings.Replace(clave, "-050-", "-010-", 1)
	if _, ok := LicenciaValida(conPuestosDistintos); ok {
		t.Fatal("clave con puestos alterados no debe validar")
	}
	if _, ok := LicenciaValida("JARVIS-XXXX-005-abcdefabcdef-abcdefabcdef"); ok {
		t.Fatal("clave de formato raro no debe validar")
	}
}

func TestPuestosLicencia(t *testing.T) {
	if PuestosLicencia("") != 0 {
		t.Fatal("sin licencia = 0 puestos")
	}
	clave, _ := GenerarLicencia(PlanLite, 1)
	if PuestosLicencia(clave) != 1 {
		t.Fatal("licencia lite = 1 puesto")
	}
	if PuestosLicencia("basura") != 0 {
		t.Fatal("clave inválida = 0 puestos")
	}
}

func TestEstadoLicencia(t *testing.T) {
	if !strings.Contains(EstadoLicencia(""), "Sin licencia") {
		t.Fatal("sin licencia debe avisar modo piloto")
	}
	clave, _ := GenerarLicencia(PlanEmpresa, 50)
	if !strings.Contains(EstadoLicencia(clave), "EMPRESA") || !strings.Contains(EstadoLicencia(clave), "50") {
		t.Fatalf("estado inesperado: %q", EstadoLicencia(clave))
	}
	if !strings.Contains(EstadoLicencia("clave-falsa"), "no es válida") {
		t.Fatal("clave inválida debe avisar")
	}
}

func TestExtraerLicenciaDelComando(t *testing.T) {
	clave, _ := GenerarLicencia(PlanPro, 5)
	got := extraerLicenciaDelComando("activá la licencia " + clave)
	if got != clave {
		t.Errorf("got %q, esperaba %q", got, clave)
	}
	if extraerLicenciaDelComando("qué licencia tengo") != "" {
		t.Fatal("no debe extraer nada sin clave")
	}
}

func TestActivarLicenciaConSetter(t *testing.T) {
	h := NewHands()
	guardada := ""
	h.LicenseSetter = func(clave string) bool { guardada = clave; return true }

	clave, _ := GenerarLicencia(PlanPro, 5)
	got := h.activarLicencia("activá la licencia " + clave)
	if !strings.Contains(got, "PRO") || h.LicenseKey != clave || guardada != clave {
		t.Fatalf("activación falló: %q (key=%q saved=%q)", got, h.LicenseKey, guardada)
	}

	// Clave inválida: no persiste.
	h.LicenseKey = ""
	got = h.activarLicencia("activá la licencia JARVIS-PRO-005-aaaaaaaaaaaa-bbbbbbbbbbbbbbbb")
	if !strings.Contains(got, "no es válida") || h.LicenseKey != "" || guardada != clave {
		t.Fatalf("clave inválida no debe persistir: %q (key=%q saved=%q)", got, h.LicenseKey, guardada)
	}

	// Sin clave en el comando.
	got = h.activarLicencia("activá mi plan")
	if !strings.Contains(got, "No le entendí") {
		t.Fatalf("respuesta inesperada: %q", got)
	}

	// Setter que falla: no persiste en memoria.
	h.LicenseSetter = func(string) bool { return false }
	got = h.activarLicencia("activá la licencia " + clave)
	if !strings.Contains(got, "No pude guardar") || h.LicenseKey != "" {
		t.Fatalf("setter que falla no debe persistir: %q (key=%q)", got, h.LicenseKey)
	}
}

func TestConsultarLicencia(t *testing.T) {
	h := NewHands()
	if !strings.Contains(h.consultarLicencia(), "Sin licencia") {
		t.Fatal("sin licencia debe reportar piloto")
	}
	clave, _ := GenerarLicencia(PlanLite, 1)
	h.LicenseKey = clave
	if !strings.Contains(h.consultarLicencia(), "LITE") {
		t.Fatalf("respuesta inesperada: %q", h.consultarLicencia())
	}
}

func TestActivarLicenciaActualizaLimitePerfil(t *testing.T) {
	g := NuevoGestorPerfil(filepath.Join(t.TempDir(), "perfil.json"))
	h := NewHands(HandsOpciones{Perfil: g})
	h.LicenseSetter = func(string) bool { return true }

	clave, _ := GenerarLicencia(PlanLite, 1)
	h.activarLicencia("activá la licencia " + clave)
	if g.LimitePuestos != 1 {
		t.Fatalf("LimitePuestos tras activar lite = %d, esperaba 1", g.LimitePuestos)
	}

	claveEmpresa, _ := GenerarLicencia(PlanEmpresa, 50)
	h.activarLicencia("activá la licencia " + claveEmpresa)
	if g.LimitePuestos != 50 {
		t.Fatalf("LimitePuestos tras activar empresa = %d, esperaba 50", g.LimitePuestos)
	}

	if !g.AgregarUsuario("Ana", "", "admin") {
		t.Fatal("no se pudo agregar a Ana")
	}
	if g.LimiteAlcanzado() {
		t.Fatal("con 1/50 aún no debe alcanzar; hay que llenar 50")
	}
	// Con lite (1 puesto), el segundo usuario debe rechazarse.
	h.activarLicencia("activá la licencia " + clave)
	if g.AgregarUsuario("Pedro", "", "empleado") {
		t.Fatal("con lite (1 puesto) y Ana registrada, Pedro debe rechazarse")
	}
}
