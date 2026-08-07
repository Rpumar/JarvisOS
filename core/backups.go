package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// backupsDir es la subcarpeta dentro de JarvisOS-datos donde se guardan los
// backups con rotación. Cada backup es una copia completa de la carpeta de
// datos del usuario (excepto la propia carpeta de backups).
const backupsDir = "backups"

// BackupsMax es cuántos backups se conservan por defecto (rotación FIFO).
const BackupsMax = 7

// RealizarBackup copia la carpeta de datos del usuario a backups/backup-AAAA-MM-DD-HHMMSS
// y rota los más antiguos para no superar maxCantidad. Devuelve la ruta del
// backup creado. Es local y no toca la configuración con secretos del repo.
func RealizarBackup(datosDir string, maxCantidad int) (string, error) {
	if strings.TrimSpace(datosDir) == "" {
		return "", fmt.Errorf("carpeta de datos vacía")
	}
	if maxCantidad < 1 {
		maxCantidad = BackupsMax
	}
	destino := filepath.Join(datosDir, backupsDir)
	if err := os.MkdirAll(destino, 0o700); err != nil {
		return "", err
	}
	nombre := "backup-" + time.Now().Format("2006-01-02-150405.000000000")
	ruta := filepath.Join(destino, nombre)
	if _, err := os.Stat(ruta); err == nil {
		nombre += "-" + fmt.Sprintf("%d", time.Now().UnixNano())
		ruta = filepath.Join(destino, nombre)
	}
	if err := copiarArbol(datosDir, ruta); err != nil {
		return "", err
	}
	if err := rotarBackups(destino, maxCantidad); err != nil {
		return ruta, err
	}
	return ruta, nil
}

// copiarArbol copia recursivamente el contenido de origen a destino, sin
// incluir la propia carpeta de backups (evita recursión infinita).
func copiarArbol(origen, destino string) error {
	return filepath.Walk(origen, func(ruta string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(origen, ruta)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return os.MkdirAll(destino, 0o700)
		}
		if rel == backupsDir || strings.HasPrefix(rel, backupsDir+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dst := filepath.Join(destino, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o700)
		}
		return copiarArchivo(ruta, dst)
	})
}

// copiarArchivo copia un archivo preservando permisos de solo lectura/escritura.
func copiarArchivo(origen, destino string) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o700); err != nil {
		return err
	}
	src, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

// rotarBackups borra los backups más antiguos hasta dejar maxCantidad.
func rotarBackups(destino string, maxCantidad int) error {
	entradas, err := os.ReadDir(destino)
	if err != nil {
		return err
	}
	var backups []string
	for _, e := range entradas {
		if e.IsDir() && strings.HasPrefix(e.Name(), "backup-") {
			backups = append(backups, e.Name())
		}
	}
	if len(backups) <= maxCantidad {
		return nil
	}
	sort.Strings(backups)
	for _, nombre := range backups[:len(backups)-maxCantidad] {
		if err := os.RemoveAll(filepath.Join(destino, nombre)); err != nil {
			return err
		}
	}
	return nil
}

// ListarBackups devuelve los nombres de los backups existentes (más reciente
// primero) o vacío si no hay carpeta de backups.
func ListarBackups(datosDir string) []string {
	destino := filepath.Join(datosDir, backupsDir)
	entradas, err := os.ReadDir(destino)
	if err != nil {
		return nil
	}
	var backups []string
	for _, e := range entradas {
		if e.IsDir() && strings.HasPrefix(e.Name(), "backup-") {
			backups = append(backups, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))
	return backups
}

// hacerBackupManual crea un backup ahora y lo reporta por voz.
func (h *Hands) hacerBackupManual() string {
	ruta, err := RealizarBackup(h.DatosDir, BackupsMax)
	if err != nil {
		return fmt.Sprintf("No pude hacer el respaldo, señor: %v", err)
	}
	return fmt.Sprintf("Respaldo creado, señor: %s", ruta)
}

// listarBackupsVoz describe los respaldos disponibles.
func (h *Hands) listarBackupsVoz() string {
	backups := ListarBackups(h.DatosDir)
	if len(backups) == 0 {
		return "No hay respaldos todavía, señor. Diga 'hacé un respaldo' para crear el primero."
	}
	return "Tengo estos respaldos, señor: " + strings.Join(backups, ", ") + "."
}
