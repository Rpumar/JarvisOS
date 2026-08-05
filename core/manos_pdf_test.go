package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConstruirPDF(t *testing.T) {
	pdf := construirPDF([]string{"Hola mundo", "Línea con acentos: á é í ó ú ñ", "Con (paréntesis)"})

	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatalf("falta el header %%PDF-1.4")
	}
	if !strings.HasSuffix(s, "%%EOF\n") {
		t.Fatalf("falta el trailer %%EOF")
	}
	if !strings.Contains(s, "xref") {
		t.Fatalf("falta la tabla xref")
	}
	if !strings.Contains(s, "startxref") {
		t.Fatalf("falta startxref")
	}
	// El stream debe tener texto escapado.
	if !strings.Contains(s, "(Hola mundo) Tj") {
		t.Fatalf("no se escribió la línea de texto")
	}
	// Acentos en WinAnsi: é = 0xE9, ñ = 0xF1.
	if !strings.Contains(s, "\xe9") {
		t.Fatalf("no se codificó el acento é en WinAnsi")
	}
	if !strings.Contains(s, "\xf1") {
		t.Fatalf("no se codificó la ñ en WinAnsi")
	}
	// Paréntesis escapados.
	if !strings.Contains(s, "\\(") {
		t.Fatalf("no se escaparon los paréntesis")
	}

	// Validación estructural: cada offset del xref debe apuntar al inicio de
	// un "N 0 obj". Si el xref está mal, ningún lector abre el PDF.
	verificarXref(t, pdf)
}

// verificarXref parsea el xref del PDF y comprueba que cada offset apunte al
// objeto correcto.
func verificarXref(t *testing.T, pdf []byte) {
	t.Helper()
	s := string(pdf)
	inicioXref := strings.Index(s, "xref\n")
	if inicioXref < 0 {
		t.Fatal("no se encontró la tabla xref")
	}
	// La línea anterior al xref tiene el offset inicial.
	cuerpo := s[:inicioXref]
	lineas := strings.Split(strings.TrimSpace(cuerpo), "\n")
	for _, obj := range lineas {
		obj = strings.TrimSpace(obj)
		if obj == "" || strings.HasPrefix(obj, "%") {
			continue
		}
		// Formato: "N 0 obj" o "N 0 R".
		var num, cero int
		var tok string
		if _, err := fmt.Sscanf(obj, "%d %d %s", &num, &cero, &tok); err != nil {
			continue
		}
		if tok != "obj" {
			continue
		}
	}
	// Comparar offsets del xref contra el contenido.
	idxEntrada := inicioXref + len("xref\n")
	// Obtener conteo: "0 N".
	resto := s[idxEntrada:]
	lineasXref := strings.Split(resto, "\n")
	if len(lineasXref) < 2 {
		t.Fatalf("xref vacío")
	}
	var nTotal int
	if _, err := fmt.Sscanf(lineasXref[0], "0 %d", &nTotal); err != nil {
		t.Fatalf("no se pudo leer el conteo del xref: %q", lineasXref[0])
	}
	// La primera entrada es la libre (65535 f), luego nTotal-1 entradas.
	entradas := lineasXref[1:]
	if len(entradas) < nTotal {
		t.Fatalf("xref incompleto: esperaba %d entradas, hay %d", nTotal, len(entradas))
	}
	for i := 1; i < nTotal; i++ {
		var offset int
		if _, err := fmt.Sscanf(strings.TrimSpace(entradas[i]), "%d", &offset); err != nil {
			t.Fatalf("entrada xref %d mal formada: %q", i, entradas[i])
		}
		if offset < 0 || offset >= len(pdf) {
			t.Fatalf("offset %d fuera de rango para el objeto %d", offset, i)
		}
		esperado := fmt.Sprintf("%d 0 obj", i)
		if !strings.HasPrefix(string(pdf[offset:]), esperado) {
			t.Fatalf("el objeto %d no está en el offset %d (xref corrupto)", i, offset)
		}
	}
	// startxref debe apuntar a la tabla xref.
	idxStart := strings.LastIndex(s, "startxref\n")
	if idxStart < 0 {
		t.Fatal("falta startxref")
	}
	var startRef int
	if _, err := fmt.Sscanf(s[idxStart+len("startxref\n"):], "%d", &startRef); err != nil {
		t.Fatalf("startxref mal formado")
	}
	if startRef != inicioXref {
		t.Fatalf("startxref=%d no coincide con el inicio real del xref (%d)", startRef, inicioXref)
	}
}

