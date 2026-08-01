package core

import (
	"fmt"
	"os/exec"
	"strings"
)

func (h *Hands) wifiListar() string {
	ps := `(netsh wlan show interfaces) -match 'SSID' | ForEach-Object { $_ -replace '.*:\s*', '' }`
	out, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-Command", ps)
	if err != nil {
		return "No pude listar redes WiFi, señor."
	}
	lineas := strings.Split(strings.TrimSpace(string(out)), "\n")
	filtradas := make([]string, 0)
	for _, l := range lineas {
		if l = strings.TrimSpace(l); l != "" {
			filtradas = append(filtradas, l)
		}
	}
	if len(filtradas) == 0 {
		return "No hay redes WiFi disponibles, señor."
	}
	return fmt.Sprintf("Redes disponibles: %s", strings.Join(filtradas, ", "))
}

func (h *Hands) wifiDesconectar() string {
	if err := exec.Command("netsh", "wlan", "disconnect").Run(); err != nil {
		return "No pude desconectar el WiFi, señor."
	}
	return "WiFi desconectado, señor."
}

func (h *Hands) bluetoothActivar() string {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$radio=[Windows.Devices.Radios.Radio,Windows.System.Devices,ContentType=WindowsRuntime]::GetRadiosAsync().GetAwaiter().GetResult() | Where-Object {$_.Kind -eq 'Bluetooth'}; if($radio){[Windows.Devices.Radios.RadioAccessStatus]$radio[0].SetStateAsync([Windows.Devices.Radios.RadioState]::On).GetAwaiter().GetResult()}`)
	if err := cmd.Run(); err != nil {
		fallback := exec.Command("powershell", "-NoProfile", "-Command",
			`(New-Object -ComObject Shell.Application).ToggleDesktop(); Start-Process ms-settings:bluetooth`)
		fallback.Run()
	}
	return "Bluetooth activado, señor."
}

func (h *Hands) bluetoothDesactivar() string {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$radio=[Windows.Devices.Radios.Radio,Windows.System.Devices,ContentType=WindowsRuntime]::GetRadiosAsync().GetAwaiter().GetResult() | Where-Object {$_.Kind -eq 'Bluetooth'}; if($radio){[Windows.Devices.Radios.RadioAccessStatus]$radio[0].SetStateAsync([Windows.Devices.Radios.RadioState]::Off).GetAwaiter().GetResult()}`)
	if err := cmd.Run(); err != nil {
	}
	return "Bluetooth desactivado, señor."
}

func (h *Hands) listarProcesos() string {
	ps := `Get-Process | Sort-Object CPU -Descending | Select-Object -First 15 Name,CPU,PM | Format-Table -AutoSize | Out-String`
	out, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-Command", ps)
	if err != nil {
		return "No pude listar procesos, señor."
	}
	texto := string(out)
	fmt.Println("Procesos activos (top 15 por CPU):")
	fmt.Println(texto)
	return "Mostrando los 15 procesos con más uso de CPU, señor."
}

func (h *Hands) matarProceso(nombre string) string {
	if !esRutaSegura(nombre) {
		return fmt.Sprintf("No puedo matar '%s', señor: nombre no válido.", nombre)
	}
	proceso := nombre
	if !strings.HasSuffix(proceso, ".exe") {
		proceso += ".exe"
	}
	if esProcesoProtegido(proceso) {
		return fmt.Sprintf("No voy a matar %s, señor: es un proceso del sistema.", nombre)
	}
	if err := exec.Command("taskkill", "/IM", proceso, "/F").Run(); err != nil {
		return fmt.Sprintf("No pude matar %s. ¿Está ejecutándose?", nombre)
	}
	return fmt.Sprintf("%s terminado, señor.", nombre)
}

func (h *Hands) listarUSB() string {
	ps := `Get-PnpDevice -Class USB | Where-Object {$_.Status -eq 'OK'} | Select-Object FriendlyName | Format-Table -HideTableHeaders | Out-String`
	out, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-Command", ps)
	if err != nil {
		return "No pude listar dispositivos USB, señor."
	}
	lineas := strings.Split(strings.TrimSpace(string(out)), "\n")
	filtradas := make([]string, 0)
	for _, l := range lineas {
		if l = strings.TrimSpace(l); l != "" {
			filtradas = append(filtradas, l)
		}
	}
	if len(filtradas) == 0 {
		return "No hay dispositivos USB conectados, señor."
	}
	fmt.Println("Dispositivos USB conectados:")
	for _, l := range filtradas {
		fmt.Println(" -", l)
	}
	return fmt.Sprintf("%d dispositivos USB conectados, señor.", len(filtradas))
}

func (h *Hands) cambiarModoPantalla(modo string) string {
	modos := map[string]string{
		"duplicar": "duplicate",
		"extender": "extend",
		"unico":    "internal",
		"segundo":  "external",
	}
	m, ok := modos[modo]
	if !ok {
		return "Modo no válido, señor. Use: duplicar, extender, único, segundo."
	}
	if err := exec.Command("DisplaySwitch.exe", fmt.Sprintf("/%s", m)).Run(); err != nil {
		return fmt.Sprintf("No pude cambiar el modo de pantalla, señor: %v", err)
	}
	return fmt.Sprintf("Pantalla cambiada a modo %s, señor.", modo)
}

func (h *Hands) infoBateriaDetallada() string {
	ps := `(Get-WmiObject Win32_Battery).EstimatedChargeRemaining`
	out, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-Command", ps)
	if err != nil {
		return "No hay batería en este equipo, señor."
	}
	porcentaje := strings.TrimSpace(string(out))
	if porcentaje == "" {
		return "No hay batería en este equipo, señor."
	}
	ps2 := `(Get-WmiObject Win32_Battery).BatteryStatus`
	out2, _ := h.ejecutarConTimeout("powershell", "-NoProfile", "-Command", ps2)
	status := strings.TrimSpace(string(out2))
	estado := "desconocido"
	switch status {
	case "1":
		estado = "sin carga"
	case "2":
		estado = "cargando"
	case "3":
		estado = "descargando"
	}
	return fmt.Sprintf("Batería al %s%% (%s), señor.", porcentaje, estado)
}
