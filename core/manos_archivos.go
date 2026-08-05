package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func carpetaJarvisOS() string {
	return filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos")
}

func carpetaEscritorio() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	for _, nombre := range []string{"Desktop", "Escritorio"} {
		candidata := filepath.Join(home, nombre)
		if info, e := os.Stat(candidata); e == nil && info.IsDir() {
			return candidata
		}
	}
	return home
}

func carpetasUsuario() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Escritorio"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Documentos"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Descargas"),
	}
}

func (h *Hands) buscarArchivo(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué archivo quiere que busque, señor?"
	}
	encontrados := buscarEnUsuario(nombre, 10)
	if len(encontrados) == 0 {
		return fmt.Sprintf("No encontré archivos con '%s', señor.", nombre)
	}
	fmt.Println("Coincidencias encontradas:")
	for _, r := range encontrados {
		fmt.Println(" -", r)
	}
	return fmt.Sprintf("Encontré %d archivos, señor. Vea las rutas en la consola.", len(encontrados))
}

func buscarEnUsuario(nombre string, limite int) []string {
	nombre = strings.ToLower(nombre)
	var encontrados []string
	for _, raiz := range carpetasUsuario() {
		if info, err := os.Stat(raiz); err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.Walk(raiz, func(ruta string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.Contains(strings.ToLower(info.Name()), nombre) {
				encontrados = append(encontrados, ruta)
			}
			if len(encontrados) >= limite {
				return filepath.SkipDir
			}
			return nil
		})
		if len(encontrados) >= limite {
			break
		}
	}
	return encontrados
}

func (h *Hands) encontrarArchivo(nombre string) string {
	encontrados := buscarEnUsuario(nombre, 1)
	if len(encontrados) == 0 {
		return ""
	}
	return encontrados[0]
}

func (h *Hands) abrirUbicacion(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué archivo quiere ubicar, señor?"
	}
	ruta := h.encontrarArchivo(nombre)
	if ruta == "" {
		return fmt.Sprintf("No encontré '%s' en sus carpetas de usuario, señor.", nombre)
	}
	ps := fmt.Sprintf(`$p = '%s'; if (Test-Path $p) { explorer.exe /select, $p }`, strings.ReplaceAll(ruta, "'", "''"))
	if _, err := ejecutarPS(ps); err != nil {
		return fmt.Sprintf("No pude abrir la ubicación: %v", err)
	}
	return fmt.Sprintf("Abriendo la ubicación de '%s', señor.", filepath.Base(ruta))
}

func (h *Hands) borrarArchivo(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué archivo quiere borrar, señor?"
	}
	ruta := h.encontrarArchivo(nombre)
	if ruta == "" {
		return fmt.Sprintf("No encontré '%s' en sus carpetas de usuario, señor.", nombre)
	}
	ps := fmt.Sprintf(`Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('%s','OnlyErrorDialogs','SendToRecycleBin')`, strings.ReplaceAll(ruta, "'", "''"))
	if _, err := ejecutarPS(ps); err != nil {
		return fmt.Sprintf("No pude borrar el archivo: %v", err)
	}
	return fmt.Sprintf("Archivo '%s' enviado a la papelera de reciclaje, señor.", filepath.Base(ruta))
}

func (h *Hands) tomarNota(texto string) string {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return "¿Qué quiere que anote, señor?"
	}
	dir := carpetaJarvisOS()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Sprintf("No pude guardar la nota: %v", err)
	}
	ruta := filepath.Join(dir, "notas.txt")
	fecha := time.Now().Format("02/01/2006 15:04")
	f, err := os.OpenFile(ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Sprintf("No pude guardar la nota: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("[%s] %s\n", fecha, texto)); err != nil {
		return fmt.Sprintf("No pude guardar la nota: %v", err)
	}
	return fmt.Sprintf("Anotado, señor: %s", texto)
}

func (h *Hands) leerNotas() string {
	ruta := filepath.Join(carpetaJarvisOS(), "notas.txt")
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return "No tiene notas guardadas todavía, señor."
	}
	lineas := strings.Split(strings.TrimSpace(string(contenido)), "\n")
	total := len(lineas)
	inicio := total - 5
	if inicio < 0 {
		inicio = 0
	}
	fmt.Println("Últimas notas:")
	for i := inicio; i < total; i++ {
		fmt.Println(" -", lineas[i])
	}
	return fmt.Sprintf("Tiene %d notas guardadas, señor. Vea las últimas en la consola.", total)
}
