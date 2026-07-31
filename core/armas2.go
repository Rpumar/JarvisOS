package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (h *Hands) limpiarTemporales() string {
	out, err := ejecutarPS(`
$dirs=@($env:TEMP,"C:\Windows\Temp")
$n=0
foreach($d in $dirs){
  if(Test-Path $d){
    Get-ChildItem $d -Force -ErrorAction SilentlyContinue | ForEach-Object { $n++; Remove-Item $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }
  }
}
"Archivos temporales eliminados: {0}" -f $n`)
	if err != nil {
		return "No pude limpiar los archivos temporales, señor."
	}
	fmt.Println(out)
	return out
}

func (h *Hands) organizarDescargas() string {
	out, err := ejecutarPS(`
$dl=Join-Path $env:USERPROFILE 'Downloads'
if(-not(Test-Path $dl)){'no existe';exit}
$n=0
Get-ChildItem $dl -File -ErrorAction SilentlyContinue | Where-Object {$_.Extension} | ForEach-Object {
  $cat=$_.Extension.TrimStart('.').ToLower()
  $dest=Join-Path $dl $cat
  if(-not(Test-Path $dest)){New-Item -ItemType Directory -Path $dest -Force | Out-Null}
  Move-Item $_.FullName -Destination $dest -Force -ErrorAction SilentlyContinue
  $n++
}
"Archivos organizados en Descargas por tipo: {0}" -f $n`)
	if err != nil {
		return "No pude organizar las descargas, señor."
	}
	fmt.Println(out)
	return out
}

func (h *Hands) grabarPantalla() string {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "Necesito ffmpeg instalado para grabar la pantalla, señor. ¿Quiere que le explique cómo instalarlo?"
	}
	dir := filepath.Join(os.Getenv("USERPROFILE"), "Videos")
	os.MkdirAll(dir, 0o755)
	archivo := fmt.Sprintf("jarvis-captura-%s.mp4", time.Now().Format("20060102-150405"))
	destino := filepath.Join(dir, archivo)
	cmd := exec.Command("cmd", "/C", "start", "", "ffmpeg", "-y", "-f", "gdigrab", "-framerate", "30", "-i", "desktop", "-t", "30", destino)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("No pude iniciar la grabación, señor: %v", err)
	}
	return fmt.Sprintf("Grabando la pantalla por 30 segundos, señor. Se guardará en %s.", destino)
}

func (h *Hands) modoOscuro() string {
	out, err := ejecutarPS(`
$r='HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize'
$v=(Get-ItemProperty -Path $r -ErrorAction SilentlyContinue).AppsUseLightTheme
if($v -eq 1){
  Set-ItemProperty -Path $r -Name AppsUseLightTheme -Value 0
  Set-ItemProperty -Path $r -Name SystemUsesLightTheme -Value 0
  'oscuro'
}else{
  Set-ItemProperty -Path $r -Name AppsUseLightTheme -Value 1
  Set-ItemProperty -Path $r -Name SystemUsesLightTheme -Value 1
  'claro'
}`)
	if err != nil || out == "" {
		return "No pude cambiar el modo oscuro, señor."
	}
	modo := "claro"
	if strings.TrimSpace(out) == "oscuro" {
		modo = "oscuro"
	}
	return fmt.Sprintf("Tema cambiado a modo %s, señor.", modo)
}

func (h *Hands) firewallEstado() string {
	out, err := ejecutarPS(`
Get-NetFirewallProfile | ForEach-Object { "{0}: {1}" -f $_.Name, $(if($_.Enabled){'activo'}else{'inactivo'}) } | Out-String`)
	if err != nil || out == "" {
		return "No pude consultar el firewall, señor."
	}
	fmt.Println(out)
	return "Estado del firewall consultado, señor. Mire la consola para ver los perfiles."
}

func (h *Hands) puertosEnUso() string {
	out, err := ejecutarPS(`
Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
  Select-Object -Unique LocalPort,OwningProcess |
  ForEach-Object {
    $p=(Get-Process -Id $_.OwningProcess -ErrorAction SilentlyContinue).ProcessName
    "Puerto {0}: {1}" -f $_.LocalPort,$p
  } | Sort-Object | Out-String`)
	if err != nil || out == "" {
		return "No pude listar los puertos en uso, señor."
	}
	fmt.Println(out)
	return "Puertos en uso listados, señor. Mire la consola para verlos."
}

func (h *Hands) procesosConRed() string {
	out, err := ejecutarPS(`
Get-NetTCPConnection -State Established -ErrorAction SilentlyContinue |
  ForEach-Object { Get-Process -Id $_.OwningProcess -ErrorAction SilentlyContinue } |
  Group-Object ProcessName |
  Sort-Object Count -Descending |
  Select-Object -First 10 Count,Name |
  Format-Table -AutoSize | Out-String`)
	if err != nil || out == "" {
		return "No pude ver qué procesos usan la red, señor."
	}
	fmt.Println(out)
	return "Procesos que usan la red listados, señor. Mire la consola."
}

func (h *Hands) sesionesActivas() string {
	out, err := ejecutarPS("query user")
	if err != nil {
		return "No pude listar las sesiones activas, señor."
	}
	fmt.Println(out)
	return "Sesiones activas listadas, señor. Mire la consola."
}
