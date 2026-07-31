package core

import (
	"strings"
	"testing"
)

func TestSesionesActivas(t *testing.T) {
	h := NewHands()
	resp := h.sesionesActivas()
	t.Logf("Respuesta: %q", resp)
	if strings.Contains(resp, "No pude") {
		t.Fatalf("falló: %s", resp)
	}
}
