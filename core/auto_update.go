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
// rutas contra comillas simples/dobles del comando "copy".
func scriptActualizacion(exeActual, rutaDescarga string) string {
	viejo := comillaCmd(exeActual)
	nuevo := comillaCmd(rutaDescarga)
	respaldo := comillaCmd(exeActual + ".backup")
	return `@echo off
rem Espera a que el proceso JarvisOS termine (máx 15s).
echo Esperando que JarvisOS se cierre...
timeout /t 1 /nobreak >nul
for /L %%i in (1,1,15) do (
  tasklist /FI "IMAGENAME eq JarvisOS.exe" 2>nul | find /I "JarvisOS.exe" >nul
  if errorlevel 1 goto copiar
  timeout /t 1 /nobreak >nul
)
echo El proceso sigue activo; se cancela la actualizacion.
exit /B 1

:copiar
if exist ` + respaldo + ` del /q ` + respaldo + `
copy /y "` + nuevo + `" "` + viejo + `" >nul
if errorlevel 1 (
  echo No se pudo reemplazar el ejecutable.
  exit /B 1
)
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
