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

	OpenWeatherKey string `json:"open_weather_key"`
	NewsAPIKey     string `json:"news_api_key"`
	GoogleCalendar bool   `json:"google_calendar_enabled"`
	SpotifyEnabled bool   `json:"spotify_enabled"`

	BrilloPaso int `json:"brillo_paso"`
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
		OpenWeatherKey:      "",
		NewsAPIKey:          "",
		GoogleCalendar:      false,
		SpotifyEnabled:      false,
		BrilloPaso:          20,
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
		OpenWeatherKey      string  `json:"open_weather_key"`
		NewsAPIKey          string  `json:"news_api_key"`
		GoogleCalendar      bool    `json:"google_calendar_enabled"`
		SpotifyEnabled      bool    `json:"spotify_enabled"`
		BrilloPaso          int     `json:"brillo_paso"`
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
	if alias.OpenWeatherKey != "" { cfg.OpenWeatherKey = alias.OpenWeatherKey }
	if alias.NewsAPIKey != "" { cfg.NewsAPIKey = alias.NewsAPIKey }
	cfg.GoogleCalendar = alias.GoogleCalendar
	cfg.SpotifyEnabled = alias.SpotifyEnabled
	if alias.BrilloPaso > 0 { cfg.BrilloPaso = alias.BrilloPaso }

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
		"open_weather_key":       c.OpenWeatherKey,
		"news_api_key":           c.NewsAPIKey,
		"google_calendar_enabled": c.GoogleCalendar,
		"spotify_enabled":        c.SpotifyEnabled,
		"brillo_paso":            c.BrilloPaso,
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
