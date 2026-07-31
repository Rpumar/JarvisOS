package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LlamadaHerramienta struct {
	Nombre    string
	Argumentos map[string]string
}

type ResultadoHerramienta struct {
	Exito bool
	Salida string
}

func ParsearHerramientas(texto string) []LlamadaHerramienta {
	var herramientas []LlamadaHerramienta
	bloques := strings.Split(texto, "---")
	for _, bloque := range bloques {
		bloque = strings.TrimSpace(bloque)
		if bloque == "" {
			continue
		}
		lineas := strings.Split(bloque, "\n")
		var h LlamadaHerramienta
		for i, l := range lineas {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "HERRAMIENTA|") {
				h.Nombre = strings.TrimPrefix(l, "HERRAMIENTA|")
			} else if strings.HasPrefix(l, "ARGUMENTOS|") {
				jsonStr := strings.TrimPrefix(l, "ARGUMENTOS|")
				json.Unmarshal([]byte(jsonStr), &h.Argumentos)
			}
			if i == len(lineas)-1 && h.Nombre != "" {
				herramientas = append(herramientas, h)
				h = LlamadaHerramienta{}
			}
		}
		if h.Nombre != "" {
			herramientas = append(herramientas, h)
		}
	}
	return herramientas
}

type EjecutorHerramientas struct {
	WorkspaceRoot string
}

func NuevoEjecutorHerramientas(root string) *EjecutorHerramientas {
	return &EjecutorHerramientas{WorkspaceRoot: root}
}

func (e *EjecutorHerramientas) Ejecutar(h LlamadaHerramienta) ResultadoHerramienta {
	switch h.Nombre {
	case "read_file":
		return e.leerArchivo(h.Argumentos)
	case "write_file":
		return e.escribirArchivo(h.Argumentos)
	case "edit_file":
		return e.editarArchivo(h.Argumentos)
	case "glob":
		return e.buscarArchivos(h.Argumentos)
	case "grep":
		return e.buscarContenido(h.Argumentos)
	case "run":
		return e.ejecutarComando(h.Argumentos)
	case "read_dir":
		return e.leerDirectorio(h.Argumentos)
	case "run_test":
		return e.ejecutarTests(h.Argumentos)
	case "leer_entrada":
		return ResultadoHerramienta{Exito: true, Salida: "(pendiente de entrada del usuario)"}
	default:
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Herramienta desconocida: %s", h.Nombre)}
	}
}

func (e *EjecutorHerramientas) rutaAbsoluta(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.WorkspaceRoot, path)
}

func (e *EjecutorHerramientas) leerArchivo(args map[string]string) ResultadoHerramienta {
	path := e.rutaAbsoluta(args["path"])
	contenido, err := os.ReadFile(path)
	if err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al leer '%s': %v", path, err)}
	}
	texto := string(contenido)
	if len(texto) > 50000 {
		texto = texto[:50000] + "\n... (archivo truncado a 50000 caracteres)"
	}
	return ResultadoHerramienta{Exito: true, Salida: fmt.Sprintf("=== %s ===\n%s", path, texto)}
}

func (e *EjecutorHerramientas) escribirArchivo(args map[string]string) ResultadoHerramienta {
	path := e.rutaAbsoluta(args["path"])
	content := args["content"]
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al crear directorio: %v", err)}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al escribir: %v", err)}
	}
	return ResultadoHerramienta{Exito: true, Salida: fmt.Sprintf("Archivo escrito: %s (%d bytes)", path, len(content))}
}

func (e *EjecutorHerramientas) editarArchivo(args map[string]string) ResultadoHerramienta {
	path := e.rutaAbsoluta(args["path"])
	old := args["old"]
	newText := args["new"]

	contenido, err := os.ReadFile(path)
	if err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al leer '%s': %v", path, err)}
	}

	texto := string(contenido)
	if !strings.Contains(texto, old) {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("No se encontró el texto a reemplazar en '%s'", path)}
	}

	nuevo := strings.Replace(texto, old, newText, 1)
	if err := os.WriteFile(path, []byte(nuevo), 0o644); err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al escribir: %v", err)}
	}
	return ResultadoHerramienta{Exito: true, Salida: fmt.Sprintf("Archivo editado: %s", path)}
}

