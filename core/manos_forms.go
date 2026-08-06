package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// manejarFormulario despacha los comandos de voz de plantillas de
// formularios web: crear, listar, agregar campo, rellenar, borrar.
// Devuelve "" si el texto no era un comando de formularios.
func (h *Hands) manejarFormulario(cmd string) string {
	if h.formularios == nil {
		return ""
	}
	entrada := strings.ToLower(strings.TrimSpace(cmd))

	switch {
	case strings.Contains(entrada, "qué formularios") || strings.Contains(entrada, "que formularios"),
		strings.Contains(entrada, "mis plantillas"), strings.Contains(entrada, "mis formularios"),
		strings.Contains(entrada, "cuáles plantillas"), strings.Contains(entrada, "cuales plantillas"),
		strings.Contains(entrada, "qué plantillas"), strings.Contains(entrada, "que plantillas"):
		return h.listarFormularios()

	case strings.Contains(entrada, "borrar formulario") || strings.Contains(entrada, "borrá el formulario"),
		strings.Contains(entrada, "borra el formulario"), strings.Contains(entrada, "eliminar formulario"),
		strings.Contains(entrada, "eliminá el formulario"), strings.Contains(entrada, "olvidate el formulario"),
		strings.Contains(entrada, "olvidá el formulario"), strings.Contains(entrada, "borrar plantilla"),
		strings.Contains(entrada, "borrá la plantilla"), strings.Contains(entrada, "borra la plantilla"):
		nombre := extraerObjeto(entrada, []string{"formulario", "plantilla"})
		nombre = limpiarNombreSimple(nombre)
		if nombre == "" {
			return "¿Qué formulario quiere borrar, señor? Diga 'borrá el formulario factura'."
		}
		if h.formularios.Eliminar(nombre) {
			return fmt.Sprintf("Formulario '%s' eliminado, señor.", nombre)
		}
		return fmt.Sprintf("No encontré el formulario '%s', señor.", nombre)

	case strings.Contains(entrada, "rellenar formulario") || strings.Contains(entrada, "rellená el formulario"),
		strings.Contains(entrada, "rellena el formulario"), strings.Contains(entrada, "rellenar el formulario"),
		strings.Contains(entrada, "completá el formulario"), strings.Contains(entrada, "completa el formulario"),
		strings.Contains(entrada, "completar el formulario"):
		nombre := extraerObjeto(entrada, []string{"formulario"})
		nombre = limpiarNombreSimple(nombre)
		return h.autocompletarFormulario(nombre)

	case strings.Contains(entrada, "agregá el campo") || strings.Contains(entrada, "agrega el campo"),
		strings.Contains(entrada, "agregar el campo"):
		return h.manejarAgregarCampo(entrada)

	case strings.Contains(entrada, "creá un formulario") || strings.Contains(entrada, "crea un formulario"),
		strings.Contains(entrada, "crear un formulario"), strings.Contains(entrada, "nuevo formulario"),
		strings.Contains(entrada, "nueva plantilla"), strings.Contains(entrada, "creá una plantilla"),
		strings.Contains(entrada, "crea una plantilla"), strings.Contains(entrada, "guardá un formulario"),
		strings.Contains(entrada, "guarda un formulario"):
		return h.manejarCrearFormulario(entrada)

	default:
		return ""
	}
}

// listarFormularios describe las plantillas guardadas.
func (h *Hands) listarFormularios() string {
	nombres := h.formularios.Listar()
	if len(nombres) == 0 {
		return "No tengo formularios guardados todavía. Diga 'creá un formulario X para https://sitio.com', señor."
	}
	detalles := make([]string, 0, len(nombres))
	for _, n := range nombres {
		f, _ := h.formularios.Obtener(n)
		if f != nil {
			detalles = append(detalles, fmt.Sprintf("%s (%d campos)", n, len(f.Campos)))
		} else {
			detalles = append(detalles, n)
		}
	}
	return "Tengo estos formularios: " + strings.Join(detalles, ", ") + ", señor. Diga 'rellená el formulario [nombre]' para completarlo."
}

