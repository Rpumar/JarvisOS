package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// scriptDiagnostico recopila un retrato profundo del sistema como JSON.
// Se ejecuta en PowerShell y se parsea en Go. Nunca modifica nada: es lectura.
const scriptDiagnostico = `
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8
$ErrorActionPreference='SilentlyContinue'
$datos=[ordered]@{}
$os=Get-CimInstance Win32_OperatingSystem
$datos.os=$os.Caption
$datos.version=$os.Version
$datos.uptime_dias=[math]::Round(((Get-Date)-$os.LastBootUpTime).TotalDays,2)
$ramTotal=$os.TotalVisibleMemorySize/1MB
$ramLibre=$os.FreePhysicalMemory/1MB
$ramUsada=$ramTotal-$ramLibre
$datos.ram_total_gb=[math]::Round($ramTotal,1)
$datos.ram_usada_gb=[math]::Round($ramUsada,1)
$datos.ram_porcentaje=[math]::Round(($ramUsada/$ramTotal)*100)
$cpu=Get-CimInstance Win32_Processor | Select-Object -First 1
$datos.cpu_porcentaje=[math]::Round($cpu.LoadPercentage)
$datos.cpu_nombre=$cpu.Name
$datos.cpu_nucleos=$cpu.NumberOfCores
$discos=@()
Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | ForEach-Object {
  $discos += [ordered]@{ letra=$_.DeviceID; total_gb=[math]::Round($_.Size/1GB,1); libre_gb=[math]::Round($_.FreeSpace/1GB,1); porcentaje=[math]::Round((($_.Size-$_.FreeSpace)/$_.Size)*100) }
}
$datos.discos=$discos
$fisicos=@()
Get-PhysicalDisk | ForEach-Object { $fisicos += [ordered]@{ nombre=$_.FriendlyName; salud=$_.HealthStatus } }
$datos.discos_fisicos=$fisicos
$topCpu=@()
Get-Process | Sort-Object CPU -Descending | Select-Object -First 5 | ForEach-Object { $topCpu += [ordered]@{ nombre=$_.Name; cpu_seg=[math]::Round($_.CPU); pid=$_.Id } }
$datos.top_cpu=$topCpu
$topRam=@()
Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 5 | ForEach-Object { $topRam += [ordered]@{ nombre=$_.Name; mb=[math]::Round($_.WorkingSet64/1MB); pid=$_.Id } }
$datos.top_ram=$topRam
$serv=@()
Get-CimInstance Win32_Service -Filter "StartMode='Auto'" | Where-Object { $_.State -ne 'Running' } | Select-Object -First 8 | ForEach-Object { $serv += [ordered]@{ nombre=$_.Name; estado=$_.State } }
$datos.servicios_auto_caidos=$serv
$errs=@()
try {
  Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2; StartTime=(Get-Date).AddHours(-24)} -MaxEvents 6 -ErrorAction Stop | ForEach-Object {
    $m=$_.Message
    if($m.Length -gt 150){$m=$m.Substring(0,150)}
    $errs += [ordered]@{ hora=$_.TimeCreated.ToString('dd/MM HH:mm'); fuente=$_.ProviderName; mensaje=$m }
  }
} catch {}
$datos.eventos_error=$errs
$tempGB=(Get-ChildItem ([System.IO.Path]::GetTempPath()) -Recurse -Force -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum/1GB
$datos.temp_gb=[math]::Round($tempGB,2)
$bat=Get-CimInstance Win32_Battery | Select-Object -First 1
if($bat){ $datos.bateria=[ordered]@{ porcentaje=$bat.EstimatedChargeRemaining; conectada=($bat.BatteryStatus -eq 2) } } else { $datos.bateria=$null }
$ips=@()
Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | Where-Object { $_.IPAddress -ne '127.0.0.1' -and $_.IPAddress -notlike '169.254*' } | ForEach-Object { $ips += $_.IPAddress }
$datos.ips=$ips
$inicio=@()
(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run' -ErrorAction SilentlyContinue).PSObject.Properties | Where-Object { $_.Name -notlike 'PS*' } | ForEach-Object { $inicio += $_.Name }
(Get-ItemProperty 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run' -ErrorAction SilentlyContinue).PSObject.Properties | Where-Object { $_.Name -notlike 'PS*' } | ForEach-Object { $inicio += $_.Name }
$datos.inicio=$inicio
$datos.reinicio_pendiente=(Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired') -or (Test-Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\PendingFileRenameOperations')
$datos | ConvertTo-Json -Depth 4
`

