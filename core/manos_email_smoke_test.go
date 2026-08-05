package core

import (
	"fmt"
	"os"
	"testing"
)

// TestEmailHumoVivo valida el flujo real contra Gmail. Se corre solo si
// JARVIS_EMAIL_SMOKE=1, porque necesita credenciales reales. Envía un correo
// de prueba al dueño y luego lo lee por IMAP.
func TestEmailHumoVivo(t *testing.T) {
	if os.Getenv("JARVIS_EMAIL_SMOKE") == "" {
		t.Skip("skip: requiere JARVIS_EMAIL_SMOKE=1 y credenciales reales en config.json")
	}
	usuario := os.Getenv("JARVIS_EMAIL_USUARIO")
	clave := os.Getenv("JARVIS_EMAIL_CLAVE")
	if usuario == "" || clave == "" {
		t.Fatal("faltan JARVIS_EMAIL_USUARIO o JARVIS_EMAIL_CLAVE")
	}

	h := &Hands{
		EmailEnabled:  true,
		EmailSmtpHost: "smtp.gmail.com",
		EmailSmtpPort: 587,
		EmailUsuario:  usuario,
		EmailPassword: clave,
		EmailDesde:    "Jarvis Test",
		EmailImapHost: "imap.gmail.com",
		EmailImapPort: 993,
		EmailImapMax:  5,
	}

	asunto := "Prueba JarvisOS " + t.Name()
	if err := h.enviarEmailSMTP(usuario, asunto, "Hola, esta es una prueba del envío SMTP de JarvisOS."); err != nil {
		t.Fatalf("enviarEmailSMTP: %v", err)
	}
	t.Log("SMTP OK: correo enviado")

	correos, err := leerBandejaIMAP(h.EmailImapHost, h.EmailImapPort, usuario, clave, 5)
	if err != nil {
		t.Fatalf("leerBandejaIMAP: %v", err)
	}
	t.Logf("IMAP OK: %d correos leídos", len(correos))
	if len(correos) == 0 {
		t.Fatal("no se leyeron correos de la bandeja")
	}
	encontrado := false
	for _, c := range correos {
		if c.Asunto == asunto {
			encontrado = true
			break
		}
	}
	if !encontrado {
		t.Errorf("no se encontró el correo de prueba '%s' entre los últimos correos", asunto)
	} else {
		t.Log("FELIZ: el correo de prueba fue encontrado en la bandeja")
	}
	_ = fmt.Sprint
}