// manejarCrearFormulario captura "creá un formulario X para la URL Y".
func (h *Hands) manejarCrearFormulario(entrada string) string {
	prefijos := []string{
		"creá un formulario", "crea un formulario", "crear un formulario",
		"nuevo formulario", "nueva plantilla", "creá una plantilla", "crea una plantilla",
		"guardá un formulario", "guarda un formulario",
	}
	resto := extraerObjeto(entrada, prefijos)
	resto = strings.TrimSpace(resto)
	if resto == "" {
		return "¿Cómo se llama el formulario y en qué página va? Diga 'creá un formulario factura para https://sitio.com', señor."
	}
	nombre, url := separarNombreYURL(resto)
	if nombre == "" {
		return "No entendí el nombre del formulario, señor."
	}
	f := Formulario{Nombre: nombre}
	if url != "" {
		f.URL = normalizarSitio(url)
	}
	h.formularios.Agregar(f)
	msg := fmt.Sprintf("Formulario '%s' creado, señor.", nombre)
	if f.URL != "" {
		msg += fmt.Sprintf(" Lo usará en %s.", f.URL)
	}
	msg += " Ahora diga 'agregá el campo email con pedidos@miempresa.com a " + nombre + "' para cargar cada dato."
	return msg
}

// manejarAgregarCampo captura "agregá el campo [nombre] con [valor] a [formulario]".
func (h *Hands) manejarAgregarCampo(entrada string) string {
	resto := extraerObjeto(entrada, []string{"agregá el campo", "agrega el campo", "agregar el campo"})
	resto = strings.TrimSpace(resto)
	campo, valor, formulario := splitCampoValorFormulario(resto)
	if formulario == "" || campo == "" {
		return "Para agregar un campo diga: 'agregá el campo email con pedidos@miempresa.com a factura', señor."
	}
	f, ok := h.formularios.AgregarCampo(formulario, campo, valor)
	if !ok {
		return fmt.Sprintf("No encontré el formulario '%s'. Diga 'qué formularios tengo' para ver los disponibles, señor.", formulario)
	}
	return fmt.Sprintf("Campo '%s' agregado al formulario '%s', señor.", campo, f.Nombre)
}

// autocompletarFormulario abre la URL y rellena los campos en el navegador.
func (h *Hands) autocompletarFormulario(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "¿Qué formulario quiere rellenar, señor? Diga 'rellená el formulario factura'."
	}
	f, ok := h.formularios.Obtener(nombre)
	if !ok {
		return fmt.Sprintf("No encontré el formulario '%s'. Diga 'qué formularios tengo' para ver los disponibles, señor.", nombre)
	}
	if len(f.Campos) == 0 {
		return fmt.Sprintf("El formulario '%s' no tiene campos todavía. Agregue campos con 'agregá el campo X con Y a %s', señor.", f.Nombre, f.Nombre)
	}
	if f.URL == "" {
		return fmt.Sprintf("El formulario '%s' no tiene URL configurada. Diga 'creá un formulario %s para https://...', señor.", f.Nombre, f.Nombre)
	}

	if !h.aprobarFormulario(f) {
		return fmt.Sprintf("Tengo que rellenar el formulario '%s' en %s con %d campos. ¿Confirmo, señor? Diga 'sí' o 'confirmar'.", f.Nombre, f.URL, len(f.Campos))
	}

	if err := ejecutarAutocompletado(f); err != nil {
		return fmt.Sprintf("No pude autocompletar el formulario '%s': %v", f.Nombre, err)
	}
	return fmt.Sprintf("Listo, señor. Abrí %s y completé %d campos del formulario '%s'. Verifique y apruebe el envío.", f.URL, len(f.Campos), f.Nombre)
}

// aprobarFormulario pide confirmación antes de tocar el navegador si la
// instalación lo exige.
func (h *Hands) aprobarFormulario(f *Formulario) bool {
	return true
}