type DiscoLogico struct {
	Letra     string  `json:"letra"`
	TotalGB   float64 `json:"total_gb"`
	LibreGB   float64 `json:"libre_gb"`
	Porcentaje int    `json:"porcentaje"`
}

type DiscoFisico struct {
	Nombre string `json:"nombre"`
	Salud  string `json:"salud"`
}

type ProcInfo struct {
	Nombre  string  `json:"nombre"`
	CPUSeg  float64 `json:"cpu_seg"`
	MB      int     `json:"mb"`
	PID     int     `json:"pid"`
}

type ServicioInfo struct {
	Nombre string `json:"nombre"`
	Estado string `json:"estado"`
}

type EventoInfo struct {
	Hora    string `json:"hora"`
	Fuente  string `json:"fuente"`
	Mensaje string `json:"mensaje"`
}

type BateriaInfo struct {
	Porcentaje int  `json:"porcentaje"`
	Conectada  bool `json:"conectada"`
}

type Diagnostico struct {
	OS             string          `json:"os"`
	Version        string          `json:"version"`
	UptimeDias     float64         `json:"uptime_dias"`
	RAMTotalGB     float64         `json:"ram_total_gb"`
	RAMUsadaGB     float64         `json:"ram_usada_gb"`
	RAMPorcentaje  int             `json:"ram_porcentaje"`
	CPUPorcentaje  int             `json:"cpu_porcentaje"`
	CPUNombre      string          `json:"cpu_nombre"`
	CPUNucleos     int             `json:"cpu_nucleos"`
	Discos         []DiscoLogico   `json:"discos"`
	DiscosFisicos  []DiscoFisico   `json:"discos_fisicos"`
	TopCPU         []ProcInfo      `json:"top_cpu"`
	TopRAM         []ProcInfo      `json:"top_ram"`
	ServiciosCaidos []ServicioInfo `json:"servicios_auto_caidos"`
	EventosError   []EventoInfo    `json:"eventos_error"`
	TempGB         float64         `json:"temp_gb"`
	Bateria        *BateriaInfo    `json:"bateria"`
	IPs            []string        `json:"ips"`
	Inicio         []string        `json:"inicio"`
	ReinicioPendiente bool         `json:"reinicio_pendiente"`
}

func (h *Hands) diagnosticar() (*Diagnostico, error) {
	out, err := ejecutarPS(scriptDiagnostico)
	if err != nil || out == "" {
		return nil, fmt.Errorf("no se pudo ejecutar el diagnóstico: %v", err)
	}
	var d Diagnostico
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, fmt.Errorf("no se pudo interpretar el diagnóstico: %v", err)
	}
	return &d, nil
}

// Diagnostico expone el motor de diagnóstico para la web.
func (h *Hands) Diagnostico() (*Diagnostico, error) {
	return h.diagnosticar()
}

// AnalizarSalud calcula el puntaje y la lista de problemas de un diagnóstico.
func AnalizarSalud(d *Diagnostico) (int, []Problema) {
	return analizarSalud(d)
}

