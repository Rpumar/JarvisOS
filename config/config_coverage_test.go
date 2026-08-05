package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type aliasPrueba struct {
	AppName                string            `json:"app_name"`
	Version                string            `json:"version"`
	RequireApproval        bool              `json:"require_approval"`
	TimeoutSegundos        int               `json:"timeout_segundos"`
	RutaMemoria            string            `json:"ruta_memoria"`
	MaxHistorialIA         int               `json:"max_historial_ia"`
	ModeloIA               string            `json:"modelo_ia"`
	IAURL                  string            `json:"ia_url"`
	IAAPIKey               string            `json:"ia_api_key"`
	PINHash                string            `json:"pin_hash"`
	WorkspaceRoot          string            `json:"workspace_root"`
	OpenWeatherKey         string            `json:"open_weather_key"`
	NewsAPIKey             string            `json:"news_api_key"`
	Apps                   map[string]string `json:"apps"`
	ComandoTimeoutSegundos int               `json:"comando_timeout_segundos"`
	LoginPasswordHash      string            `json:"login_password_hash"`
	EmailEnabled           bool              `json:"email_enabled"`
	EmailSmtpHost          string            `json:"email_smtp_host"`
	EmailSmtpPort          int               `json:"email_smtp_port"`
	EmailUsuario           string            `json:"email_usuario"`
	EmailPassword          string            `json:"email_password"`
	EmailDesde             string            `json:"email_desde"`
	EmailImapHost          string            `json:"email_imap_host"`
	EmailImapPort          int               `json:"email_imap_port"`
	EmailImapMax           int               `json:"email_imap_max"`
}

func TestDefaultConfigValoresBase(t *testing.T) {
	c := defaultConfig()
	if c.AppName != "JARVISOS" {
		t.Errorf("AppName = %q, esperaba JARVISOS", c.AppName)
	}
	if !c.RequireApproval {
		t.Error("RequireApproval debería ser true por defecto")
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, esperaba 30s", c.Timeout)
	}
	if c.MaxHistorialIA != 20 {
		t.Errorf("MaxHistorialIA = %d, esperaba 20", c.MaxHistorialIA)
	}
	if c.ComandoTimeoutSegundos != 30 {
		t.Errorf("ComandoTimeoutSegundos = %d, esperaba 30", c.ComandoTimeoutSegundos)
	}
	if c.EmailSmtpPort != 587 || c.EmailImapPort != 993 || c.EmailImapMax != 10 {
		t.Errorf("defaults email incorrectos: %+v", c)
	}
	if _, ok := c.Apps["code"]; !ok {
		t.Error("el mapa de apps por defecto debería incluir code")
	}
	if c.EmailEnabled {
		t.Error("EmailEnabled debería ser false por defecto")
	}
}

func TestLoadCreaArchivoSiNoExiste(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)

	c := Load()

	ruta := filepath.Join(dir, "JarvisOS-datos", "config.json")
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("Load debería crear config.json si no existe: %v", err)
	}
	if c.AppName != "JARVISOS" {
		t.Errorf("con archivo ausente debería usar defaults, AppName = %q", c.AppName)
	}
	if c.WorkspaceRoot != filepath.Join(dir, "Desktop") {
		t.Errorf("WorkspaceRoot = %q, esperaba Desktop del usuario", c.WorkspaceRoot)
	}
}

func TestLoadConfigCorruptoUsaDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	sub := filepath.Join(dir, "JarvisOS-datos")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "config.json"), []byte("{esto no es json"), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Load()
	if c.AppName != "JARVISOS" {
		t.Errorf("config corrupto debería caer a defaults, AppName = %q", c.AppName)
	}
}

