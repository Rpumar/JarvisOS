package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (h *Hands) enviarNotificacion(texto string) string {
	if texto == "" {
		texto = "Esto es una notificación de prueba, señor."
	}
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$ni = New-Object System.Windows.Forms.NotifyIcon
$ni.Icon = [System.Drawing.SystemIcons]::Information
$ni.Visible = $true
$ni.BalloonTipTitle = 'JARVIS'
$ni.BalloonTipText = '%s'
$ni.ShowBalloonTip(5000)
Start-Sleep -Seconds 6
$ni.Dispose()`, texto)
	out, err := ejecutarPS(script)
	if err != nil && out == "" {
		return "No pude mostrar la notificación, señor."
	}
	return fmt.Sprintf("Notificación mostrada, señor: %s", texto)
}

func (h *Hands) comprimirCarpeta(cmd string) string {
	nombre := extraerObjeto(cmd, []string{"comprimí la carpeta ", "comprimi la carpeta ", "comprimir la carpeta ", "comprimí ", "comprimi ", "comprimir ", "comprime "})
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué carpeta desea comprimir, señor? Por ejemplo: 'comprimí la carpeta descargas'."
	}
	origen := resolverCarpetaUsuario(nombre)
	if origen == "" {
		return "Solo puedo comprimir descargas, escritorio, documentos, música o imágenes, señor."
	}
	if !existeRuta(origen) {
		return fmt.Sprintf("No encontré la carpeta '%s', señor.", nombre)
	}
	destino := origen + ".zip"
	out, err := ejecutarPS(fmt.Sprintf("Compress-Archive -Path '%s' -DestinationPath '%s' -Force", origen, destino))
	if err != nil && out == "" {
		return fmt.Sprintf("No pude comprimir la carpeta, señor: %v", err)
	}
	return fmt.Sprintf("Carpeta '%s' comprimida en %s.zip, señor.", nombre, origen)
}

func (h *Hands) descomprimirArchivo(cmd string) string {
	nombre := extraerObjeto(cmd, []string{"descomprimí ", "descomprimi ", "descomprimir ", "descomprime "})
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué archivo zip desea descomprimir, señor?"
	}
	if !strings.HasSuffix(strings.ToLower(nombre), ".zip") {
		nombre += ".zip"
	}
	archivo := nombre
	if !existeRuta(archivo) {
		candidato := filepath.Join(os.Getenv("USERPROFILE"), "Downloads", nombre)
		if existeRuta(candidato) {
			archivo = candidato
		} else {
			return fmt.Sprintf("No encontré el archivo '%s' ni en Descargas, señor.", nombre)
		}
	}
	destino := strings.TrimSuffix(archivo, ".zip")
	exec.Command("cmd", "/C", "mkdir", destino).Run()
	out, err := ejecutarPS(fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", archivo, destino))
	if err != nil && out == "" {
		return fmt.Sprintf("No pude descomprimir, señor: %v", err)
	}
	return fmt.Sprintf("Archivo %s descomprimido en %s, señor.", filepath.Base(archivo), destino)
}

func (h *Hands) expulsarDisco() string {
	out, err := ejecutarPS(`
$sh = New-Object -ComObject Shell.Application
$n=0
$sh.Namespace(17).Items() | ForEach-Object { $_.InvokeVerb('Eject'); $n++ }
"Unidades expulsadas: {0}" -f $n`)
	if err != nil {
		return "No pude expulsar los dispositivos, señor."
	}
	fmt.Println(out)
	return "Dispositivos extraíbles expulsados, señor."
}

func (h *Hands) mantenerDespierto() string {
	script := "powercfg /change standby-timeout-ac 0; powercfg /change standby-timeout-dc 0; powercfg /change monitor-timeout-ac 0; powercfg /change hibernate-timeout-ac 0"
	if _, err := ejecutarPS(script); err != nil {
		return "No pude cambiar la configuración de suspensión, señor."
	}
	return "La PC se mantendrá despierta, señor. Diga 'activar suspensión' para restaurarla."
}

func (h *Hands) activarSuspension() string {
	script := "powercfg /change standby-timeout-ac 15; powercfg /change standby-timeout-dc 10; powercfg /change monitor-timeout-ac 10"
	if _, err := ejecutarPS(script); err != nil {
		return "No pude restaurar la suspensión, señor."
	}
	return "Suspensión restaurada, señor: la PC dormirá tras 15 minutos."
}

func (h *Hands) probarSonido() string {
	if _, err := ejecutarPS("[console]::Beep(700,250); [console]::Beep(900,250); [console]::Beep(1100,350)"); err != nil {
		return "No pude reproducir el sonido de prueba, señor."
	}
	return "Sonido de prueba reproducido, señor. ¿Lo escuchó?"
}

func (h *Hands) listarAudio() string {
	out, err := ejecutarPS(`
Get-CimInstance Win32_SoundDevice | Select-Object -ExpandProperty Name | Where-Object {$_} | Out-String`)
	if err != nil || out == "" {
		return "No detecté dispositivos de audio, señor."
	}
	lineas := filtrarLineas(out)
	if len(lineas) == 0 {
		return "No detecté dispositivos de audio, señor."
	}
	fmt.Println("Dispositivos de audio:")
	for _, l := range lineas {
		fmt.Println(" -", l)
	}
	return fmt.Sprintf("%d dispositivos de audio detectados, señor.", len(lineas))
}

func (h *Hands) listarCamaras() string {
	out, err := ejecutarPS(`
Get-PnpDevice -Class Camera -Status OK -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FriendlyName | Out-String`)
	if err != nil || out == "" {
		return "No detecté cámaras en este equipo, señor."
	}
	lineas := filtrarLineas(out)
	if len(lineas) == 0 {
		return "No detecté cámaras en este equipo, señor."
	}
	fmt.Println("Cámaras detectadas:")
	for _, l := range lineas {
		fmt.Println(" -", l)
	}
	return fmt.Sprintf("%d cámaras detectadas, señor.", len(lineas))
}

func (h *Hands) informeSistema() string {
	info := []struct{ etiqueta, valor string }{
		{"Procesador", h.infoCPU()},
		{"RAM", h.infoRAM()},
		{"Sistema", h.infoSO()},
		{"Arquitectura", h.infoArquitectura()},
		{"Núcleos", h.infoNucleos()},
		{"Tiempo activo", h.infoUptime()},
		{"Usuario", h.infoUsuario()},
		{"Equipo", h.infoPC()},
	}
	fmt.Println("=== INFORME DEL SISTEMA ===")
	for _, i := range info {
		fmt.Printf("%s: %s\n", i.etiqueta, i.valor)
	}
	fmt.Println("=== FIN DEL INFORME ===")
	return fmt.Sprintf("Informe del sistema generado con %d secciones, señor. Mire la consola.", len(info))
}

func (h *Hands) verPortapapeles() string {
	out, err := ejecutarPS("Get-Clipboard")
	if err != nil || out == "" {
		return "El portapapeles está vacío, señor."
	}
	texto := strings.TrimSpace(out)
	if len(texto) > 200 {
		texto = texto[:200] + "..."
	}
	return fmt.Sprintf("En el portapapeles hay: %s, señor.", texto)
}

func resolverCarpetaUsuario(nombre string) string {
	switch strings.ToLower(strings.TrimSpace(nombre)) {
	case "descargas":
		return filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	case "escritorio":
		return filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	case "documentos":
		return filepath.Join(os.Getenv("USERPROFILE"), "Documents")
	case "música", "musica":
		return filepath.Join(os.Getenv("USERPROFILE"), "Music")
	case "imágenes", "imagenes", "fotos":
		return filepath.Join(os.Getenv("USERPROFILE"), "Pictures")
	}
	return ""
}

func existeRuta(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func filtrarLineas(texto string) []string {
	lineas := strings.Split(texto, "\n")
	resultado := make([]string, 0, len(lineas))
	for _, l := range lineas {
		if l = strings.TrimSpace(l); l != "" {
			resultado = append(resultado, l)
		}
	}
	return resultado
}
