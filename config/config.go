package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	AppName         string        `json:"app_name"`
	Version         string        `json:"version"`
	RequireApproval bool          `json:"require_approval"`
	Timeout         time.Duration `json:"timeout"`
	RutaMemoria     string        `json:"ruta_memoria"`
	RutaConfig      string        `json:"-"`

	MaxHistorialIA int `json:"max_historial_ia"`

	ModeloIA       string            `json:"modelo_ia"`
	IAURL          string            `json:"ia_url"`
	IAAPIKey       string            `json:"ia_api_key"`
	PINHash        string            `json:"pin_hash"`
	WorkspaceRoot  string            `json:"workspace_root"`
	OpenWeatherKey string            `json:"open_weather_key"`
	NewsAPIKey     string            `json:"news_api_key"`
	Apps           map[string]string `json:"apps"`

	// ComandoTimeoutSegundos es el límite de ejecución de comandos externos
	// (30 s por defecto).
	ComandoTimeoutSegundos int `json:"comando_timeout_segundos"`

	// LoginPasswordHash es el hash SHA-256 (hex) de la contraseña de acceso
	// al panel web. Si está vacío, el panel está abierto (modo piloto); si
	// está definido, exige inicio de sesión (dueño = Admin).
	LoginPasswordHash string `json:"login_password_hash"`

	// EmailEnabled habilita el envío de correos por SMTP.
	EmailEnabled bool `json:"email_enabled"`
	// EmailSmtpHost es el servidor SMTP (ej. smtp.gmail.com, smtp.office365.com).
	EmailSmtpHost string `json:"email_smtp_host"`
	// EmailSmtpPort es el puerto SMTP (587/TLS por defecto).
	EmailSmtpPort int `json:"email_smtp_port"`
	// EmailUsuario es la cuenta que envía (ej. dueño@gmail.com).
	EmailUsuario string `json:"email_usuario"`
	// EmailPassword es la contraseña de aplicación del SMTP.
	EmailPassword string `json:"email_password"`
	// EmailDesde es el nombre visible del remitente (ej. "Jarvis").
	EmailDesde string `json:"email_desde"`
	// EmailImapHost es el servidor IMAP (ej. imap.gmail.com).
	EmailImapHost string `json:"email_imap_host"`
	// EmailImapPort es el puerto IMAP TLS (993 por defecto).
	EmailImapPort int `json:"email_imap_port"`
	// EmailImapMax es cuántos correos leer de la bandeja por defecto (10).
	EmailImapMax int `json:"email_imap_max"`

	// XApiKey / XApiSecret son las claves de la app de X (Twitter API v2,
	// OAuth 1.0a). XAccessToken / XAccessSecret son los tokens del usuario.
	XApiKey       string `json:"x_api_key"`
	XApiSecret    string `json:"x_api_secret"`
	XAccessToken  string `json:"x_access_token"`
	XAccessSecret string `json:"x_access_secret"`

	// LinkedInToken es un token de acceso OAuth 2.0 (Bearer) para la API de
	// LinkedIn, y LinkedInAuthor es el URN del autor ("urn:li:person:..." o
	// "urn:li:organization:...").
	LinkedInToken  string `json:"linkedin_token"`
	LinkedInAuthor string `json:"linkedin_author"`

	// LicenseKey es la clave de licencia local (JARVIS-PLAN-PUESTOS-NONCE-
	// FIRMA). Vacía = modo piloto (1 puesto). Se valida al arrancar.
	LicenseKey string `json:"license_key"`

	// ControlURL es la URL del plano de control en la nube (licencias,
	// heartbeat y puestos). Vacía = sin plano de control (100% local).
	ControlURL string `json:"control_url"`
	// IdInstalacion identifica de forma única esta instalación ante el
	// plano de control. Se genera una vez y se persiste.
	IdInstalacion string `json:"id_instalacion"`
}