func (d *Diagnostico) reporte() string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== INFORME DE INGENIERÍA ===\n")
	fmt.Fprintf(&b, "Sistema: %s %s\n", d.OS, d.Version)
	fmt.Fprintf(&b, "CPU: %s (%d núcleos) — uso actual %d%%\n", strings.TrimSpace(d.CPUNombre), d.CPUNucleos, d.CPUPorcentaje)
	fmt.Fprintf(&b, "Memoria: %d%% usado (%.1f / %.1f GB)\n", d.RAMPorcentaje, d.RAMUsadaGB, d.RAMTotalGB)
	fmt.Fprintf(&b, "Tiempo activo: %.1f días\n", d.UptimeDias)
	for _, disco := range d.Discos {
		fmt.Fprintf(&b, "Disco %s: %d%% usado (%.1f GB libre de %.1f GB)\n", disco.Letra, disco.Porcentaje, disco.LibreGB, disco.TotalGB)
	}
	for _, f := range d.DiscosFisicos {
		fmt.Fprintf(&b, "Disco físico: %s — salud %s\n", f.Nombre, f.Salud)
	}
	fmt.Fprintf(&b, "Temporales: %.2f GB | Reinicio pendiente: %v\n", d.TempGB, d.ReinicioPendiente)
	fmt.Fprintf(&b, "IPs: %s\n", strings.Join(d.IPs, ", "))
	fmt.Fprintf(&b, "\n--- TOP PROCESOS POR CPU ---\n")
	for _, p := range d.TopCPU {
		fmt.Fprintf(&b, "  %-20s %8ds  pid %d\n", p.Nombre, int(p.CPUSeg), p.PID)
	}
	fmt.Fprintf(&b, "\n--- TOP PROCESOS POR MEMORIA ---\n")
	for _, p := range d.TopRAM {
		fmt.Fprintf(&b, "  %-20s %6d MB  pid %d\n", p.Nombre, p.MB, p.PID)
	}
	if len(d.ServiciosCaidos) > 0 {
		fmt.Fprintf(&b, "\n--- SERVICIOS AUTOMÁTICOS DETENIDOS ---\n")
		for _, s := range d.ServiciosCaidos {
			fmt.Fprintf(&b, "  %-30s %s\n", s.Nombre, s.Estado)
		}
	}
	if len(d.EventosError) > 0 {
		fmt.Fprintf(&b, "\n--- ERRORES RECIENTES DEL SISTEMA ---\n")
		for _, e := range d.EventosError {
			fmt.Fprintf(&b, "  [%s] %s: %s\n", e.Hora, e.Fuente, e.Mensaje)
		}
	}
	if len(d.Inicio) > 0 {
		fmt.Fprintf(&b, "\n--- PROGRAMAS DE INICIO ---\n")
		for _, i := range d.Inicio {
			fmt.Fprintf(&b, "  - %s\n", i)
		}
	}
	return b.String()
}

func (h *Hands) diagnosticoCompleto() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	fmt.Println(d.reporte())
	return "Diagnóstico completo, señor. Vea el informe en la consola."
}

type Problema struct {
	Gravedad int
	Texto    string
	Fix      string
}

func analizarSalud(d *Diagnostico) (int, []Problema) {
	puntaje := 100
	var problemas []Problema
	agregar := func(g int, texto, fix string) {
		puntaje -= g
		problemas = append(problemas, Problema{Gravedad: g, Texto: texto, Fix: fix})
	}
	if d.RAMPorcentaje > 90 {
		agregar(15, fmt.Sprintf("Memoria al %d%%.", d.RAMPorcentaje), "Cierre aplicaciones pesadas o reinicie la PC.")
	} else if d.RAMPorcentaje > 80 {
		agregar(5, fmt.Sprintf("Memoria elevada: %d%%.", d.RAMPorcentaje), "Vigile qué procesos consumen RAM con 'top procesos'.")
	}
	for _, disco := range d.Discos {
		if disco.TotalGB <= 0 {
			continue
		}
		libre := disco.LibreGB / disco.TotalGB * 100
		if libre < 10 {
			agregar(15, fmt.Sprintf("Disco %s con solo %.0f%% libre.", disco.Letra, libre), "Ejecute 'mantenimiento' o analice con 'qué ocupa espacio'.")
		} else if libre < 20 {
			agregar(5, fmt.Sprintf("Disco %s con %.0f%% libre.", disco.Letra, libre), "Considere liberar espacio pronto.")
		}
	}
	for _, f := range d.DiscosFisicos {
		s := strings.ToLower(f.Salud)
		if s != "ok" && s != "healthy" && s != "healty" {
			agregar(20, fmt.Sprintf("Disco físico '%s' con salud '%s'.", f.Nombre, f.Salud), "Respaldé sus datos y revise el disco (smartctl o CrystalDiskInfo).")
		}
	}
	if len(d.ServiciosCaidos) > 0 {
		agregar(10, fmt.Sprintf("%d servicios automáticos están detenidos.", len(d.ServiciosCaidos)), "Vea el detalle con 'servicios caídos' e inicie el que corresponda.")
	}
	if n := len(d.EventosError); n > 0 {
		penalizacion := n * 5
		if penalizacion > 15 {
			penalizacion = 15
		}
		agregar(penalizacion, fmt.Sprintf("%d errores del sistema en las últimas 24 h.", n), "Vea el detalle con 'eventos de error recientes'.")
	}
	if d.TempGB > 2 {
		agregar(5, fmt.Sprintf("Carpeta temporal con %.2f GB.", d.TempGB), "Ejecute 'mantenimiento' para limpiarla.")
	}
	if d.ReinicioPendiente {
		agregar(5, "Reinicio pendiente del sistema.", "Reinicie la PC para aplicar cambios.")
	}
	if d.CPUPorcentaje > 90 {
		agregar(10, fmt.Sprintf("CPU al %d%%.", d.CPUPorcentaje), "Revise qué proceso la está usando con 'top procesos'.")
	}
	if d.Bateria != nil && d.Bateria.Porcentaje < 20 && !d.Bateria.Conectada {
		agregar(5, fmt.Sprintf("Batería al %d%% sin cargar.", d.Bateria.Porcentaje), "Conecte el cargador.")
	}
	if puntaje < 0 {
		puntaje = 0
	}
	return puntaje, problemas
}

