package config

import (
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
	ModeloVoz       string        `json:"modelo_voz"`
	RutaMemoria     string        `json:"ruta_memoria"`
	RutaConfig      string        `json:"-"`

	ContinuousListening bool     `json:"continuous_listening"`
	WakeWords           []string `json:"wake_words"`
	TTSVoice            string   `json:"tts_voice"`
	TTSRate             int      `json:"tts_rate"`
	MaxHistorialIA      int      `json:"max_historial_ia"`

	ModeloIA       string `json:"modelo_ia"`
	IAURL          string `json:"ia_url"`
	IAAPIKey       string `json:"ia_api_key"`
	PINHash        string `json:"pin_hash"`
	WorkspaceRoot  string `json:"workspace_root"`
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
}

func defaultConfig() *Config {
	return &Config{
		AppName:             "JARVISOS",
		Version:             "0.14.0",
		RequireApproval:     true,
		Timeout:             30 * time.Second,
		ModeloVoz:           "./modelo-voz-es",
		RutaMemoria:         filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "memoria.json"),
		ContinuousListening: false,
		WakeWords:           []string{"jarvis"},
		TTSVoice:            "",
		TTSRate:             0,
		MaxHistorialIA:      20,
		ModeloIA:            "qwen2.5-coder:7b",
		IAURL:               "",
		IAAPIKey:            "",
		WorkspaceRoot:       filepath.Join(os.Getenv("USERPROFILE"), "Desktop"),
		OpenWeatherKey:      "",
		NewsAPIKey:          "",
		ComandoTimeoutSegundos: 30,
		EmailEnabled:           false,
		EmailSmtpPort:          587,
		EmailDesde:             "Jarvis",
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
	cfg.RutaConfig = filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "config.json")

	contenido, err := os.ReadFile(cfg.RutaConfig)
	if err != nil {
		if os.IsNotExist(err) {
			_ = cfg.Save()
			return cfg
		}
		fmt.Fprintf(os.Stderr, "[ADVERTENCIA] No se pudo leer config: %v. Usando valores por defecto.\n", err)
		return cfg
	}

	type configAlias struct {
		AppName             string  `json:"app_name"`
		Version             string  `json:"version"`
		RequireApproval     bool    `json:"require_approval"`
		TimeoutSegundos     int     `json:"timeout_segundos"`
		ModeloVoz           string  `json:"modelo_voz"`
		RutaMemoria         string  `json:"ruta_memoria"`
		ContinuousListening bool    `json:"continuous_listening"`
		WakeWords           []string `json:"wake_words"`
		TTSVoice            string  `json:"tts_voice"`
		TTSRate             int     `json:"tts_rate"`
		MaxHistorialIA      int     `json:"max_historial_ia"`
		ModeloIA            string  `json:"modelo_ia"`
		IAURL               string  `json:"ia_url"`
		IAAPIKey            string  `json:"ia_api_key"`
		PINHash             string  `json:"pin_hash"`
		WorkspaceRoot       string  `json:"workspace_root"`
		OpenWeatherKey      string  `json:"open_weather_key"`
		NewsAPIKey          string  `json:"news_api_key"`
		Apps                map[string]string `json:"apps"`
		ComandoTimeoutSegundos int   `json:"comando_timeout_segundos"`
		LoginPasswordHash    string `json:"login_password_hash"`
		EmailEnabled         bool   `json:"email_enabled"`
		EmailSmtpHost        string `json:"email_smtp_host"`
		EmailSmtpPort        int    `json:"email_smtp_port"`
		EmailUsuario         string `json:"email_usuario"`
		EmailPassword        string `json:"email_password"`
		EmailDesde           string `json:"email_desde"`
	}

	var alias configAlias
	if err := json.Unmarshal(contenido, &alias); err != nil {
		fmt.Fprintf(os.Stderr, "[ADVERTENCIA] Config.json corrupto: %v. Usando valores por defecto.\n", err)
		return cfg
	}

	if alias.AppName != "" { cfg.AppName = alias.AppName }
	if alias.Version != "" { cfg.Version = alias.Version }
	cfg.RequireApproval = alias.RequireApproval
	if alias.TimeoutSegundos > 0 { cfg.Timeout = time.Duration(alias.TimeoutSegundos) * time.Second }
	if alias.ModeloVoz != "" { cfg.ModeloVoz = alias.ModeloVoz }
	if alias.RutaMemoria != "" { cfg.RutaMemoria = alias.RutaMemoria }
	cfg.ContinuousListening = alias.ContinuousListening
	if len(alias.WakeWords) > 0 { cfg.WakeWords = alias.WakeWords }
	if alias.TTSVoice != "" { cfg.TTSVoice = alias.TTSVoice }
	cfg.TTSRate = alias.TTSRate
	if alias.MaxHistorialIA > 0 { cfg.MaxHistorialIA = alias.MaxHistorialIA }
	if alias.ModeloIA != "" { cfg.ModeloIA = alias.ModeloIA }
	if alias.IAURL != "" { cfg.IAURL = alias.IAURL }
	if alias.IAAPIKey != "" { cfg.IAAPIKey = alias.IAAPIKey }
	if alias.PINHash != "" { cfg.PINHash = alias.PINHash }
	if alias.WorkspaceRoot != "" { cfg.WorkspaceRoot = alias.WorkspaceRoot }
	if alias.OpenWeatherKey != "" { cfg.OpenWeatherKey = alias.OpenWeatherKey }
	if alias.NewsAPIKey != "" { cfg.NewsAPIKey = alias.NewsAPIKey }
	if alias.Apps != nil { cfg.Apps = alias.Apps }
	if alias.ComandoTimeoutSegundos > 0 { cfg.ComandoTimeoutSegundos = alias.ComandoTimeoutSegundos }
	if alias.LoginPasswordHash != "" { cfg.LoginPasswordHash = alias.LoginPasswordHash }
	cfg.EmailEnabled = alias.EmailEnabled
	if alias.EmailSmtpHost != "" { cfg.EmailSmtpHost = alias.EmailSmtpHost }
	if alias.EmailSmtpPort > 0 { cfg.EmailSmtpPort = alias.EmailSmtpPort }
	if alias.EmailUsuario != "" { cfg.EmailUsuario = alias.EmailUsuario }
	if alias.EmailPassword != "" { cfg.EmailPassword = alias.EmailPassword }
	if alias.EmailDesde != "" { cfg.EmailDesde = alias.EmailDesde }

	return cfg
}

