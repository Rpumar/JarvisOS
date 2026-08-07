package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealizarBackup_CopiaYRotacion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "memoria.json"), []byte(`{"x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auditoria.jsonl"), []byte("linea\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "informes"), 0o700); err != nil {
		t.Fatal(err)
	}

	ruta, err := RealizarBackup(dir, 2)
	if err != nil {
		t.Fatalf("RealizarBackup: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(ruta), "backup-") {
		t.Errorf("nombre de backup inesperado: %q", filepath.Base(ruta))
	}
	for _, f := range []string{"memoria.json", "auditoria.jsonl", filepath.Join("informes")} {
		if _, err := os.Stat(filepath.Join(ruta, f)); err != nil {
			t.Errorf("backup sin %s: %v", f, err)
		}
	}

	// Hacer más backups que el máximo: la rotación debe dejar solo 2.
	for i := 0; i < 3; i++ {
		if _, err := RealizarBackup(dir, 2); err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
	}
	if got := ListarBackups(dir); len(got) != 2 {
		t.Fatalf("rotación: esperaba 2 backups, tengo %d (%v)", len(got), got)
	}
}

func TestRealizarBackup_NoRecursaSobreSiMisma(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RealizarBackup(dir, 5); err != nil {
		t.Fatal(err)
	}
	// Correr de nuevo: el backup anterior no debe incluirse a sí mismo.
	if _, err := RealizarBackup(dir, 5); err != nil {
		t.Fatalf("segundo backup: %v", err)
	}
	destino := filepath.Join(dir, backupsDir)
	entradas, err := os.ReadDir(destino)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if e.IsDir() {
			anidado := filepath.Join(destino, e.Name(), backupsDir)
			if _, err := os.Stat(anidado); err == nil {
				t.Fatalf("el backup contiene una copia anidada de backups: %s", anidado)
			}
		}
	}
}

func TestRealizarBackup_CarpetaVacia(t *testing.T) {
	dir := t.TempDir()
	if _, err := RealizarBackup(dir, 5); err != nil {
		t.Fatalf("backup sobre carpeta casi vacía no debe fallar: %v", err)
	}
	if got := ListarBackups(dir); len(got) != 1 {
		t.Fatalf("esperaba 1 backup, tengo %d", len(got))
	}
}

func TestRealizarBackup_CarpetaVaciaArgumento(t *testing.T) {
	if _, err := RealizarBackup("  ", 5); err == nil {
		t.Fatal("carpeta de datos vacía debe devolver error")
	}
}

func TestListarBackups_SinCarpeta(t *testing.T) {
	if got := ListarBackups(t.TempDir()); len(got) != 0 {
		t.Fatalf("sin backups debe devolver vacío, tengo %v", got)
	}
}
