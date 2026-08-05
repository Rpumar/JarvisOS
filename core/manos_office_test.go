package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTipoOffice(t *testing.T) {
	casos := []struct {
		cmd      string
		esperado string
	}{
		{"crea un documento word", "word"},
		{"crear un word", "word"},
		{"crear una planilla de excel", "excel"},
		{"crear un excel", "excel"},
		{"crear una hoja de calculo", "excel"},
		{"crear una presentacion de powerpoint", "powerpoint"},
		{"crear diapositivas", "powerpoint"},
		{"abrir chrome", ""},
		{"decime la hora", ""},
	}
	for _, c := range casos {
		if got := tipoOffice(c.cmd); got != c.esperado {
			t.Errorf("tipoOffice(%q) = %q, esperaba %q", c.cmd, got, c.esperado)
		}
	}
}

func TestExtraerNombreArchivoOffice(t *testing.T) {
	casos := []struct {
		cmd      string
		tipo     string
		esperado string
	}{
		{"crea un documento word llamado informe", "word", "informe.docx"},
		{"crear un excel en presupuesto", "excel", "presupuesto.xlsx"},
		{"crear una presentacion de powerpoint", "powerpoint", "documento-jarvis.pptx"},
		{"crea un word", "word", "documento-jarvis.docx"},
		{"crea una planilla de excel en datos ya con extension.xlsx", "excel", "datos ya con extension.xlsx"},
	}
	for _, c := range casos {
		if got := extraerNombreArchivoOffice(c.cmd, c.tipo); got != c.esperado {
			t.Errorf("extraerNombreArchivoOffice(%q, %q) = %q, esperaba %q", c.cmd, c.tipo, got, c.esperado)
		}
	}
}

func TestScriptOfficeGenerado(t *testing.T) {
	word := scriptOffice("word", `C:\Jarvis\mi doc.docx`)
	for _, esperado := range []string{"Word.Application", "SaveAs2", "16", "mi doc.docx"} {
		if !strings.Contains(word, esperado) {
			t.Errorf("scriptOffice(word) no contiene %q", esperado)
		}
	}

	excel := scriptOffice("excel", `C:\Jarvis\planilla.xlsx`)
	for _, esperado := range []string{"Excel.Application", "SaveAs", "51", "planilla.xlsx"} {
		if !strings.Contains(excel, esperado) {
			t.Errorf("scriptOffice(excel) no contiene %q", esperado)
		}
	}

	ppt := scriptOffice("powerpoint", `C:\Jarvis\presentacion.pptx`)
	for _, esperado := range []string{"PowerPoint.Application", "SaveAs", "24", "presentacion.pptx"} {
		if !strings.Contains(ppt, esperado) {
			t.Errorf("scriptOffice(powerpoint) no contiene %q", esperado)
		}
	}
}

func TestScriptOfficeEscapaComillas(t *testing.T) {
	script := scriptOffice("excel", `C:\Jarvis\planilla final.xlsx`)
	if strings.Contains(script, "planilla final.xlsx'") == false {
		t.Errorf("la ruta debería quedar entre comillas simples: %s", script)
	}
}

// TestOfficeHumoVivo genera documentos reales con Office instalado. Se corre
// solo si JARVIS_SMOKE=1, porque necesita Word/Excel/PowerPoint de verdad.
func TestOfficeHumoVivo(t *testing.T) {
	if os.Getenv("JARVIS_SMOKE") == "" {
		t.Skip("skip: requiere JARVIS_SMOKE=1 y Office instalado")
	}
	dir := t.TempDir()
	h := &Hands{WorkspaceRoot: dir}
	for _, tipo := range []string{"word", "excel", "powerpoint"} {
		ext := extensionesOffice[tipo]
		ruta := filepath.Join(dir, "humo"+ext)
		if err := h.crearDocumentoOffice(tipo, ruta); err != nil {
			t.Errorf("crearDocumentoOffice(%s): %v", tipo, err)
			continue
		}
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("no se generó %s: %v", ruta, err)
		}
	}
}
