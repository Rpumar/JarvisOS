package core

import (
	"os"
	"testing"
)

// TestOutlookSmokeVivo valida que el calendario de Outlook sea legible por
// COM en esta máquina. Se corre solo si JARVIS_SMOKE=1. Es de solo lectura:
// no crea citas, para no ensuciar el calendario real del dueño.
func TestOutlookSmokeVivo(t *testing.T) {
	if os.Getenv("JARVIS_SMOKE") == "" {
		t.Skip("skip: requiere JARVIS_SMOKE=1 y Outlook instalado")
	}
	h := &Hands{}
	msg := h.leerOutlook()
	if len(msg) == 0 {
		t.Fatal("leerOutlook devolvió vacío")
	}
	t.Logf("Outlook OK: %s", msg)
}