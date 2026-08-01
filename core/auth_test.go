package core

import (
	"path/filepath"
	"testing"

	"JarvisOS/core/audit"
)

func testHandsConAuditoria(t *testing.T) (*Hands, func()) {
	t.Helper()
	dir := t.TempDir()
	h := &Hands{Auditoria: audit.NuevoRegistro(filepath.Join(dir, "auditoria.jsonl"))}
	return h, func() {}
}

func TestEstablecerContrasenaValidaHash(t *testing.T) {
	h, _ := testHandsConAuditoria(t)
	msg := h.EstablecerContrasena("MiClave")
	if len(msg) < 10 {
		t.Fatalf("mensaje inesperado: %q", msg)
	}
	if h.ContrasenaHash == "" {
		t.Fatal("debería haberse guardado el hash")
	}
	if !h.contrasenaValida("MICLAVE") {
		t.Error("la clave normalizada a minúsculas debería ser válida")
	}
	if h.contrasenaValida("otra") {
		t.Error("una clave distinta no debería ser válida")
	}
}

func TestEstablecerContrasenaCortaRechazada(t *testing.T) {
	h, _ := testHandsConAuditoria(t)
	msg := h.EstablecerContrasena("abc")
	if h.ContrasenaHash != "" {
		t.Fatal("una clave muy corta no debería configurarse")
	}
	if len(msg) < 5 {
		t.Fatalf("debería rechazarse con un mensaje, fue %q", msg)
	}
}

func TestSinContrasenaElAccesoEsValido(t *testing.T) {
	h, _ := testHandsConAuditoria(t)
	if !h.contrasenaValida("cualquiera") {
		t.Error("sin contraseña configurada cualquier clave es válida")
	}
}

func TestContrasenaSetterPersiste(t *testing.T) {
	h, _ := testHandsConAuditoria(t)
	persistido := ""
	h.ContrasenaSetter = func(hash string) bool {
		persistido = hash
		return true
	}
	h.EstablecerContrasena("clave123")
	if persistido == "" {
		t.Error("el setter debería haber recibido el hash para persistirlo")
	}
}

func TestExtraerContrasena(t *testing.T) {
	casos := map[string]string{
		"configurá la contraseña de acceso miClave":  "miClave",
		"configurá la contraseña 123456":             "123456",
		"cambiá el password AbCdEf":                  "AbCdEf",
		"configurá la contraseña de acceso":          "",
		"hola cómo estás":                            "",
	}
	for entrada, esperada := range casos {
		if got := extraerContrasena(entrada); got != esperada {
			t.Errorf("extraerContrasena(%q) = %q, esperaba %q", entrada, got, esperada)
		}
	}
}

func TestAuditoriaPanelRecientes(t *testing.T) {
	h, _ := testHandsConAuditoria(t)
	for i := 0; i < 5; i++ {
		h.Auditoria.Registrar(audit.Entrada{Comando: "comando"})
	}
	entradas := h.AuditoriaPanel()
	if len(entradas) != 5 {
		t.Fatalf("se esperaban 5 entradas, hay %d", len(entradas))
	}
}
