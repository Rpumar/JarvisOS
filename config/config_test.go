package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGuardarYCargarEmail(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "config.json")

	c := defaultConfig()
	c.RutaConfig = ruta
	c.EmailEnabled = true
	c.EmailSmtpHost = "smtp.gmail.com"
	c.EmailSmtpPort = 587
	c.EmailUsuario = "dueno@gmail.com"
	c.EmailPassword = "pass-aplicacion"
	c.EmailDesde = "Jarvis"
	c.EmailImapHost = "imap.gmail.com"
	c.EmailImapPort = 993
	c.EmailImapMax = 15
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer archivo: %v", err)
	}
	alias := struct {
		EmailEnabled  bool   `json:"email_enabled"`
		EmailSmtpHost string `json:"email_smtp_host"`
		EmailSmtpPort int    `json:"email_smtp_port"`
		EmailUsuario  string `json:"email_usuario"`
		EmailPassword string `json:"email_password"`
		EmailDesde    string `json:"email_desde"`
		EmailImapHost string `json:"email_imap_host"`
		EmailImapPort int    `json:"email_imap_port"`
		EmailImapMax  int    `json:"email_imap_max"`
	}{}
	if err := json.Unmarshal(contenido, &alias); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !alias.EmailEnabled || alias.EmailSmtpHost != c.EmailSmtpHost || alias.EmailSmtpPort != c.EmailSmtpPort ||
		alias.EmailUsuario != c.EmailUsuario || alias.EmailPassword != c.EmailPassword || alias.EmailDesde != c.EmailDesde ||
		alias.EmailImapHost != c.EmailImapHost || alias.EmailImapPort != c.EmailImapPort || alias.EmailImapMax != c.EmailImapMax {
		t.Errorf("round-trip email incorrecto: %+v", alias)
	}
}
