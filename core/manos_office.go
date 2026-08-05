package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// === OFFICE (F3): crear documentos de Word/Excel/PowerPoint por COM ===
// Se usa PowerShell con objetos COM (igual que el resto de las herramientas
// de Windows del proyecto). Sin librerías externas (no hay red en esta
// máquina). Los scripts guardan el documento en WorkspaceRoot.

// tipoOffice identifica la aplicación pedida en el comando.
func tipoOffice(cmd string) string {
	switch {
	case strings.Contains(cmd, "word") || strings.Contains(cmd, "documento"):
		return "word"
	case strings.Contains(cmd, "excel") || strings.Contains(cmd, "planilla") ||
		strings.Contains(cmd, "hoja de cálculo") || strings.Contains(cmd, "hoja de calculo"):
		return "excel"
	case strings.Contains(cmd, "powerpoint") || strings.Contains(cmd, "power point") ||
		strings.Contains(cmd, "presentación") || strings.Contains(cmd, "presentacion") ||
		strings.Contains(cmd, "diapositiva") || strings.Contains(cmd, "ppt"):
		return "powerpoint"
	}
	return ""
}

// extensionesOffice por tipo de documento.
var extensionesOffice = map[string]string{
	"word":       ".docx",
	"excel":      ".xlsx",
	"powerpoint": ".pptx",
}

// manejarOffice crea un documento de Office desde un comando de voz.
func (h *Hands) manejarOffice(cmd string) string {
	tipo := tipoOffice(cmd)
	if tipo == "" {
		return "Puedo crear documentos de Word, planillas de Excel o presentaciones de PowerPoint, señor. Diga por ejemplo 'creá un documento word'."
	}

	nombre := extraerNombreArchivoOffice(cmd, tipo)
	if !esRutaSegura(nombre) {
		return fmt.Sprintf("El nombre '%s' no es válido, señor.", nombre)
	}
	ruta := filepath.Join(h.WorkspaceRoot, nombre)

	if err := h.crearDocumentoOffice(tipo, ruta); err != nil {
		return fmt.Sprintf("No pude crear el documento, señor. ¿Está instalado %s? Error: %v", appOffice(tipo), err)
	}
	return fmt.Sprintf("Documento de %s creado en %s, señor.", appOffice(tipo), ruta)
}

// appOffice devuelve el nombre legible de la aplicación.
func appOffice(tipo string) string {
	switch tipo {
	case "word":
		return "Word"
	case "excel":
		return "Excel"
	case "powerpoint":
		return "PowerPoint"
	}
	return tipo
}

// extraerNombreArchivoOffice saca el nombre del archivo tras "llamado / en /
// como". Por defecto usa "documento-jarvis.docx" según el tipo.
func extraerNombreArchivoOffice(cmd, tipo string) string {
	ext := extensionesOffice[tipo]
	for _, pref := range []string{" llamado ", "llamado ", " en ", "como "} {
		if idx := strings.Index(cmd, pref); idx >= 0 {
			nombre := strings.TrimSpace(cmd[idx+len(pref):])
			nombre = strings.TrimSuffix(nombre, ".")
			if nombre != "" {
				if !strings.HasSuffix(strings.ToLower(nombre), ext) {
					nombre += ext
				}
				return nombre
			}
		}
	}
	return "documento-jarvis" + ext
}

// crearDocumentoOffice ejecuta el script PowerShell COM que arma y guarda el
// documento. El nombre del archivo se valida con esRutaSegura antes.
func (h *Hands) crearDocumentoOffice(tipo, ruta string) error {
	script := scriptOffice(tipo, ruta)
	_, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	return err
}

// scriptOffice arma el código PowerShell. La ruta va entre comillas simples
// (escapando las internas) para que no haya interpolación.
func scriptOffice(tipo, ruta string) string {
	rutaPS := strings.ReplaceAll(ruta, "'", "''")
	barraNPS := "`" + "r`" + "n"
	fecha := time.Now().Format("02/01/2006")

	switch tipo {
	case "word":
		return fmt.Sprintf(`
$w = New-Object -ComObject Word.Application
$w.Visible = $false
$doc = $w.Documents.Add()
$doc.Content.Text = "Documento generado por JarvisOS%sFecha: %s%sRuta: %s"
$doc.SaveAs2('%s', 16)
$doc.Close()
$w.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($w) | Out-Null`, barraNPS, fecha, barraNPS, ruta, rutaPS)
	case "excel":
		return fmt.Sprintf(`
$e = New-Object -ComObject Excel.Application
$e.Visible = $false
$e.DisplayAlerts = $false
$wb = $e.Workbooks.Add()
$ws = $wb.Worksheets.Item(1)
$ws.Cells.Item(1,1) = "JarvisOS"
$ws.Cells.Item(2,1) = "Fecha"
$ws.Cells.Item(2,2) = '%s'
$ws.Cells.Item(3,1) = "Ruta"
$ws.Cells.Item(3,2) = '%s'
$wb.SaveAs('%s', 51)
$wb.Close()
$e.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($e) | Out-Null`, fecha, ruta, rutaPS)
	case "powerpoint":
		return fmt.Sprintf(`
$p = New-Object -ComObject PowerPoint.Application
$pres = $p.Presentations.Add()
$slide = $pres.Slides.Add(1, 1)
$slide.Shapes.Title.TextFrame.TextRange.Text = "JARVIS"
$slide.Shapes.Placeholders.Item(2).TextFrame.TextRange.Text = "Presentación generada por JarvisOS - %s"
$pres.SaveAs('%s', 24)
$pres.Close()
$p.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($p) | Out-Null`, fecha, rutaPS)
	}
	return ""
}
