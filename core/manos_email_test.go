package core

import (
	"strings"
	"testing"
)

func TestParsearEmail(t *testing.T) {
	casos := []struct {
		comando               string
		destino, asunto, cuerpo string
		ok                    bool
	}{
		{
			"enviar un email a juan@gmail.com con asunto presupuesto y el texto hola juan",
			"juan@gmail.com", "presupuesto", "hola juan", true,
		},
		{
			"enviá un correo a ana@empresa.com con asunto reunión el cuerpo nos vemos mañana",
			"ana@empresa.com", "reunión", "nos vemos mañana", true,
		},
		{
			"mandar un mail a info@site.com.ar asunto consulta texto necesito más información",
			"info@site.com.ar", "consulta", "necesito más información", true,
		},
		{
			"enviar email a sin-direccion",
			"", "", "", false,
		},
		{
			"hola cómo estás",
			"", "", "", false,
		},
		{
			"enviar un email a x@y.com con asunto solo asunto",
			"x@y.com", "solo asunto", "", true,
		},
	}
	for _, c := range casos {
		destino, asunto, cuerpo, ok := parsearEmail(c.comando)
		if ok != c.ok {
			t.Errorf("parsearEmail(%q): ok=%v, esperaba %v", c.comando, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if destino != c.destino {
			t.Errorf("parsearEmail(%q): destino=%q, esperaba %q", c.comando, destino, c.destino)
		}
		if asunto != c.asunto {
			t.Errorf("parsearEmail(%q): asunto=%q, esperaba %q", c.comando, asunto, c.asunto)
		}
		if cuerpo != c.cuerpo {
			t.Errorf("parsearEmail(%q): cuerpo=%q, esperaba %q", c.comando, cuerpo, c.cuerpo)
		}
	}
}

func TestEnviarEmailSinConfigurar(t *testing.T) {
	h := &Hands{EmailEnabled: true}
	msg := h.enviarEmailDesdeComando("enviar un email a juan@gmail.com con asunto hola y el texto cómo estás")
	if !strings.Contains(msg, "config") {
		t.Errorf("debería pedir configuración, fue: %q", msg)
	}
}

func TestEnviarEmailDesactivado(t *testing.T) {
	h := &Hands{EmailEnabled: false}
	msg := h.enviarEmailDesdeComando("enviar un email a juan@gmail.com con asunto hola y el texto cómo estás")
	if !strings.Contains(msg, "desactivado") {
		t.Errorf("debería avisar que está desactivado, fue: %q", msg)
	}
}

func TestEnviarEmailSinDestino(t *testing.T) {
	h := &Hands{EmailEnabled: true, EmailSmtpHost: "x", EmailUsuario: "a@b.com", EmailPassword: "p"}
	msg := h.enviarEmailDesdeComando("enviar un email")
	if !strings.Contains(msg, "destinatario") {
		t.Errorf("debería pedir destinatario, fue: %q", msg)
	}
}

func TestConstruirCorreo(t *testing.T) {
	msg := string(construirCorreo("Jarvis <jarvis@gmail.com>", "juan@empresa.com", "Presupuesto", "Hola Juan"))
	for _, esperado := range []string{
		"From: Jarvis <jarvis@gmail.com>",
		"To: juan@empresa.com",
		"Subject: Presupuesto",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"\r\n\r\nHola Juan",
	} {
		if !strings.Contains(msg, esperado) {
			t.Errorf("construirCorreo no contiene %q:\n%s", esperado, msg)
		}
	}
}

func TestManejarEmailAbrirGmail(t *testing.T) {
	h := &Hands{}
	msg := h.manejarEmail("abrir correo")
	if !strings.Contains(msg, "gmail") {
		t.Errorf("'abrir correo' debería abrir gmail, fue: %q", msg)
	}
}

func TestManejarEmailLeerPendiente(t *testing.T) {
	h := &Hands{}
	msg := h.manejarEmail("leer mis correos")
	if !strings.Contains(msg, "leer") {
		t.Errorf("debería explicar que la lectura está pendiente, fue: %q", msg)
	}
}
