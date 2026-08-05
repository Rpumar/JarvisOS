package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCarpetaJarvisOS(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\Test`)
	if got := carpetaJarvisOS(); got != `C:\Users\Test\JarvisOS-datos` {
		t.Errorf("carpetaJarvisOS() = %q", got)
	}
}

func TestCarpetasUsuario(t *testing.T) {
	home, _ := os.UserHomeDir()
	carpetas := carpetasUsuario()
	if len(carpetas) != 6 {
		t.Errorf("esperaba 6 carpetas, obtuve %d", len(carpetas))
	}
	esperado := []string{"Desktop", "Escritorio", "Documents", "Documentos", "Downloads", "Descargas"}
	for i, e := range esperado {
		if carpetas[i] != filepath.Join(home, e) {
			t.Errorf("carpetasUsuario[%d] = %q, esperaba %q", i, carpetas[i], filepath.Join(home, e))
		}
	}
}

func TestBuscarEnUsuario(t *testing.T) {
	// Estructura: <tmp>/Desktop/informe-final.txt y .oculta/ ignorada.
	root := t.TempDir()
	desktop := filepath.Join(root, "Desktop")
	os.MkdirAll(desktop, 0o700)
	os.WriteFile(filepath.Join(desktop, "informe-final.txt"), []byte("hola"), 0o600)
	oculta := filepath.Join(desktop, ".oculta")
	os.MkdirAll(oculta, 0o700)
	os.WriteFile(filepath.Join(oculta, "informe-secreto.txt"), []byte("x"), 0o600)

	t.Setenv("USERPROFILE", root)
	encontrados := buscarEnUsuario("informe-final", 10)
	if len(encontrados) != 1 {
		t.Fatalf("esperaba 1 archivo, obtuvo %d: %v", len(encontrados), encontrados)
	}
	if encontrados[0] != filepath.Join(desktop, "informe-final.txt") {
		t.Errorf("ruta inesperada: %q", encontrados[0])
	}
}

func TestBuscarEnUsuarioLimite(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "Desktop")
	os.MkdirAll(desktop, 0o700)
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(desktop, fmt.Sprintf("dato%d", i)), []byte("x"), 0o600)
	}
	t.Setenv("USERPROFILE", root)
	encontrados := buscarEnUsuario("dato", 3)
	if len(encontrados) != 3 {
		t.Errorf("el límite debería cortar en 3, obtuvo %d", len(encontrados))
	}
}

func TestBuscarEnUsuarioSinCoincidencia(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "Desktop")
	os.MkdirAll(desktop, 0o700)
	os.WriteFile(filepath.Join(desktop, "reporte.txt"), []byte("x"), 0o600)
	t.Setenv("USERPROFILE", root)
	encontrados := buscarEnUsuario("zzzz-inexistente", 10)
	if len(encontrados) != 0 {
		t.Errorf("no debería encontrar nada, obtuvo %v", encontrados)
	}
}

func TestEncontrarArchivoYUbicacion(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "Desktop")
	os.MkdirAll(desktop, 0o700)
	os.WriteFile(filepath.Join(desktop, "mi-documento.txt"), []byte("x"), 0o600)
	t.Setenv("USERPROFILE", root)

	h := &Hands{}
	if got := h.encontrarArchivo("mi-documento"); got != filepath.Join(desktop, "mi-documento.txt") {
		t.Errorf("encontrarArchivo = %q", got)
	}
	if got := h.encontrarArchivo("nada-que-ver"); got != "" {
		t.Errorf("encontrarArchivo inexistente = %q", got)
	}
	if got := h.abrirUbicacion(""); got != "¿Qué archivo quiere ubicar, señor?" {
		t.Errorf("abrirUbicacion vacío = %q", got)
	}
}

func TestBorrarArchivo(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "Desktop")
	os.MkdirAll(desktop, 0o700)
	os.WriteFile(filepath.Join(desktop, "para-borrar.txt"), []byte("x"), 0o600)
	t.Setenv("USERPROFILE", root)

	h := &Hands{}
	if got := h.borrarArchivo(""); got != "¿Qué archivo quiere borrar, señor?" {
		t.Errorf("borrarArchivo vacío = %q", got)
	}
	if got := h.borrarArchivo("archivo-que-no-existe"); got != "No encontré 'archivo-que-no-existe' en sus carpetas de usuario, señor." {
		t.Errorf("borrarArchivo inexistente = %q", got)
	}
}

func TestBuscarArchivoVacio(t *testing.T) {
	h := &Hands{}
	if got := h.buscarArchivo("   "); got != "¿Qué archivo quiere que busque, señor?" {
		t.Errorf("buscarArchivo vacío = %q", got)
	}
}

func TestTomarNotaYLeerNotas(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	h := &Hands{}

	resp := h.tomarNota("comprar pan")
	if resp != "Anotado, señor: comprar pan" {
		t.Errorf("tomarNota = %q", resp)
	}
	if resp := h.tomarNota("   "); resp != "¿Qué quiere que anote, señor?" {
		t.Errorf("tomarNota vacío = %q", resp)
	}

	ruta := filepath.Join(dir, "JarvisOS-datos", "notas.txt")
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("no se creó notas.txt: %v", err)
	}
	out := h.leerNotas()
	if out != "Tiene 1 notas guardadas, señor. Vea las últimas en la consola." {
		t.Errorf("leerNotas = %q", out)
	}

	// Sin notas: nuevo perfil
	dir2 := t.TempDir()
	t.Setenv("USERPROFILE", dir2)
	h2 := &Hands{}
	if out := h2.leerNotas(); out != "No tiene notas guardadas todavía, señor." {
		t.Errorf("leerNotas sin notas = %q", out)
	}
}
