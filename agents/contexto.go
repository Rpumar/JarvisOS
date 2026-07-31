package agents

import (
	"os"
	"path/filepath"
	"strings"
)

type ProyectoInfo struct {
	Lenguaje    string
	Framework   string
	BuildCmd    string
	TestCmd     string
	Dependencias []string
}

func DetectarProyecto(root string) *ProyectoInfo {
	info := &ProyectoInfo{}

	if existe(filepath.Join(root, "go.mod")) {
		info.Lenguaje = "Go"
		info.BuildCmd = "go build ./..."
		info.TestCmd = "go test ./..."
		data, _ := os.ReadFile(filepath.Join(root, "go.mod"))
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				info.Framework = strings.TrimPrefix(line, "module ")
			}
		}
		return info
	}

	if existe(filepath.Join(root, "package.json")) {
		info.Lenguaje = "Node.js"
		info.BuildCmd = "npm run build"
		info.TestCmd = "npm test"
		data, _ := os.ReadFile(filepath.Join(root, "package.json"))
		content := string(data)
		if strings.Contains(content, "next") {
			info.Framework = "Next.js"
		} else if strings.Contains(content, "react") {
			info.Framework = "React"
		} else if strings.Contains(content, "express") {
			info.Framework = "Express"
		}
		if strings.Contains(content, "typescript") || strings.Contains(content, "TypeScript") {
			info.Lenguaje = "TypeScript"
		}
		return info
	}

	if existe(filepath.Join(root, "Cargo.toml")) {
		info.Lenguaje = "Rust"
		info.BuildCmd = "cargo build"
		info.TestCmd = "cargo test"
		return info
	}

	if existe(filepath.Join(root, "requirements.txt")) || existe(filepath.Join(root, "pyproject.toml")) {
		info.Lenguaje = "Python"
		info.BuildCmd = "pip install -r requirements.txt"
		info.TestCmd = "pytest"
		if existe(filepath.Join(root, "manage.py")) {
			info.Framework = "Django"
		} else if existe(filepath.Join(root, "app.py")) || existe(filepath.Join(root, "main.py")) {
			info.Framework = "Flask/FastAPI"
		}
		return info
	}

	if existe(filepath.Join(root, "CMakeLists.txt")) {
		info.Lenguaje = "C/C++"
		info.BuildCmd = "cmake --build ."
		info.TestCmd = "ctest"
		return info
	}

	info.Lenguaje = "desconocido"
	return info
}

func (p *ProyectoInfo) Resumen() string {
	parts := []string{"Lenguaje: " + p.Lenguaje}
	if p.Framework != "" {
		parts = append(parts, "Framework: "+p.Framework)
	}
	if p.BuildCmd != "" {
		parts = append(parts, "Build: "+p.BuildCmd)
	}
	if p.TestCmd != "" {
		parts = append(parts, "Test: "+p.TestCmd)
	}
	return strings.Join(parts, " | ")
}

func (p *ProyectoInfo) PromptContexto(root string) string {
	return strings.Join([]string{
		"Directorio del proyecto: " + root,
		p.Resumen(),
	}, "\n")
}

func existe(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