func (c *Config) Save() error {
	datos := map[string]interface{}{
		"app_name":               c.AppName,
		"version":                c.Version,
		"require_approval":       c.RequireApproval,
		"timeout_segundos":       int(c.Timeout.Seconds()),
		"modelo_voz":             c.ModeloVoz,
		"ruta_memoria":           c.RutaMemoria,
		"continuous_listening":   c.ContinuousListening,
		"wake_words":             c.WakeWords,
		"tts_voice":              c.TTSVoice,
		"tts_rate":               c.TTSRate,
		"max_historial_ia":       c.MaxHistorialIA,
		"modelo_ia":              c.ModeloIA,
		"ia_url":                 c.IAURL,
		"ia_api_key":             c.IAAPIKey,
		"pin_hash":               c.PINHash,
		"workspace_root":         c.WorkspaceRoot,
		"open_weather_key":       c.OpenWeatherKey,
		"news_api_key":           c.NewsAPIKey,
		"apps":                   c.Apps,
		"comando_timeout_segundos": c.ComandoTimeoutSegundos,
		"login_password_hash":      c.LoginPasswordHash,
		"email_enabled":            c.EmailEnabled,
		"email_smtp_host":          c.EmailSmtpHost,
		"email_smtp_port":          c.EmailSmtpPort,
		"email_usuario":            c.EmailUsuario,
		"email_password":           c.EmailPassword,
		"email_desde":              c.EmailDesde,
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
