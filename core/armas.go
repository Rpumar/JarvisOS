package core

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// tiempoMaximoPS evita que un comando de PowerShell se cuelgue para siempre:
// pasados los 5 minutos se cancela y la petición responde con error.
const tiempoMaximoPS = 5 * time.Minute

func ejecutarPS(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tiempoMaximoPS)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("comando cancelado por tiempo límite: %v", ctx.Err())
	}
	return strings.TrimSpace(string(out)), err
}

func (h *Hands) velocidadInternet() string {
	out, err := ejecutarPS(`
$u='https://speed.cloudflare.com/__down?bytes=20000000'
$w=$null
$r=Measure-Command {$w=Invoke-WebRequest $u -UseBasicParsing}
if(-not $w){'sin respuesta';exit}
$mb=($w.RawContentLength*8)/$r.TotalSeconds/1e6
"Velocidad de descarga: {0:N1} Mbps (tiempo: {1:N1} s)" -f $mb,$r.TotalSeconds`)
	if err != nil {
		return "No pude medir la velocidad de internet, señor."
	}
	fmt.Println(out)
	return "Velocidad de internet medida, señor. Mire la consola para ver el resultado."
}

func (h *Hands) escanearRed() string {
	out, err := ejecutarPS("arp -a")
	if err != nil {
		return "No pude escanear la red, señor."
	}
	fmt.Println(out)
	return "Red local escaneada, señor. Mire la consola para ver los dispositivos."
}

func (h *Hands) limpiarDNS() string {
	if _, err := h.ejecutarConTimeout("ipconfig", "/flushdns"); err != nil {
		return "No pude limpiar el DNS, señor."
	}
	return "Cache DNS limpiado, señor."
}

func (h *Hands) infoRedDetallada() string {
	out, err := ejecutarPS(`
Get-NetAdapter | Where-Object Status -eq 'Up' |
  Select-Object Name,InterfaceDescription,LinkSpeed,MacAddress |
  Format-List | Out-String`)
	if err != nil || out == "" {
		return "No pude obtener info de las tarjetas de red, señor."
	}
	fmt.Println(out)
	return "Info de red obtenida, señor. Mire la consola para ver los adaptadores."
}

func (h *Hands) usoRAM() string {
	out, err := ejecutarPS(`
$os=Get-CimInstance Win32_OperatingSystem
$tot=$os.TotalVisibleMemorySize/1MB
$libre=$os.FreePhysicalMemory/1MB
$usada=$tot-$libre
$pct=[math]::Round($usada/$tot*100)
"RAM: {0:N1} GB usados de {1:N1} GB ({2}%), {3:N1} GB libres" -f $usada,$tot,$pct,$libre`)
	if err != nil {
		return "No pude medir la RAM, señor."
	}
	fmt.Println(out)
	return out
}

func (h *Hands) planEnergia() string {
	out, err := ejecutarPS("powercfg /getactivescheme")
	if err != nil {
		return "No pude obtener el plan de energía, señor."
	}
	return fmt.Sprintf("Plan de energía activo: %s, señor.", strings.TrimSpace(out))
}