func TestConstruirPDFMultiPagina(t *testing.T) {
	muchas := make([]string, 200)
	for i := range muchas {
		muchas[i] = "línea de prueba " + string(rune('A'+i%26))
	}
	pdf := construirPDF(muchas)
	s := string(pdf)
	if !strings.Contains(s, "/Count 5") {
		t.Fatalf("esperaba 5 páginas, got: %s", s)
	}
	if strings.Count(s, "/Type /Page ") < 5 {
		t.Fatalf("esperaba 5 objetos de página")
	}
}

func TestGenerarPDFEscribe(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "notas.pdf")
	if err := generarPDFEscribe(ruta, "Título de prueba", []string{"primera", "segunda"}); err != nil {
		t.Fatalf("generarPDFEscribe: %v", err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se escribió el archivo: %v", err)
	}
	if !strings.HasPrefix(string(datos), "%PDF") {
		t.Fatalf("el archivo no parece un PDF")
	}}

func TestExtraerNombrePDF(t *testing.T) {
	casos := []struct {
		entrada string
		want    string
	}{
		{"convertí informe.docx a pdf", "informe.docx"},
		{"convertí el informe a pdf", "informe"},
		{"convertí planilla.xlsx a pdf", "planilla.xlsx"},
		{"convertí el documento a pdf", ""},
	}
	for _, c := range casos {
		if got := extraerNombrePDF(c.entrada); got != c.want {
			t.Errorf("extraerNombrePDF(%q) = %q, quería %q", c.entrada, got, c.want)
		}
	}
}

func TestLocalizarDocumento(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "informe.docx"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(dir, "datos.xlsx"), []byte("x"), 0o600)

	ruta, tipo := localizarDocumento(dir, "informe.docx")
	if ruta == "" || tipo != "word" {
		t.Fatalf("no encontró informe.docx: ruta=%q tipo=%q", ruta, tipo)
	}
	ruta, tipo = localizarDocumento(dir, "informe")
	if ruta == "" || tipo != "word" {
		t.Fatalf("no encontró 'informe' por nombre: ruta=%q tipo=%q", ruta, tipo)
	}
	ruta, _ = localizarDocumento(dir, "noexiste.pdf")
	if ruta != "" {
		t.Fatalf("no debería encontrar un archivo inexistente, obtuve %q", ruta)
	}
}

func TestScriptConvertirAPDF(t *testing.T) {
	word := scriptConvertirAPDF("word", `C:\a\informe.docx`, `C:\a\informe.pdf`)
	if !strings.Contains(word, "SaveAs2") || !strings.Contains(word, "17") {
		t.Fatalf("script word mal formado: %s", word)
	}
	excel := scriptConvertirAPDF("excel", `C:\a\d.xlsx`, `C:\a\d.pdf`)
	if !strings.Contains(excel, "ExportAsFixedFormat") {
		t.Fatalf("script excel mal formado: %s", excel)
	}
	ppt := scriptConvertirAPDF("powerpoint", `C:\a\p.pptx`, `C:\a\p.pdf`)
	if !strings.Contains(ppt, "SaveAs") || !strings.Contains(ppt, "32") {
		t.Fatalf("script powerpoint mal formado: %s", ppt)
	}
	if scriptConvertirAPDF("otro", "a", "b") != "" {
		t.Fatal("tipo desconocido debería devolver vacío")
	}
}

func TestClasificarPDF(t *testing.T) {
	c := NuevoClasificador()
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"creá un pdf con el texto bienvenidos", "pdf"},
		{"exportá mis notas a pdf", "pdf"},
		{"convertí informe.docx a pdf", "pdf"},
		{"pasá el documento a pdf", "pdf"},
		{"abrir pdf", ""},
	}
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}
