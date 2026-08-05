package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// === PDF (F3): dos vías sin librerías externas ===
//  1. Generador de PDF puro en Go (texto plano, notas): construye el archivo
//     a mano con el formato PDF mínimo (catálogo, páginas, stream de texto,
//     xref). Sirve para "creá un pdf con ..." / "exportá mis notas a pdf".
//  2. Conversión Office -> PDF por COM (Word/Excel/PowerPoint): abre el
//     documento existente y lo exporta a PDF usando el formato nativo de
//     cada aplicación. Comando: "convertí informe.docx a pdf".

// manejarPDF despacha las operaciones de PDF.
func (h *Hands) manejarPDF(cmd string) string {
	lower := strings.ToLower(cmd)

	// Exportar las notas a PDF.
	if strings.Contains(lower, "notas") && (strings.Contains(lower, "pdf") ||
		strings.Contains(lower, "export")) {
		return h.notasAPDF()
	}

	// Convertir un documento de Office existente a PDF.
	if strings.Contains(lower, "a pdf") && (strings.Contains(lower, "convert") ||
		strings.Contains(lower, "pas") || strings.Contains(lower, "mand")) {
		return h.convertirOfficeAPDF(cmd)
	}

	return "Puedo crear un PDF con texto ('creá un pdf con ...'), exportar mis notas a PDF, o convertir un documento de Word/Excel/PowerPoint a PDF ('convertí informe.docx a pdf'), señor."
}

// --- Generador de PDF puro en Go ---

// generarPDFEscribe construye un PDF de una página con las líneas dadas.
// El título va primero; el resto en Helvetica.
func generarPDFEscribe(ruta string, titulo string, lineas []string) error {
	if titulo != "" {
		lineas = append([]string{titulo, ""}, lineas...)
	}
	pdf := construirPDF(lineas)
	return os.WriteFile(ruta, pdf, 0o600)
}

// construirPDF arma el array de bytes de un PDF válido.
func construirPDF(lineas []string) []byte {
	const ( // A4: 595 x 842 puntos
		alto      = 842
		ancho     = 595
		espaciado = 16
		margenIzq = 60
	)

	// Paginar: ~45 líneas útiles por página a 12pt con espaciado 16.
	porPagina := (alto - 120) / espaciado
	var paginas [][]string
	for i := 0; i < len(lineas); i += porPagina {
		fin := i + porPagina
		if fin > len(lineas) {
			fin = len(lineas)
		}
		paginas = append(paginas, lineas[i:fin])
	}
	if len(paginas) == 0 {
		paginas = [][]string{{" "}}
	}

	// Numeración de objetos:
	//   1 catálogo, 2 páginas, 3 fuente, y por página: 4+2i (página), 5+2i (contenido).
	catalog := "<< /Type /Catalog /Pages 2 0 R >>"
	nPag := len(paginas)
	kids := make([]string, 0, nPag)
	for i := 0; i < nPag; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 4+2*i))
	}
	pages := fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), nPag)

	total := 3 + nPag*2
	cuerpo := make([][]byte, total+1) // índice 1-based
	cuerpo[1] = []byte(fmt.Sprintf("1 0 obj\n%s\nendobj\n", catalog))
	cuerpo[2] = []byte(fmt.Sprintf("2 0 obj\n%s\nendobj\n", pages))
	cuerpo[3] = []byte("3 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")
	streams := make([]string, nPag)
	for i, p := range paginas {
		pageID := 4 + 2*i
		contID := 5 + 2*i
		streams[i] = streamTexto(p, margenIzq, alto-80, espaciado)
		cuerpo[pageID] = []byte(fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>\nendobj\n",
			pageID, ancho, alto, contID))
		cuerpo[contID] = []byte(fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", contID, len(streams[i]), streams[i]))
	}

	// Calcular offsets del xref.
	offsets := make([]int, total+1)
	var out strings.Builder
	out.WriteString("%PDF-1.4\n")
	for id := 1; id <= total; id++ {
		offsets[id] = out.Len()
		out.Write(cuerpo[id])
	}
	xrefOffset := out.Len()
	out.WriteString("xref\n")
	out.WriteString(fmt.Sprintf("0 %d\n", total+1))
	out.WriteString("0000000000 65535 f \n")
	for id := 1; id <= total; id++ {
		out.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[id]))
	}
	out.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\n", total+1))
	out.WriteString("startxref\n")
	out.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	out.WriteString("%%EOF\n")
	return []byte(out.String())
}

