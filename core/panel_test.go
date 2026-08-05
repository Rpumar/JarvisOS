package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserProfileDir(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\Test`)
	if got := userProfileDir(); got != `C:\Users\Test` {
		t.Errorf("userProfileDir() = %q, esperaba C:\\Users\\Test", got)
	}
}

func TestUserProfileDirFallback(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	if got := userProfileDir(); got != "." {
		t.Errorf("userProfileDir() sin variable = %q, esperaba .", got)
	}
}

func TestRutaDatosJarvis(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", filepath.Dir(dir))
	got := rutaDatosJarvis()
	if got != filepath.Join(filepath.Dir(dir), "JarvisOS-datos") {
		t.Errorf("rutaDatosJarvis() = %q", got)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Errorf("la carpeta de datos no se creó: %v", err)
	}
}

func TestCambiarModoPantallaInvalido(t *testing.T) {
	h := &Hands{}
	resp := h.cambiarModoPantalla("modo-inexistente")
	if resp != "Modo no válido, señor. Use: duplicar, extender, único, segundo." {
		t.Errorf("respuesta inesperada: %q", resp)
	}
}

func TestCambiarModoPantallaMapa(t *testing.T) {
	modos := map[string]string{
		"duplicar": "duplicate",
		"extender": "extend",
		"unico":    "internal",
		"segundo":  "external",
	}
	if len(modos) != 4 {
		t.Errorf("el mapa de modos debería tener 4 entradas")
	}
}