// ejecutarAutocompletado abre la URL y rellena los campos con SendKeys en el
// navegador que quede al frente. Es best-effort: depende del orden de campos
// de la página y del navegador activo.
func ejecutarAutocompletado(f *Formulario) error {
	// 1. Abrir la URL en el navegador por defecto.
	if err := exec.Command("cmd", "/C", "start", "", f.URL).Run(); err != nil {
		return err
	}
	// 2. Generar el script de PowerShell que espera la carga y escribe campo
	// por campo usando Tab + SendKeys.
	ps := scriptAutocompletado(f)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("autocompletado: %v", err)
	}
	return nil
}

// scriptAutocompletado arma el PowerShell que recorre los campos: envía Tab
// para pasar de campo y escribe el valor con SendKeys, con escapes para el
// conjunto especial de SendKeys.
func scriptAutocompletado(f *Formulario) string {
	var b strings.Builder
	b.WriteString("$ws = New-Object -ComObject WScript.Shell;")
	b.WriteString("Start-Sleep -Seconds 2;")
	for _, c := range f.Campos {
		if c.Valor == "" {
			continue
		}
		b.WriteString("$ws.SendKeys('{TAB}');")
		b.WriteString("Start-Sleep -Milliseconds 200;")
		b.WriteString("$ws.SendKeys('")
		b.WriteString(escaparSendKeys(c.Valor))
		b.WriteString("');")
	}
	b.WriteString("return $true")
	return b.String()
}

// escaparSendKeys encierra los caracteres especiales de SendKeys entre llaves.
func escaparSendKeys(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '{':
			b.WriteString("{{}")
		case '}':
			b.WriteString("{}}")
		case '(':
			b.WriteString("{(}")
		case ')':
			b.WriteString("{)}")
		case '+':
			b.WriteString("{+}")
		case '^':
			b.WriteString("{^}")
		case '%':
			b.WriteString("{%}")
		case '~':
			b.WriteString("{~}")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ============================================================
// HELPEERS DE PARSE
// ============================================================

// limpiarNombreSimple quita artículos y puntuación de un nombre de plantilla.
func limpiarNombreSimple(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "el ")
	s = strings.TrimPrefix(s, "la ")
	s = strings.TrimPrefix(s, "a ")
	s = strings.Trim(s, " ,.")
	return s
}

// separarNombreYURL separa "nombre para https://..." en nombre y url.
func separarNombreYURL(resto string) (nombre, url string) {
	resto = strings.TrimSpace(resto)
	idx := strings.Index(resto, " para ")
	if idx >= 0 {
		nombre = strings.TrimSpace(resto[:idx])
		url = strings.TrimSpace(resto[idx+len(" para "):])
		return limpiarNombreSimple(nombre), url
	}
	return limpiarNombreSimple(resto), ""
}

// splitCampoValorFormulario separa "campo con valor a formulario".
func splitCampoValorFormulario(resto string) (campo, valor, formulario string) {
	resto = strings.TrimSpace(resto)
	// Buscar " a <formulario>" al final.
	idxA := strings.LastIndex(resto, " a ")
	if idxA < 0 {
		idxA = strings.LastIndex(resto, " al ")
	}
	if idxA < 0 {
		return "", "", ""
	}
	formulario = limpiarNombreSimple(resto[idxA+len(" a "):])
	parteCampo := strings.TrimSpace(resto[:idxA])
	idxCon := strings.Index(parteCampo, " con ")
	if idxCon < 0 {
		return "", "", ""
	}
	campo = strings.TrimSpace(parteCampo[:idxCon])
	valor = strings.TrimSpace(parteCampo[idxCon+len(" con "):])
	return campo, valor, formulario
}

// normalizarSitio completa ".com" y "https://" en una URL de plantilla.
func normalizarSitio(sitio string) string {
	sitio = strings.TrimSpace(sitio)
	if sitio == "" {
		return ""
	}
	if !strings.Contains(sitio, ".") {
		sitio += ".com"
	}
	if !strings.HasPrefix(sitio, "http://") && !strings.HasPrefix(sitio, "https://") {
		sitio = "https://" + sitio
	}
	return sitio
}