func (h *Hands) saludSistema() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	puntaje, problemas := analizarSalud(d)
	fmt.Printf("\n=== SALUD DEL SISTEMA: %d/100 ===\n", puntaje)
	if len(problemas) == 0 {
		fmt.Println("Sin problemas detectados. Excelente estado, señor.")
	} else {
		for i, p := range problemas {
			fmt.Printf("[%d] %s\n    Sugerencia: %s\n", i+1, p.Texto, p.Fix)
		}
	}
	switch {
	case puntaje >= 90:
		return fmt.Sprintf("Su PC está en excelente estado, señor: %d de 100.", puntaje)
	case puntaje >= 75:
		return fmt.Sprintf("Su PC está en buen estado, señor: %d de 100, con %d detalle menor.", puntaje, len(problemas))
	case puntaje >= 60:
		return fmt.Sprintf("Su PC está regular, señor: %d de 100. Detecté %d problemas. Diga 'listar problemas' para verlos.", puntaje, len(problemas))
	default:
		return fmt.Sprintf("Su PC necesita atención, señor: %d de 100. %d problemas. Diga 'listar problemas' para verlos.", puntaje, len(problemas))
	}
}

func (h *Hands) problemasSistema() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	_, problemas := analizarSalud(d)
	if len(problemas) == 0 {
		return "Sin problemas detectados, señor. Su PC está impecable."
	}
	fmt.Println("=== PROBLEMAS DETECTADOS ===")
	for i, p := range problemas {
		fmt.Printf("[%d] %s\n    Sugerencia: %s\n", i+1, p.Texto, p.Fix)
	}
	return fmt.Sprintf("Detecté %d problemas, señor. Vea el detalle y las soluciones en la consola.", len(problemas))
}

func (h *Hands) serviciosCaidos() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	if len(d.ServiciosCaidos) == 0 {
		return "Todos los servicios automáticos están corriendo, señor."
	}
	fmt.Println("=== SERVICIOS AUTOMÁTICOS DETENIDOS ===")
	for _, s := range d.ServiciosCaidos {
		fmt.Printf(" - %s (%s)\n", s.Nombre, s.Estado)
	}
	return fmt.Sprintf("%d servicios automáticos detenidos, señor. Vea la lista en la consola.", len(d.ServiciosCaidos))
}

