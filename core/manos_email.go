package core

import (
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
)

// === EMAIL (F3): envío por SMTP con la librería estándar ===
// Gmail y Outlook se configuran con contraseña de aplicación en config.json
// (email_smtp_host, email_smtp_port, email_usuario, email_password). El envío
// es una acción externa: está marcada como sensible en core/security y pasa
// por aprobación (PIN/panel) antes de ejecutarse.

// emailRe detecta una dirección de correo dentro del comando.
var emailRe = regexp.MustCompile(`[\w.+-]+@[\w-]+(\.[\w-]+)+`)

// manejarEmail interpreta un comando de correo del dueño.
func (h *Hands) manejarEmail(cmd string) string {
	comando := strings.ToLower(strings.TrimSpace(cmd))

	// "abrir correo/gmail" sigue abriendo el sitio, no envía ni lee.
	if strings.Contains(comando, "abrir") {
		return h.irASitio("gmail.com")
	}

	switch {
	case strings.Contains(comando, "enviar") || strings.Contains(comando, "enviá") ||
		strings.Contains(comando, "envia") || strings.Contains(comando, "mandar") ||
		strings.Contains(comando, "mandá") || strings.Contains(comando, "manda"):
		return h.enviarEmailDesdeComando(comando)
	case strings.Contains(comando, "leer") || strings.Contains(comando, "revisar") ||
		strings.Contains(comando, "ver mis") || strings.Contains(comando, "cuantos") ||
		strings.Contains(comando, "cuántos"):
		return h.leerEmail(comando)
	default:
		return "Puedo enviar un correo. Diga 'enviá un email a persona@dominio.com con asunto ... y el texto ...', señor."
	}
}

// parsearEmail separa un comando de envío en destinatario, asunto y cuerpo.
func parsearEmail(comando string) (destino, asunto, cuerpo string, ok bool) {
	destino = emailRe.FindString(comando)
	if destino == "" {
		return "", "", "", false
	}

	// Asunto: todo lo que sigue a "asunto " hasta "el texto / texto / cuerpo / con el texto".
	asunto = extraerCampo(comando, []string{"asunto ", "con el asunto "}, []string{"el texto", "el cuerpo", "cuerpo ", "texto "})
	cuerpo = extraerCampo(comando, []string{"el texto ", "cuerpo ", "con el texto ", "con el cuerpo ", "texto "}, nil)

	if asunto == "" && cuerpo == "" {
		return destino, "", "", true
	}
	return destino, asunto, cuerpo, true
}

// extraerCampo toma el texto que sigue a uno de los marcadores iniciales,
// cortado en el primer marcador final (si alguno aparece).
func extraerCampo(entrada string, desde, hasta []string) string {
	inicio := -1
	marcador := ""
	for _, m := range desde {
		if i := strings.Index(entrada, m); i >= 0 && (inicio < 0 || i < inicio) {
			inicio = i
			marcador = m
		}
	}
	if inicio < 0 {
		return ""
	}
	campo := entrada[inicio+len(marcador):]
	if inicioHasta, ok := campoHasta(campo, hasta); ok {
		campo = inicioHasta
	}
	campo = strings.TrimSpace(campo)
	// "asunto presupuesto y el texto ..." → asunto "presupuesto" (el "y"
	// enlaza asunto con el cuerpo).
	campo = strings.TrimSuffix(campo, " y")
	return strings.TrimSpace(campo)
}

func campoHasta(campo string, hasta []string) (string, bool) {
	for _, m := range hasta {
		if i := strings.Index(campo, m); i >= 0 {
			return campo[:i], true
		}
	}
	return campo, false
}

// enviarEmailDesdeComando valida la configuración y delega el envío.
func (h *Hands) enviarEmailDesdeComando(comando string) string {
	destino, asunto, cuerpo, ok := parsearEmail(comando)
	if !ok {
		return "No pude encontrar el destinatario. Diga 'enviá un email a persona@dominio.com con asunto ... y el texto ...', señor."
	}
	if asunto == "" {
		return "Falta el asunto. Diga 'enviá un email a persona@dominio.com con asunto ... y el texto ...', señor."
	}
	if err := h.enviarEmailSMTP(destino, asunto, cuerpo); err != nil {
		return fmt.Sprintf("No pude enviar el correo, señor: %v", err)
	}
	return fmt.Sprintf("Correo enviado a %s con asunto '%s', señor.", destino, asunto)
}

// enviarEmailSMTP envía un correo por SMTP (TLS en el puerto 587).
func (h *Hands) enviarEmailSMTP(destino, asunto, cuerpo string) error {
	if !h.EmailEnabled {
		return fmt.Errorf("el correo está desactivado. Configúrelo en config.json (email_enabled: true) y escriba host, usuario y contraseña de aplicación")
	}
	if h.EmailSmtpHost == "" || h.EmailUsuario == "" || h.EmailPassword == "" {
		return fmt.Errorf("falta la configuración SMTP en config.json (email_smtp_host, email_usuario, email_password)")
	}

	desde := h.EmailUsuario
	if h.EmailDesde != "" {
		desde = fmt.Sprintf("%s <%s>", h.EmailDesde, h.EmailUsuario)
	}

	msg := construirCorreo(desde, destino, asunto, cuerpo)

	puerto := h.EmailSmtpPort
	if puerto == 0 {
		puerto = 587
	}
	servidor := fmt.Sprintf("%s:%d", h.EmailSmtpHost, puerto)
	auth := smtp.PlainAuth("", h.EmailUsuario, h.EmailPassword, h.EmailSmtpHost)

	return smtp.SendMail(servidor, auth, h.EmailUsuario, []string{destino}, msg)
}

// construirCorreo arma el mensaje RFC 5322 (cabeceras + cuerpo UTF-8).
// Usa las direcciones en bruto tal como se escribieron (el SMTP y el
// destinatario las entienden; evitar normalizarlas con net/mail evita
// comillas o <...> extra en las cabeceras).
func construirCorreo(desde, para, asunto, cuerpo string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + desde + "\r\n")
	sb.WriteString("To: " + para + "\r\n")
	sb.WriteString("Subject: " + asunto + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(cuerpo)
	return []byte(sb.String())
}

// leerEmail informa el estado de la lectura de correos (pendiente F3).
func (h *Hands) leerEmail(comando string) string {
	return "Todavía no puedo leer la bandeja de entrada, señor: la lectura por IMAP requiere una librería externa y la dejo para el próximo paso de F3. El envío de correos ya funciona: diga 'enviá un email a persona@dominio.com con asunto ... y el texto ...'."
}