func TestLoadAplicaCamposDelArchivo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	sub := filepath.Join(dir, "JarvisOS-datos")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	contenido := `{
		"app_name":"Prueba",
		"version":"9.9",
		"require_approval":false,
		"timeout_segundos":15,
		"max_historial_ia":5,
		"modelo_ia":"mi-modelo",
		"comando_timeout_segundos":60,
		"login_password_hash":"abc",
		"email_enabled":true,
		"email_smtp_host":"smtp.x.com",
		"email_smtp_port":465,
		"email_usuario":"u@x.com",
		"email_password":"pass",
		"email_desde":"Dueno",
		"email_imap_host":"imap.x.com",
		"email_imap_port":143,
		"email_imap_max":3,
		"apps":{"code":"code"},
		"workspace_root":"C:/ws"
	}`
	if err := os.WriteFile(filepath.Join(sub, "config.json"), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Load()
	checks := []struct {
		nombre string
		ok     bool
	}{
		{"app_name", c.AppName == "Prueba"},
		{"version", c.Version == "9.9"},
		{"require_approval false", !c.RequireApproval},
		{"timeout 15s", c.Timeout == 15*time.Second},
		{"max_historial 5", c.MaxHistorialIA == 5},
		{"modelo_ia", c.ModeloIA == "mi-modelo"},
		{"comando_timeout 60", c.ComandoTimeoutSegundos == 60},
		{"login_hash", c.LoginPasswordHash == "abc"},
		{"email_enabled", c.EmailEnabled},
		{"smtp_host", c.EmailSmtpHost == "smtp.x.com"},
		{"smtp_port 465", c.EmailSmtpPort == 465},
		{"usuario", c.EmailUsuario == "u@x.com"},
		{"password", c.EmailPassword == "pass"},
		{"desde", c.EmailDesde == "Dueno"},
		{"imap_host", c.EmailImapHost == "imap.x.com"},
		{"imap_port 143", c.EmailImapPort == 143},
		{"imap_max 3", c.EmailImapMax == 3},
		{"apps code", c.Apps["code"] == "code"},
		{"workspace_root", c.WorkspaceRoot == "C:/ws"},
	}
	for _, ch := range checks {
		if !ch.ok {
			t.Errorf("Load no aplicó el campo %q", ch.nombre)
		}
	}
}

func TestSaveRoundTripTodosLosCampos(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "config.json")

	c := defaultConfig()
	c.RutaConfig = ruta
	c.AppName = "RoundTrip"
	c.RequireApproval = false
	c.Timeout = 45 * time.Second
	c.RutaMemoria = "C:/mem.json"
	c.MaxHistorialIA = 8
	c.ModeloIA = "modelo-x"
	c.IAURL = "http://ollama:11434"
	c.IAAPIKey = "clave"
	c.PINHash = "pinhash"
	c.WorkspaceRoot = "C:/ws"
	c.OpenWeatherKey = "ow"
	c.NewsAPIKey = "news"
	c.ComandoTimeoutSegundos = 120
	c.LoginPasswordHash = "L"
	c.EmailEnabled = true
	c.EmailSmtpHost = "smtp.gmail.com"
	c.EmailSmtpPort = 587
	c.EmailUsuario = "a@b.com"
	c.EmailPassword = "pass"
	c.EmailDesde = "Jarvis"
	c.EmailImapHost = "imap.gmail.com"
	c.EmailImapPort = 993
	c.EmailImapMax = 25
	c.Apps = map[string]string{"code": "code", "chrome": "chrome"}

	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer archivo: %v", err)
	}
	var alias aliasPrueba
	if err := json.Unmarshal(contenido, &alias); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if alias.AppName != c.AppName || alias.Version != c.Version || alias.RequireApproval != c.RequireApproval ||
		alias.TimeoutSegundos != 45 || alias.RutaMemoria != c.RutaMemoria ||
		alias.MaxHistorialIA != 8 || alias.ModeloIA != c.ModeloIA ||
		alias.IAURL != c.IAURL || alias.IAAPIKey != c.IAAPIKey || alias.PINHash != c.PINHash ||
		alias.WorkspaceRoot != c.WorkspaceRoot || alias.OpenWeatherKey != "ow" || alias.NewsAPIKey != "news" ||
		alias.ComandoTimeoutSegundos != 120 || alias.LoginPasswordHash != "L" ||
		!alias.EmailEnabled || alias.EmailSmtpHost != c.EmailSmtpHost || alias.EmailSmtpPort != 587 ||
		alias.EmailUsuario != c.EmailUsuario || alias.EmailPassword != c.EmailPassword ||
		alias.EmailDesde != c.EmailDesde || alias.EmailImapHost != c.EmailImapHost ||
		alias.EmailImapPort != 993 || alias.EmailImapMax != 25 ||
		alias.Apps["code"] != "code" || alias.Apps["chrome"] != "chrome" {
		t.Errorf("round-trip no conservó todos los campos: %+v", alias)
	}
}

func TestSaveCamposVaciosNoSePisanAlCargar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	sub := filepath.Join(dir, "JarvisOS-datos")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	contenido := `{"app_name":"Prueba"}` // solo un campo
	if err := os.WriteFile(filepath.Join(sub, "config.json"), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Load()
	if c.AppName != "Prueba" {
		t.Errorf("AppName = %q, esperaba Prueba", c.AppName)
	}
	if c.ModeloIA != defaultConfig().ModeloIA {
		t.Errorf("los campos ausentes deberían conservar el default, ModeloIA = %q", c.ModeloIA)
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, esperaba el default 30s", c.Timeout)
	}
	if !c.RequireApproval {
		t.Error("RequireApproval ausente debería conservar el default true")
	}
}