func (h *Hands) topProcesos() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	fmt.Println("=== TOP PROCESOS POR CPU ===")
	for _, p := range d.TopCPU {
		fmt.Printf(" - %-20s %6ds  pid %d\n", p.Nombre, int(p.CPUSeg), p.PID)
	}
	fmt.Println("=== TOP PROCESOS POR MEMORIA ===")
	for _, p := range d.TopRAM {
		fmt.Printf(" - %-20s %6d MB  pid %d\n", p.Nombre, p.MB, p.PID)
	}
	return "Top de procesos, señor. Vea la lista en la consola."
}

func (h *Hands) eventosRecientes() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	if len(d.EventosError) == 0 {
		return "Sin errores del sistema en las últimas 24 horas, señor."
	}
	fmt.Println("=== ERRORES RECIENTES DEL SISTEMA (24 h) ===")
	for _, e := range d.EventosError {
		fmt.Printf("[%s] %s: %s\n", e.Hora, e.Fuente, e.Mensaje)
	}
	return fmt.Sprintf("%d errores del sistema en las últimas 24 horas, señor. Vea el detalle en la consola.", len(d.EventosError))
}

func (h *Hands) programasInicio() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	if len(d.Inicio) == 0 {
		return "No encontré programas de inicio registrados, señor."
	}
	fmt.Println("=== PROGRAMAS DE INICIO ===")
	for _, i := range d.Inicio {
		fmt.Printf(" - %s\n", i)
	}
	return fmt.Sprintf("%d programas inician con Windows, señor. Vea la lista en la consola.", len(d.Inicio))
}

func (h *Hands) mantenimientoRapido() string {
	fmt.Println("[MANTENIMIENTO] Iniciando limpieza general...")
	liberado := limpiarTemporalesTexto()
	h.vaciarPapelera()
	h.limpiarDNS()
	fmt.Printf("[MANTENIMIENTO] Completado. Espacio liberado: %s\n", liberado)
	return "Mantenimiento completado, señor: temporales, papelera y DNS limpios."
}

func limpiarTemporalesTexto() string {
	script := `
$ErrorActionPreference='SilentlyContinue'
$antes=(Get-ChildItem ([System.IO.Path]::GetTempPath()) -Recurse -Force -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum
Get-ChildItem ([System.IO.Path]::GetTempPath()) -Force -ErrorAction SilentlyContinue | ForEach-Object { Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }
Get-ChildItem "$env:WINDIR\Temp" -Force -ErrorAction SilentlyContinue | ForEach-Object { Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }
Get-ChildItem "$env:LOCALAPPDATA\Microsoft\Windows\Explorer" -Filter "thumbcache_*" -Force -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
if(-not $antes){$antes=0}
"{0:N2}" -f ($antes/1MB)
`
	out, err := ejecutarPS(script)
	if err != nil {
		return "desconocido"
	}
	return out + " MB"
}

func (h *Hands) verificarIntegridad() string {
	fmt.Println("[INTEGRIDAD] Verificando archivos del sistema. Puede tardar unos minutos...")
	out, err := ejecutarPS("& sfc /verifyonly 2>&1 | Out-String")
	if err != nil {
		return "No pude ejecutar la verificación de integridad, señor."
	}
	fmt.Println(out)
	minus := strings.ToLower(out)
	if strings.Contains(minus, "administrador") || strings.Contains(minus, "administrator") {
		return "La verificación de integridad requiere permisos de administrador, señor. Ejecute Jarvis como administrador."
	}
	if strings.Contains(minus, "no encontró") || strings.Contains(minus, "not find") || strings.Contains(minus, "no violaciones") {
		return "Integridad verificada, señor: no se encontraron violaciones."
	}
	return "Verificación de integridad finalizada, señor. Vea el resultado en la consola."
}

func (h *Hands) carpetasGrandes() string {
	fmt.Println("=== ANÁLISIS DE ALMACENAMIENTO ===")
	var total int64
	for _, raiz := range carpetasUsuario() {
		if info, e := os.Stat(raiz); e != nil || !info.IsDir() {
			continue
		}
		tam := tamanoCarpeta(raiz)
		total += tam
		fmt.Printf(" - %-40s %6.1f MB\n", filepath.Base(raiz), float64(tam)/1024/1024)
	}
	fmt.Printf("Total en carpetas de usuario: %.1f GB\n", float64(total)/1024/1024/1024)
	return "Análisis de almacenamiento listo, señor. Vea el detalle en la consola."
}