// streamTexto convierte líneas de texto en el operador de texto del PDF,
// escapando paréntesis y codificando a WinAnsi (Latin-1 para el español).
func streamTexto(lineas []string, x, y, espaciado int) string {
	var b strings.Builder
	b.WriteString("BT\n/F1 12 Tf\n")
	b.WriteString(fmt.Sprintf("1 0 0 1 %d %d Tm\n", x, y))
	for _, linea := range lineas {
		b.WriteString("(")
		b.WriteString(escaparPDF(linea))
		b.WriteString(") Tj\n")
		b.WriteString(fmt.Sprintf("0 -%d Td\n", espaciado))
	}
	b.WriteString("ET")
	return b.String()
}

// escaparPDF codifica texto UTF-8 a WinAnsi y escapa los caracteres
// especiales del literal de cadena del PDF.
func escaparPDF(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '(':
			b.WriteString("\\(")
		case ')':
			b.WriteString("\\)")
		case '\\':
			b.WriteString("\\\\")
		default:
			if bto, ok := winAnsi(r); ok {
				b.WriteByte(bto)
			}
		}
	}
	return b.String()
}

// winAnsi mapea caracteres con acento del español a la codificación WinAnsi
// (que coincide con Latin-1 para el rango usado). ASCII se mantiene igual.
func winAnsi(r rune) (byte, bool) {
	if r >= 0x20 && r <= 0x7E {
		return byte(r), true
	}
	switch r {
	case 'á':
		return 0xE1, true
	case 'é':
		return 0xE9, true
	case 'í':
		return 0xED, true
	case 'ó':
		return 0xF3, true
	case 'ú':
		return 0xFA, true
	case 'ñ':
		return 0xF1, true
	case 'ü':
		return 0xFC, true
	case 'Á':
		return 0xC1, true
	case 'É':
		return 0xC9, true
	case 'Í':
		return 0xCD, true
	case 'Ó':
		return 0xD3, true
	case 'Ú':
		return 0xDA, true
	case 'Ñ':
		return 0xD1, true
	case '¿':
		return 0xBF, true
	case '¡':
		return 0xA1, true
	}
	return 0, false
}

// --- Conversión Office -> PDF por COM ---

// convertirOfficeAPDF encuentra el documento pedido en WorkspaceRoot y lo
// exporta a PDF con la aplicación COM correspondiente.
func (h *Hands) convertirOfficeAPDF(cmd string) string {
	nombre := extraerNombrePDF(cmd)
	if nombre == "" {
		return "¿Qué documento quiere convertir a PDF, señor? Diga por ejemplo 'convertí informe.docx a pdf'."
	}
	rutaOrig, tipo := localizarDocumento(h.WorkspaceRoot, nombre)
	if rutaOrig == "" {
		return fmt.Sprintf("No encontré '%s' en el workspace, señor.", nombre)
	}
	rutaPDF := strings.TrimSuffix(rutaOrig, filepath.Ext(rutaOrig)) + ".pdf"
	script := scriptConvertirAPDF(tipo, rutaOrig, rutaPDF)
	if script == "" {
		return fmt.Sprintf("No sé convertir el tipo '%s' a PDF, señor.", tipo)
	}
	if _, err := h.ejecutarConTimeout("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script); err != nil {
		return fmt.Sprintf("No pude convertir el documento a PDF, señor. ¿Está instalado %s? Error: %v", tipo, err)
	}
	return fmt.Sprintf("Convertido a PDF: %s, señor.", rutaPDF)
}

// extraerNombrePDF saca el nombre del archivo del comando
// "convertí <nombre> a pdf". Limpia artículos y etiquetas genéricas.
func extraerNombrePDF(cmd string) string {
	lower := strings.ToLower(cmd)
	for _, pref := range []string{"convertí el ", "convierteme el ", "convertime el ", "convertí ", "convierteme ", "convertime ", "pasá el ", "pasame el ", "pasá "} {
		if strings.HasPrefix(lower, pref) {
			resto := strings.TrimSpace(cmd[len(pref):])
			if idx := strings.Index(resto, " a pdf"); idx >= 0 {
				resto = strings.TrimSpace(resto[:idx])
			} else if strings.HasSuffix(resto, " a pdf") {
				resto = strings.TrimSpace(strings.TrimSuffix(resto, " a pdf"))
			}
			resto = strings.TrimSpace(strings.TrimPrefix(resto, "documento "))
			resto = strings.TrimSpace(strings.TrimPrefix(resto, "archivo "))
			resto = strings.TrimSpace(strings.TrimPrefix(resto, "el "))
			resto = strings.TrimSpace(strings.TrimPrefix(resto, "la "))
			resto = strings.TrimSpace(strings.Trim(resto, " \",.'"))
			switch strings.ToLower(resto) {
			case "documento", "archivo", "el", "la", "doc":
				return ""
			}
			return resto
		}
	}
	return ""
}

