package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// === OUTLOOK (F3): sincronizar la agenda local con el calendario de ===
//     Outlook por COM. Se usa PowerShell con objetos COM, igual que el resto
//     de las herramientas de Windows del proyecto. Sin librerías externas.
//     Los eventos de la agenda local se crean como citas (appointments) en
//     Outlook y se marcan (`SincOutlook`) para no duplicarlos.

const folderCalendarioOutlook = 9 // olFolderCalendar

// manejarOutlook despacha las operaciones de Outlook: sincronizar la agenda
// local hacia el calendario de Outlook, o leer los próximos turnos.
func (h *Hands) manejarOutlook(cmd string) string {
	lower := strings.ToLower(cmd)

	if (strings.Contains(lower, "outlook") || strings.Contains(lower, "calendario")) &&
		(strings.Contains(lower, "sincroniz") || strings.Contains(lower, "export") ||
			strings.Contains(lower, "manda") || strings.Contains(lower, "mand") ||
			strings.Contains(lower, "mover")) &&
		(strings.Contains(lower, "agenda") || strings.Contains(lower, "evento") ||
			strings.Contains(lower, "todo") || strings.Contains(lower, "citas")) {
		return h.sincronizarOutlook()
	}

	if strings.Contains(lower, "outlook") &&
		(strings.Contains(lower, "le") || strings.Contains(lower, "leí") ||
			strings.Contains(lower, "proxim") || strings.Contains(lower, "siguiente") ||
			strings.Contains(lower, "qué tengo") || strings.Contains(lower, "que tengo")) {
		return h.leerOutlook()
	}

	return "Puedo sincronizar la agenda con Outlook ('sincronizá la agenda con outlook') o leer los próximos eventos de Outlook, señor."
}

// sincronizarOutlook exporta los eventos locales pendientes como citas en el
// calendario de Outlook y los marca como sincronizados.
func (h *Hands) sincronizarOutlook() string {
	if h.agenda == nil {
		return "No tengo agenda local configurada, señor."
	}
	pendientes := h.agenda.PendientesOutlook(50, time.Now())
	if len(pendientes) == 0 {
		return "No hay eventos pendientes por sincronizar, señor."
	}

	script := buildScriptSincronizarOutlook(pendientes)
	_, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	if err != nil {
		return fmt.Sprintf("No pude sincronizar con Outlook, señor. ¿Está instalado y configurado? Error: %v", err)
	}
	for _, e := range pendientes {
		h.agenda.MarcarSincOutlook(e.ID)
	}
	nombres := make([]string, 0, len(pendientes))
	for _, e := range pendientes {
		nombres = append(nombres, e.Titulo)
	}
	return fmt.Sprintf("Sincronicé %d evento(s) con el calendario de Outlook: %s, señor.", len(pendientes), strings.Join(nombres, ", "))
}

// buildScriptSincr arma el script PowerShell COM que crea las citas. La fecha
// de inicio se pasa como un timestamp de 13 dígitos (millis, como en COM) y
// cada título va entre comillas simples doble-escapadas.
func buildScriptSincronizarOutlook(eventos []EventoAgenda) string {
	var b strings.Builder
	b.WriteString("$ol = New-Object -ComObject Outlook.Application`n")
	b.WriteString("$f = $ol.Session.GetDefaultFolder(" + strconv.Itoa(folderCalendarioOutlook) + ")`n")

	for _, e := range eventos {
		inicio, err := time.Parse(time.RFC3339, e.Inicio)
		if err != nil {
			continue
		}
		// Como Outlook espera .Start como DateTime, se arma con [datetime] de
		// una cadena ISO local segura.
		inicioISO := inicio.Format("2006-01-02T15:04:05")
		titulo := strings.ReplaceAll(e.Titulo, "'", "''")
		b.WriteString("$ap = $f.Items.Add(1)`n") // olAppointmentItem
		b.WriteString("$ap.Subject = '" + titulo + "'`n")
		b.WriteString("$ap.Start = [datetime]'" + inicioISO + "'`n")
		if e.Fin != "" {
			fin, ferr := time.Parse(time.RFC3339, e.Fin)
			if ferr == nil {
				b.WriteString("$ap.End = [datetime]'" + fin.Format("2006-01-02T15:04:05") + "'`n")
			} else {
				b.WriteString("$ap.Duration = 60`n")
			}
		} else {
			b.WriteString("$ap.Duration = 60`n")
		}
		b.WriteString("$ap.Save() | Out-Null`n")
	}
	b.WriteString("[System.Runtime.Interopservices.Marshal]::ReleaseComObject($ol) | Out-Null`n")
	return b.String()
}

// leerOutlook devuelve los próximos N turnos del calendario de Outlook.
func (h *Hands) leerOutlook() string {
	script := leerOutlookScript(5)
	salida, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	if err != nil {
		return "No pude leer el calendario de Outlook, señor. Error: " + err.Error()
	}
	extractos := parseTurnosOutlook(string(salida))
	if len(extractos) == 0 {
		return "No encontré próximos turnos en el calendario de Outlook, señor."
	}
	return "Próximos en Outlook: " + strings.Join(extractos, " | ") + "."
}

// leerOutlookScript lista los próximos N turnos por COM, devolviendo una
// línea "Asunto|yyyy-MM-dd HH:mm" por cita. Se usa Restrict para filtrar por
// fecha (más rápido que recorrer toda la colección).
func leerOutlookScript(n int) string {
	var b strings.Builder
	b.WriteString("$ol = New-Object -ComObject Outlook.Application`n")
	b.WriteString("$cf = $ol.Session.GetDefaultFolder(" + strconv.Itoa(folderCalendarioOutlook) + ")`n")
	b.WriteString("$ahora = Get-Date`n")
	b.WriteString("$filter = 'Start >= \"' + $ahora.ToString('yyyy-MM-dd HH:mm') + '\"'`n")
	b.WriteString("$items = $cf.Items.Restrict($filter)`n")
	b.WriteString("$items.Sort('[Start]', 1)`n")
	b.WriteString("$cnt = 0`n")
	b.WriteString("foreach ($i in $items) {`n")
	b.WriteString("  Write-Output ($i.Subject + '|' + $i.Start.ToString('yyyy-MM-dd HH:mm'))`n")
	b.WriteString("  $cnt++`n")
	b.WriteString("  if ($cnt -ge " + strconv.Itoa(n) + ") { break }`n")
	b.WriteString("}`n")
	b.WriteString("[System.Runtime.Interopservices.Marshal]::ReleaseComObject($ol) | Out-Null`n")
	return b.String()
}

// parseTurnosOutlook interpreta la salida "Asunto|yyyy-MM-dd HH:mm" del script.
func parseTurnosOutlook(salida string) []string {
	var out []string
	for _, linea := range strings.Split(salida, "\n") {
		linea = strings.TrimSpace(linea)
		if linea == "" || linea == "DONE" {
			continue
		}
		idx := strings.LastIndex(linea, "|")
		if idx < 0 {
			continue
		}
		asunto := strings.TrimSpace(linea[:idx])
		fechaISO := strings.TrimSpace(linea[idx+1:])
		if asunto == "" || fechaISO == "" {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02 15:04", fechaISO, time.Local)
		if err != nil {
			continue
		}
		out = append(out, fmt.Sprintf("%s (%s)", asunto, t.Format("02/01 15:04")))
	}
	return out
}