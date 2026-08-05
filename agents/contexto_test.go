package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func escribir(t *testing.T, dir, nombre, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDetectarProyectoGo(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "go.mod", "module ejemplo/prueba\n\ngo 1.22\n")

	info := DetectarProyecto(dir)
	if info.Lenguaje != "Go" {
		t.Errorf("Lenguaje = %q, esperaba Go", info.Lenguaje)
	}
	if info.BuildCmd != "go build ./..." || info.TestCmd != "go test ./..." {
		t.Errorf("comandos Go incorrectos: build=%q test=%q", info.BuildCmd, info.TestCmd)
	}
	if info.Framework != "ejemplo/prueba" {
		t.Errorf("Framework = %q, esperaba el module name", info.Framework)
	}
}

func TestDetectarProyectoNode(t *testing.T) {
	casos := []struct {
		nombre     string
		contenido  string
		lenguaje   string
		framework  string
	}{
		{"react", `{"dependencies":{"react":"^18"}}`, "Node.js", "React"},
		{"next", `{"scripts":{"dev":"next dev"}}`, "Node.js", "Next.js"},
		{"express", `{"dependencies":{"express":"^4"}}`, "Node.js", "Express"},
		{"typescript", `{"devDependencies":{"typescript":"^5"}}`, "TypeScript", ""},
		{"simple", `{"name":"app"}`, "Node.js", ""},
	}
	for _, c := range casos {
		dir := t.TempDir()
		escribir(t, dir, "package.json", c.contenido)
		info := DetectarProyecto(dir)
		if info.Lenguaje != c.lenguaje {
			t.Errorf("[%s] Lenguaje = %q, esperaba %q", c.nombre, info.Lenguaje, c.lenguaje)
		}
		if info.Framework != c.framework {
			t.Errorf("[%s] Framework = %q, esperaba %q", c.nombre, info.Framework, c.framework)
		}
		if info.BuildCmd != "npm run build" || info.TestCmd != "npm test" {
			t.Errorf("[%s] comandos npm incorrectos", c.nombre)
		}
	}
}

func TestDetectarProyectoRustPythonC(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "Cargo.toml", "[package]\nname = \"x\"\n")
	info := DetectarProyecto(dir)
	if info.Lenguaje != "Rust" || info.BuildCmd != "cargo build" || info.TestCmd != "cargo test" {
		t.Errorf("Rust incorrecto: %+v", info)
	}

	dir2 := t.TempDir()
	escribir(t, dir2, "requirements.txt", "requests\n")
	info2 := DetectarProyecto(dir2)
	if info2.Lenguaje != "Python" || info2.TestCmd != "pytest" {
		t.Errorf("Python incorrecto: %+v", info2)
	}

	dir3 := t.TempDir()
	escribir(t, dir3, "requirements.txt", "")
	escribir(t, dir3, "manage.py", "# django")
	info3 := DetectarProyecto(dir3)
	if info3.Framework != "Django" {
		t.Errorf("Django no detectado: %+v", info3)
	}

	dir4 := t.TempDir()
	escribir(t, dir4, "requirements.txt", "flask\n")
	escribir(t, dir4, "main.py", "# app")
	info4 := DetectarProyecto(dir4)
	if info4.Framework != "Flask/FastAPI" {
		t.Errorf("Flask/FastAPI no detectado: %+v", info4)
	}

	dir5 := t.TempDir()
	escribir(t, dir5, "CMakeLists.txt", "cmake_minimum_required(VERSION 3.10)")
	info5 := DetectarProyecto(dir5)
	if info5.Lenguaje != "C/C++" || info5.BuildCmd != "cmake --build ." || info5.TestCmd != "ctest" {
		t.Errorf("C/C++ incorrecto: %+v", info5)
	}
}

func TestDetectarProyectoDesconocido(t *testing.T) {
	dir := t.TempDir()
	info := DetectarProyecto(dir)
	if info.Lenguaje != "desconocido" {
		t.Errorf("Lenguaje = %q, esperaba desconocido", info.Lenguaje)
	}
	if info.BuildCmd != "" || info.TestCmd != "" {
		t.Errorf("proyecto desconocido no debería tener comandos: %+v", info)
	}
}

func TestResumenYPromptContexto(t *testing.T) {
	dir := t.TempDir()
	escribir(t, dir, "go.mod", "module app\n")

	info := DetectarProyecto(dir)
	resumen := info.Resumen()
	if !strings.Contains(resumen, "Lenguaje: Go") || !strings.Contains(resumen, "go test ./...") {
		t.Errorf("Resumen incompleto: %q", resumen)
	}
	ctx := info.PromptContexto(dir)
	if !strings.Contains(ctx, dir) || !strings.Contains(ctx, "Lenguaje: Go") {
		t.Errorf("PromptContexto incompleto: %q", ctx)
	}
}