// DatosDir devuelve el directorio de datos persistente del usuario. Por
// defecto es %USERPROFILE%\JarvisOS-datos; la variable JARVISOS_DATOS lo
// sobrescribe (lo usa demo.ps1 para correr una sandbox sin tocar los datos
// reales).
func DatosDir() string {
	if dir := os.Getenv("JARVISOS_DATOS"); dir != "" {
		return dir
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos")
}

func defaultConfig() *Config {
	return &Config{
		AppName:                "JARVISOS",
		Version:                "0.14.1",
		RequireApproval:        true,
		Timeout:                30 * time.Second,
		RutaMemoria:            filepath.Join(DatosDir(), "memoria.json"),
		MaxHistorialIA:         20,
		ModeloIA:               "qwen2.5-coder:7b",
		IAURL:                  "",
		IAAPIKey:               "",
		WorkspaceRoot:          filepath.Join(os.Getenv("USERPROFILE"), "Desktop"),
		OpenWeatherKey:         "",
		NewsAPIKey:             "",
		ComandoTimeoutSegundos: 30,
		EmailEnabled:           false,
		EmailSmtpPort:          587,
		EmailDesde:             "Jarvis",
		EmailImapPort:          993,
		EmailImapMax:           10,
		Apps: map[string]string{
			"code":          "code",
			"calculadora":   "calc",
			"bloc":          "notepad",
			"chrome":        "chrome",
			"word":          "winword",
			"excel":         "excel",
			"powerpoint":    "powerpnt",
			"opera":         "opera",
			"firefox":       "firefox",
			"edge":          "msedge",
			"outlook":       "outlook",
			"terminal":      "windows-terminal",
			"cmd":           "cmd",
			"powershell":    "powershell",
			"paint":         "mspaint",
			"zoom":          "zoom",
			"teams":         "teams",
			"vscode":        "code",
			"calendario":    "outlookcal:",
			"cámara":        "windows+camera:",
			"camara":        "windows+camera:",
			"fotos":         "ms-photos:",
			"música":        "ms-music:",
			"musica":        "ms-music:",
			"videos":        "ms-video:",
			"mapas":         "bingmaps:",
			"noticias app":  "ms-news:",
			"clima app":     "ms-weather:",
			"bloc de notas": "notepad",
		},
	}
}

func Load() *Config {
	cfg := defaultConfig()
	cfg.RutaConfig = filepath.Join(DatosDir(), "config.json")

	contenido, err := os.ReadFile(cfg.RutaConfig)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.garantizarIdInstalacion()
			_ = cfg.Save()
			return cfg
		}
		fmt.Fprintf(os.Stderr, "[ADVERTENCIA] No se pudo leer config: %v. Usando valores por defecto.\n", err)
		return cfg
	}

	type configAlias struct {
		AppName                string            `json:"app_name"`
		Version                string            `json:"version"`
		RequireApproval        *bool             `json:"require_approval"`
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
		XApiKey                string            `json:"x_api_key"`
		XApiSecret             string            `json:"x_api_secret"`
		XAccessToken           string            `json:"x_access_token"`
		XAccessSecret          string            `json:"x_access_secret"`
		LinkedInToken          string            `json:"linkedin_token"`
		LinkedInAuthor         string            `json:"linkedin_author"`
		LicenseKey             string            `json:"license_key"`
		ControlURL             string            `json:"control_url"`
		IdInstalacion          string            `json:"id_instalacion"`
	}

	var alias configAlias
	if err := json.Unmarshal(contenido, &alias); err != nil {
		fmt.Fprintf(os.Stderr, "[ADVERTENCIA] Config.json corrupto: %v. Usando valores por defecto.\n", err)
		return cfg
	}

	if alias.AppName != "" {
		cfg.AppName = alias.AppName
	}
	if alias.Version != "" {
		cfg.Version = alias.Version
	}
	cfg.RequireApproval = alias.RequireApproval == nil || *alias.RequireApproval
	if alias.TimeoutSegundos > 0 {
		cfg.Timeout = time.Duration(alias.TimeoutSegundos) * time.Second
	}
	if alias.RutaMemoria != "" {
		cfg.RutaMemoria = alias.RutaMemoria
	}
	if alias.MaxHistorialIA > 0 {
		cfg.MaxHistorialIA = alias.MaxHistorialIA
	}
	if alias.ModeloIA != "" {
		cfg.ModeloIA = alias.ModeloIA
	}
	if alias.IAURL != "" {
		cfg.IAURL = alias.IAURL
	}
	if alias.IAAPIKey != "" {
		cfg.IAAPIKey = alias.IAAPIKey
	}
	if alias.PINHash != "" {
		cfg.PINHash = alias.PINHash
	}
	if alias.WorkspaceRoot != "" {
		cfg.WorkspaceRoot = alias.WorkspaceRoot
	}
	if alias.OpenWeatherKey != "" {
		cfg.OpenWeatherKey = alias.OpenWeatherKey
	}
	if alias.NewsAPIKey != "" {
		cfg.NewsAPIKey = alias.NewsAPIKey
	}
	if alias.Apps != nil {
		cfg.Apps = alias.Apps
	}
	if alias.ComandoTimeoutSegundos > 0 {
		cfg.ComandoTimeoutSegundos = alias.ComandoTimeoutSegundos
	}
	if alias.LoginPasswordHash != "" {
		cfg.LoginPasswordHash = alias.LoginPasswordHash
	}
	cfg.EmailEnabled = alias.EmailEnabled
	if alias.EmailSmtpHost != "" {
		cfg.EmailSmtpHost = alias.EmailSmtpHost
	}
	if alias.EmailSmtpPort > 0 {
		cfg.EmailSmtpPort = alias.EmailSmtpPort
	}
	if alias.EmailUsuario != "" {
		cfg.EmailUsuario = alias.EmailUsuario
	}
	if alias.EmailPassword != "" {
		cfg.EmailPassword = alias.EmailPassword
	}
	if alias.EmailDesde != "" {
		cfg.EmailDesde = alias.EmailDesde
	}
	if alias.EmailImapHost != "" {
		cfg.EmailImapHost = alias.EmailImapHost
	}
	if alias.EmailImapPort > 0 {
		cfg.EmailImapPort = alias.EmailImapPort
	}
	if alias.EmailImapMax > 0 {
		cfg.EmailImapMax = alias.EmailImapMax
	}
	if alias.XApiKey != "" {
		cfg.XApiKey = alias.XApiKey
	}
	if alias.XApiSecret != "" {
		cfg.XApiSecret = alias.XApiSecret
	}
	if alias.XAccessToken != "" {
		cfg.XAccessToken = alias.XAccessToken
	}
	if alias.XAccessSecret != "" {
		cfg.XAccessSecret = alias.XAccessSecret
	}
	if alias.LinkedInToken != "" {
		cfg.LinkedInToken = alias.LinkedInToken
	}
	if alias.LinkedInAuthor != "" {
		cfg.LinkedInAuthor = alias.LinkedInAuthor
	}
	if alias.LicenseKey != "" {
		cfg.LicenseKey = alias.LicenseKey
	}
	if alias.ControlURL != "" {
		cfg.ControlURL = alias.ControlURL
	}
	if alias.IdInstalacion != "" {
		cfg.IdInstalacion = alias.IdInstalacion
	}
	cfg.garantizarIdInstalacion()

	return cfg
}