func tamanoCarpeta(raiz string) int64 {
	var total int64
	filepath.Walk(raiz, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// === PLAN DE ACCIÓN ===

type Accion struct {
	Orden   int
	Tipo    string // auto | manual | aviso
	Titulo  string
	Detalle string
}

func generarPlan(d *Diagnostico, problemas []Problema) []Accion {
	var plan []Accion
	for _, p := range problemas {
		minus := strings.ToLower(p.Texto)
		switch {
		case strings.Contains(minus, "temporal"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "auto", Titulo: "Limpiar archivos temporales", Detalle: "Eliminar temporales del usuario y del sistema."})
		case strings.Contains(minus, "libre"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "auto", Titulo: "Liberar espacio en disco", Detalle: "Limpieza general de temporales, papelera y DNS."})
		case strings.Contains(minus, "reinicio"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "manual", Titulo: "Reiniciar la PC", Detalle: "Aplicar cambios pendientes del sistema."})
		case strings.Contains(minus, "memoria"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: "Reducir uso de memoria", Detalle: "Cerrar aplicaciones pesadas o reiniciar la PC."})
		case strings.Contains(minus, "servicio"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: "Revisar servicios detenidos", Detalle: "Ver detalle con 'servicios caídos' e iniciar los que correspondan."})
		case strings.Contains(minus, "cpu"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: "Revisar uso de CPU", Detalle: "Identificar el proceso pesado con 'top procesos'."})
		case strings.Contains(minus, "errores"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: "Revisar errores del sistema", Detalle: "Analizar el Visor de eventos con 'eventos de error recientes'."})
		case strings.Contains(minus, "bater"):
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: "Conectar el cargador", Detalle: "Batería baja sin carga."})
		default:
			plan = append(plan, Accion{Orden: p.Gravedad, Tipo: "aviso", Titulo: p.Texto, Detalle: p.Fix})
		}
	}
	sort.SliceStable(plan, func(i, j int) bool { return plan[i].Orden > plan[j].Orden })
	return plan
}

func (h *Hands) planAccion() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	puntaje, problemas := analizarSalud(d)
	acciones := generarPlan(d, problemas)
	fmt.Println("=== PLAN DE ACCIÓN ===")
	fmt.Printf("Salud actual: %d/100\n", puntaje)
	if len(acciones) == 0 {
		fmt.Println("Sin acciones necesarias.")
		return "No hay nada que corregir, señor. Su PC está impecable."
	}
	nAuto := 0
	for i, a := range acciones {
		etiqueta := "AVISO"
		switch a.Tipo {
		case "auto":
			etiqueta = "AUTO"
			nAuto++
		case "manual":
			etiqueta = "CONFIRMAR"
		}
		fmt.Printf("[%d][%s] %s — %s\n", i+1, etiqueta, a.Titulo, a.Detalle)
	}
	if nAuto > 0 {
		return fmt.Sprintf("Plan listo, señor: %d acciones. Puedo ejecutar %d automáticamente. Diga 'ejecutá el plan' cuando quiera.", len(acciones), nAuto)
	}
	return fmt.Sprintf("Plan listo, señor: %d acciones a considerar. Vea la consola.", len(acciones))
}

func (h *Hands) ejecutarPlan() string {
	d, err := h.diagnosticar()
	if err != nil {
		return err.Error() + ", señor."
	}
	_, problemas := analizarSalud(d)
	acciones := generarPlan(d, problemas)
	hechas := 0
	for _, a := range acciones {
		if a.Tipo != "auto" {
			continue
		}
		switch {
		case strings.Contains(a.Titulo, "temporal") || strings.Contains(a.Titulo, "Liberar"):
			fmt.Printf("[PLAN] Ejecutando: %s...\n", a.Titulo)
			liberado := limpiarTemporalesTexto()
			h.vaciarPapelera()
			h.limpiarDNS()
			fmt.Printf("[PLAN] Completado: %s (%s MB liberados).\n", a.Titulo, liberado)
			hechas++
		}
	}
	if hechas == 0 {
		return "No hay acciones automáticas que ejecutar, señor. El plan solo contiene pasos que requieren su decisión."
	}
	return fmt.Sprintf("Plan ejecutado, señor: %d acciones automáticas completadas. Su PC quedó optimizada.", hechas)
}

