package core

import (
	"os"
	"path/filepath"
	"time"
)

type EstadoPanel struct {
	Hora          string `json:"hora"`
	Fecha         string `json:"fecha"`
	RAMTotal      string `json:"ram_total"`
	RAMUsada      string `json:"ram_usada"`
	RAMPorcentaje string `json:"ram_porcentaje"`
	Bateria       string `json:"bateria"`
	Uptime        string `json:"uptime"`
	IPLocal       string `json:"ip_local"`
	CPU           string `json:"cpu"`
	Procesos      string `json:"procesos"`
}

func (h *Hands) EstadoPanel() EstadoPanel {
	return EstadoPanel{
		Hora:          time.Now().Format("15:04:05"),
		Fecha:         time.Now().Format("02/01/2006"),
		RAMTotal:      h.panelRAMTotal(),
		RAMUsada:      h.panelRAMUsada(),
		RAMPorcentaje: h.panelRAMPorcentaje(),
		Bateria:       h.panelBateria(),
		Uptime:        h.panelUptime(),
		IPLocal:       h.panelIP(),
		CPU:           h.panelCPU(),
		Procesos:      h.panelProcesos(),
	}
}

func (h *Hands) panelRAMTotal() string {
	out, _ := ejecutarPS("$o=Get-CimInstance Win32_OperatingSystem; '{0:N1}' -f ($o.TotalVisibleMemorySize/1MB)")
	return out
}

func (h *Hands) panelRAMUsada() string {
	out, _ := ejecutarPS("$o=Get-CimInstance Win32_OperatingSystem; $u=$o.TotalVisibleMemorySize-$o.FreePhysicalMemory; '{0:N1}' -f ($u/1MB)")
	return out
}

func (h *Hands) panelRAMPorcentaje() string {
	out, _ := ejecutarPS("$o=Get-CimInstance Win32_OperatingSystem; $u=$o.TotalVisibleMemorySize-$o.FreePhysicalMemory; '{0}' -f [math]::Round($u/$o.TotalVisibleMemorySize*100)")
	return out
}

func (h *Hands) panelBateria() string {
	out, _ := ejecutarPS("$b=Get-CimInstance Win32_Battery; if($b){'{0}%' -f $b.EstimatedChargeRemaining}else{'Sin batería'}")
	if out == "" {
		return "Sin batería"
	}
	return out
}

func (h *Hands) panelUptime() string {
	out, _ := ejecutarPS("$b=(Get-CimInstance Win32_OperatingSystem).LastBootUpTime; $u=(Get-Date)-$b; '{0}h {1}m' -f $u.Hours,$u.Minutes")
	return out
}

func (h *Hands) panelIP() string {
	out, _ := ejecutarPS("(Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254*'} | Select-Object -First 1).IPAddress")
	if out == "" {
		return "-"
	}
	return out
}

func (h *Hands) panelCPU() string {
	out, _ := ejecutarPS("(Get-CimInstance Win32_Processor | Select-Object -First 1).Name")
	if len(out) > 40 {
		out = out[:40] + "..."
	}
	return out
}

func (h *Hands) panelProcesos() string {
	out, _ := ejecutarPS("'{0}' -f (Get-Process | Measure-Object).Count")
	return out
}

func rutaDatosJarvis() string {
	dir := filepath.Join(userProfileDir(), "JarvisOS-datos")
	os.MkdirAll(dir, 0o700)
	return dir
}

func userProfileDir() string {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home = "."
	}
	return home
}
