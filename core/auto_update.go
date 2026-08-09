package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// aplicarActualizacion instala el binario nuevo descargado y verificado
// reemplazando el ejecutable en uso. Como Windows no permite sobrescribir un
// exe en ejecución, lanza un script auxiliar que espera a que el proceso
// actual termine, copia el binario nuevo y lo relanza.
func aplicarActualizacion(rutaDescarga string) error {
	exeActual, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo ubicar el ejecutable en uso: %v", err)
	}
	exeActual, err = filepath.Abs(exeActual)
	if err != nil {
		return fmt.Errorf("ruta del ejecutable inválida: %v", err)
	}
	if strings.EqualFold(filepath.Clean(rutaDescarga), exeActual) {
		return fmt.Errorf("el binario descargado es el ejecutable actual")
	}
	if _, err := os.Stat(rutaDescarga); err != nil {
		return fmt.Errorf("el binario descargado no existe: %v", err)
	}

	script := filepath.Join(os.TempDir(), fmt.Sprintf("jarvis-actualizar-%d.cmd", time.Now().UnixNano()))
	if err := os.WriteFile(script, []byte(scriptActualizacion(exeActual, rutaDescarga)), 0o600); err != nil {
		return fmt.Errorf("no pude crear el script de actualización: %v", err)
	}

	cmd := exec.Command("cmd.exe", "/c", "start", `""`, script)
	if err := cmd.Start(); err != nil {
		_ = os.Remove(script)
		return fmt.Errorf("no pude lanzar el script de actualización: %v", err)
	}
	return nil
}

// scriptActualizacion genera el .cmd que reemplaza el ejecutable. Escapa las
// rutas contra comillas del comando "copy".
//
// La app que inicia la actualización NO se cierra sola: el script espera un
// momento (por si entra voz), luego cierra forzosamente el proceso JarvisOS,
// copia el binario nuevo y relanza. El reemplazo se reintenta porque Windows
// no libera el exe ni bien termina el proceso.
func scriptActualizacion(exeActual, rutaDescarga string) string {
	viejo := comillaCmd(exeActual)
	nuevo := comillaCmd(rutaDescarga)
	respaldo := comillaCmd(exeActual + ".backup")
	return `@echo off
rem Espera un momento por si la app se cierra sola; si no, la cierra.
timeout /t 2 /nobreak >nul
taskkill /IM "JarvisOS.exe" /F >nul 2>nul
rem Aplicacion: reintenta el reemplazo hasta que el exe quede libre.
set /a intento = 0
:copiar
set /a intento += 1
if exist ` + respaldo + ` del /q ` + respaldo + `
copy /y "` + nuevo + `" "` + viejo + `" >nul
if not errorlevel 1 goto ok
if %intento% geq 10 (
  echo No se pudo reemplazar el ejecutable tras %intento% intentos.
  exit /B 1
)
timeout /t 1 /nobreak >nul
goto copiar

:ok
echo Actualizacion aplicada; relanzando JarvisOS...
start "" "` + viejo + `"
del /q "` + nuevo + `" >nul 2>nul
exit /B 0
`
}

// comillaCmd duplica las comillas dobles para usarse seguro el .cmd
func comillaCmd(ruta string) string {
	return strings.ReplaceAll(ruta, `"`, `""`)
}