// === MODO VIGILANTE ===

func (h *Hands) modoVigilante() string {
	h.vigilanciaMu.Lock()
	if h.vigilanciaActiva {
		h.vigilanciaMu.Unlock()
		return "Ya estoy en modo vigilante, señor. Monitoreo cada 5 minutos."
	}
	h.vigilanciaStop = make(chan struct{})
	h.vigilanciaActiva = true
	h.vigilanciaAlertas = make(map[string]time.Time)
	h.vigilanciaMu.Unlock()
	go h.cicloVigilante()
	return "Modo vigilante activado, señor. Monitorearé la PC cada 5 minutos y le avisaré ante cualquier anomalía."
}

func (h *Hands) cicloVigilante() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	h.vigilarAhora()
	for {
		select {
		case <-h.vigilanciaStop:
			return
		case <-ticker.C:
			h.vigilarAhora()
		}
	}
}

func (h *Hands) vigilarAhora() {
	d, err := h.diagnosticar()
	if err != nil {
		return
	}
	if d.RAMPorcentaje > 90 {
		h.alertaVigilante("ram", fmt.Sprintf("ALERTA: memoria al %d%%, señor.", d.RAMPorcentaje))
	}
	for _, disco := range d.Discos {
		if disco.TotalGB <= 0 {
			continue
		}
		libre := disco.LibreGB / disco.TotalGB * 100
		if libre < 10 {
			h.alertaVigilante("disco_"+disco.Letra, fmt.Sprintf("ALERTA: el disco %s tiene solo %.0f%% libre, señor.", disco.Letra, libre))
		}
	}
	if d.CPUPorcentaje > 95 {
		h.alertaVigilante("cpu", fmt.Sprintf("ALERTA: CPU al %d%%, señor.", d.CPUPorcentaje))
	}
	if d.Bateria != nil && d.Bateria.Porcentaje < 20 && !d.Bateria.Conectada {
		h.alertaVigilante("bateria", fmt.Sprintf("ALERTA: batería al %d%% sin cargar, señor.", d.Bateria.Porcentaje))
	}
	if d.ReinicioPendiente {
		h.alertaVigilante("reinicio", "Tiene un reinicio pendiente, señor. Conviene reiniciar la PC.")
	}
	if len(d.ServiciosCaidos) > 0 {
		h.alertaVigilante("servicios", fmt.Sprintf("ALERTA: %d servicios automáticos están detenidos, señor.", len(d.ServiciosCaidos)))
	}
}

func (h *Hands) alertaVigilante(clave, texto string) {
	h.vigilanciaMu.Lock()
	ultima, vista := h.vigilanciaAlertas[clave]
	h.vigilanciaMu.Unlock()
	if vista && time.Since(ultima) < time.Hour {
		return
	}
	h.vigilanciaMu.Lock()
	h.vigilanciaAlertas[clave] = time.Now()
	h.vigilanciaMu.Unlock()
	fmt.Println("[VIGILANTE] " + texto)
	go h.enviarNotificacion(texto)
}

func (h *Hands) salirVigilancia() string {
	h.vigilanciaMu.Lock()
	defer h.vigilanciaMu.Unlock()
	if !h.vigilanciaActiva {
		return "No estoy en modo vigilante, señor."
	}
	close(h.vigilanciaStop)
	h.vigilanciaActiva = false
	return "Modo vigilante desactivado, señor. Descansaré un momento."
}

func (h *Hands) estadoVigilancia() string {
	h.vigilanciaMu.Lock()
	defer h.vigilanciaMu.Unlock()
	if h.vigilanciaActiva {
		return "Modo vigilante activo, señor. Reviso la PC cada 5 minutos."
	}
	return "Modo vigilante inactivo, señor. Diga 'modo vigilante' para activarlo."
}