// localizarDocumento busca el archivo por nombre (con o sin extensión) en
// WorkspaceRoot. Devuelve la ruta y el tipo COM ("word"/"excel"/"powerpoint").
func localizarDocumento(raiz, nombre string) (string, string) {
	nombreLower := strings.ToLower(nombre)
	extensiones := map[string]string{
		".docx": "word", ".doc": "word",
		".xlsx": "excel", ".xls": "excel",
		".pptx": "powerpoint", ".ppt": "powerpoint",
	}
	// Búsqueda directa en el workspace.
	ext := strings.ToLower(filepath.Ext(nombreLower))
	if tipo, ok := extensiones[ext]; ok {
		ruta := filepath.Join(raiz, nombre)
		if info, err := os.Stat(ruta); err == nil && !info.IsDir() {
			return ruta, tipo
		}
	}
	// Buscar por nombre sin extensión.
	entradas, err := os.ReadDir(raiz)
	if err == nil {
		base := strings.TrimSuffix(nombreLower, ext)
		for _, e := range entradas {
			if e.IsDir() {
				continue
			}
			nameLower := strings.ToLower(e.Name())
			if tipo, ok := extensiones[filepath.Ext(nameLower)]; ok && strings.TrimSuffix(nameLower, filepath.Ext(nameLower)) == base {
				return filepath.Join(raiz, e.Name()), tipo
			}
		}
	}
	return "", ""
}

// scriptConvertirAPDF arma el PowerShell COM que exporta a PDF según el tipo.
// Word: SaveAs2(...,17=wdFormatPDF). Excel: ExportAsFixedFormat(0=xlTypePDF).
// PowerPoint: SaveAs(...,32=ppSaveAsPDF).
func scriptConvertirAPDF(tipo, rutaOrig, rutaPDF string) string {
	o := strings.ReplaceAll(rutaOrig, "'", "''")
	p := strings.ReplaceAll(rutaPDF, "'", "''")
	switch tipo {
	case "word":
		return fmt.Sprintf(`
$w = New-Object -ComObject Word.Application
$w.Visible = $false
$doc = $w.Documents.Open('%s')
$doc.SaveAs2('%s', 17)
$doc.Close(0)
$w.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($w) | Out-Null`, o, p)
	case "excel":
		return fmt.Sprintf(`
$e = New-Object -ComObject Excel.Application
$e.Visible = $false
$e.DisplayAlerts = $false
$wb = $e.Workbooks.Open('%s')
$wb.ExportAsFixedFormat(0, '%s')
$wb.Close(0)
$e.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($e) | Out-Null`, o, p)
	case "powerpoint":
		return fmt.Sprintf(`
$pp = New-Object -ComObject PowerPoint.Application
$pres = $pp.Presentations.Open('%s')
$pres.SaveAs('%s', 32)
$pres.Close()
$pp.Quit()
[System.Runtime.Interopservices.Marshal]::ReleaseComObject($pp) | Out-Null`, o, p)
	}
	return ""
}

// notasAPDF exporta las notas guardadas a un PDF en WorkspaceRoot.
func (h *Hands) notasAPDF() string {
	rutaNotas := filepath.Join(carpetaJarvisOS(), "notas.txt")
	contenido, err := os.ReadFile(rutaNotas)
	if err != nil {
		return "No tiene notas guardadas para exportar, señor."
	}
	lineas := strings.Split(strings.TrimSpace(string(contenido)), "\n")
	rutaSalida := filepath.Join(h.WorkspaceRoot, "notas-jarvis.pdf")
	if err := generarPDFEscribe(rutaSalida, "Mis notas de Jarvis", lineas); err != nil {
		return fmt.Sprintf("No pude generar el PDF: %v", err)
	}
	return fmt.Sprintf("Exporté sus %d notas a %s, señor.", len(lineas), rutaSalida)
}