func (e *EjecutorHerramientas) buscarArchivos(args map[string]string) ResultadoHerramienta {
	pattern := args["pattern"]
	if pattern == "" {
		return ResultadoHerramienta{Exito: false, Salida: "pattern requerido"}
	}

	var resultados []string
	walkErr := filepath.WalkDir(e.WorkspaceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(e.WorkspaceRoot, path)
		matched, _ := filepath.Match(pattern, rel)
		if matched {
			resultados = append(resultados, rel)
		}
		return nil
	})

	if walkErr != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al buscar: %v", walkErr)}
	}

	if len(resultados) == 0 {
		return ResultadoHerramienta{Exito: true, Salida: "No se encontraron archivos."}
	}

	if len(resultados) > 50 {
		resultados = resultados[:50]
	}
	return ResultadoHerramienta{Exito: true, Salida: strings.Join(resultados, "\n")}
}

func (e *EjecutorHerramientas) buscarContenido(args map[string]string) ResultadoHerramienta {
	pattern := args["pattern"]
	if pattern == "" {
		return ResultadoHerramienta{Exito: false, Salida: "pattern requerido"}
	}
	include := args["include"]
	path := e.WorkspaceRoot

	cmdArgs := []string{"-n"}
	if include != "" {
		cmdArgs = append(cmdArgs, "--include", include)
	}
	cmdArgs = append(cmdArgs, "--max-count", "5", pattern, path)

	out, err := exec.Command("rg", cmdArgs...).Output()
	if err != nil {
		out2, _ := exec.Command("findstr", "/s", "/n", pattern, filepath.Join(path, "*"+include)).Output()
		if len(out2) > 0 {
			texto := string(out2)
			if len(texto) > 10000 {
				texto = texto[:10000] + "\n... (truncado)"
			}
			return ResultadoHerramienta{Exito: true, Salida: texto}
		}
		return ResultadoHerramienta{Exito: true, Salida: "No se encontraron resultados."}
	}

	texto := string(out)
	if len(texto) > 10000 {
		texto = texto[:10000] + "\n... (truncado)"
	}
	return ResultadoHerramienta{Exito: true, Salida: texto}
}

func (e *EjecutorHerramientas) ejecutarComando(args map[string]string) ResultadoHerramienta {
	comando := args["command"]
	if comando == "" {
		return ResultadoHerramienta{Exito: false, Salida: "command requerido"}
	}

	for _, peligroso := range []string{"rm ", "del ", "format ", "diskpart", "shutdown", "taskkill"} {
		if strings.Contains(strings.ToLower(comando), peligroso) {
			return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Comando bloqueado por seguridad: contiene '%s'", peligroso)}
		}
	}

	cmd := exec.Command("cmd", "/C", comando)
	cmd.Dir = e.WorkspaceRoot
	out, err := cmd.CombinedOutput()
	salida := string(out)
	if len(salida) > 10000 {
		salida = salida[:10000] + "\n... (salida truncada a 10000 caracteres)"
	}
	if err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error: %v\nSalida: %s", err, salida)}
	}
	return ResultadoHerramienta{Exito: true, Salida: salida}
}

func (e *EjecutorHerramientas) leerDirectorio(args map[string]string) ResultadoHerramienta {
	path := e.rutaAbsoluta(args["path"])
	entries, err := os.ReadDir(path)
	if err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Error al leer directorio: %v", err)}
	}
	var lista []string
	for _, entry := range entries {
		prefijo := "  "
		if entry.IsDir() {
			prefijo = "  📁"
		} else {
			prefijo = "  📄"
		}
		lista = append(lista, fmt.Sprintf("%s %s", prefijo, entry.Name()))
	}
	return ResultadoHerramienta{Exito: true, Salida: fmt.Sprintf("Directorio: %s\n%s", path, strings.Join(lista, "\n"))}
}

func (e *EjecutorHerramientas) ejecutarTests(args map[string]string) ResultadoHerramienta {
	comando := args["command"]
	if comando == "" {
		comando = "go test ./..."
	}
	cmd := exec.Command("cmd", "/C", comando)
	cmd.Dir = e.WorkspaceRoot
	out, err := cmd.CombinedOutput()
	salida := string(out)
	if len(salida) > 5000 {
		salida = salida[:5000] + "\n... (truncado)"
	}
	if err != nil {
		return ResultadoHerramienta{Exito: false, Salida: fmt.Sprintf("Tests fallaron: %v\n%s", err, salida)}
	}
	return ResultadoHerramienta{Exito: true, Salida: salida}
}