// garantizarIdInstalacion genera y persiste un identificador único de esta
// instalación (para el plano de control) si todavía no existe.
func (c *Config) garantizarIdInstalacion() {
	if c.IdInstalacion != "" {
		return
	}
	aleatorio := make([]byte, 12)
	if _, err := rand.Read(aleatorio); err != nil {
		c.IdInstalacion = fmt.Sprintf("inst-%d", time.Now().UnixNano())
	} else {
		c.IdInstalacion = "inst-" + hex.EncodeToString(aleatorio)
	}
	_ = c.Save()
}

func (c *Config) Save() error {
	datos := map[string]interface{}{
		"app_name":                 c.AppName,
		"version":                  c.Version,
		"require_approval":         c.RequireApproval,
		"timeout_segundos":         int(c.Timeout.Seconds()),
		"ruta_memoria":             c.RutaMemoria,
		"max_historial_ia":         c.MaxHistorialIA,
		"modelo_ia":                c.ModeloIA,
		"ia_url":                   c.IAURL,
		"ia_api_key":               c.IAAPIKey,
		"pin_hash":                 c.PINHash,
		"workspace_root":           c.WorkspaceRoot,
		"open_weather_key":         c.OpenWeatherKey,
		"news_api_key":             c.NewsAPIKey,
		"apps":                     c.Apps,
		"comando_timeout_segundos": c.ComandoTimeoutSegundos,
		"login_password_hash":      c.LoginPasswordHash,
		"email_enabled":            c.EmailEnabled,
		"email_smtp_host":          c.EmailSmtpHost,
		"email_smtp_port":          c.EmailSmtpPort,
		"email_usuario":            c.EmailUsuario,
		"email_password":           c.EmailPassword,
		"email_desde":              c.EmailDesde,
		"email_imap_host":          c.EmailImapHost,
		"email_imap_port":          c.EmailImapPort,
		"email_imap_max":           c.EmailImapMax,
		"x_api_key":                c.XApiKey,
		"x_api_secret":             c.XApiSecret,
		"x_access_token":           c.XAccessToken,
		"x_access_secret":          c.XAccessSecret,
		"linkedin_token":           c.LinkedInToken,
		"linkedin_author":          c.LinkedInAuthor,
		"license_key":              c.LicenseKey,
		"control_url":              c.ControlURL,
		"id_instalacion":           c.IdInstalacion,
	}

	contenido, err := json.MarshalIndent(datos, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.RutaConfig)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	return os.WriteFile(c.RutaConfig, contenido, 0o600)
}
