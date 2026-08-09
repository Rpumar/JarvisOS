package core

import (
	"strings"
	"testing"
)

func TestScriptActualizacionCierraProcesoYReintenta(t *testing.T) {
	script := scriptActualizacion(`C:\prueba\JarvisOS.exe`, `C:\descargas\JarvisOS.exe`)
	if !strings.Contains(script, `taskkill /IM "JarvisOS.exe" /F`) {
		t.Error("el script debe cerrar forzosamente el proceso JarvisOS")
	}
	if !strings.Contains(script, "set /a intento = 0") {
		t.Error("el script debe inicializar el contador de reintentos")
	}
	if !strings.Contains(script, "if %intento% geq 10") {
		t.Error("el script debe abortar tras 10 intentos de reemplazo")
	}
	if !strings.Contains(script, "goto copiar") {
		t.Error("el script debe reintentar el copy en caso de fallo")
	}
	if !strings.Contains(script, `del /q C:\prueba\JarvisOS.exe.backup`) {
		t.Error("el script debe limpiar el respaldo previo antes de copiar")
	}
}

func TestScriptActualizacionRelanzaCuandoFalta(t *testing.T) {
	script := scriptActualizacion(`C:\prueba\JarvisOS.exe`, `C:\descarga\Jarvis-v.exe`)
	if !strings.Contains(script, `start "" "C:\prueba\JarvisOS.exe"`) {
		t.Error("el script debe relanzar el ejecutable actual al terminar")
	}
}

func TestComillaCmdEscapaComillasDobles(t *testing.T) {
	casos := map[string]string{
		`sin comillas`:        `sin comillas`,
		`con "comilla" doble`: `con ""comilla"" doble`,
		`""`:                   `""""`,
		`ruta con \" y \\`:     `ruta con \"" y \\`,
	}
	for original, esperado := range casos {
		got := comillaCmd(original)
		if got != esperado {
			t.Errorf("comillaCmd(%q) = %q, esperaba %q", original, got, esperado)
		}
	}
}