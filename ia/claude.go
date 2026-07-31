package ia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const endpointClaude = "https://api.anthropic.com/v1/messages"
const modeloClaude = "claude-sonnet-4-20250514"

type ClienteClaude struct {
	apiKey  string
	timeout time.Duration
}

func NuevoClienteClaude(apiKey string, timeout time.Duration) *ClienteClaude {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &ClienteClaude{apiKey: apiKey, timeout: timeout}
}

func (c *ClienteClaude) Disponible() bool {
	return c.apiKey != ""
}

type mensajeClaude struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type peticionClaude struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []mensajeClaude `json:"messages"`
}

type respuestaClaude struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *ClienteClaude) Consultar(system string, messages []mensajeClaude) (string, error) {
	if !c.Disponible() {
		return "", fmt.Errorf("Claude no configurado (falta api_key)")
	}

	cuerpo := peticionClaude{
		Model:     modeloClaude,
		MaxTokens: 4096,
		System:    system,
		Messages:  messages,
	}

	datosJSON, _ := json.Marshal(cuerpo)

	req, _ := http.NewRequest("POST", endpointClaude, bytes.NewReader(datosJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al contactar Claude: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var r respuestaClaude
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("error al interpretar respuesta de Claude: %w", err)
	}

	if r.Error.Message != "" {
		return "", fmt.Errorf("Claude API error: %s", r.Error.Message)
	}

	if len(r.Content) == 0 {
		return "", fmt.Errorf("Claude no devolvió contenido")
	}

	return r.Content[0].Text, nil
}

type TurnoClaude struct {
	Role    string
	Content string
}

func ClaudeSystemPrompt(workspaceRoot string) string {
	return fmt.Sprintf(`Eres un ingeniero de software senior integrado en JARVIS, un asistente que controla el sistema operativo Windows del usuario.

Tus capacidades como ingeniero son:
- Leer y analizar archivos de código fuente
- Escribir y modificar archivos
- Buscar patrones en el código
- Ejecutar comandos de terminal (build, test, etc.)
- Gestionar proyectos de software completos

REGLAS ESTRICTAS:
1. Analizá el contexto antes de actuar. Siempre entendé el proyecto primero.
2. Tus cambios deben ser mínimos y precisos.
3. Nunca ejecutes comandos destructivos (rm -rf, del /s, format, diskpart, shutdown).
4. Respondé SIEMPRE en español argentino, tratando al usuario de "señor".
5. Cuando necesites ejecutar una herramienta, usá este formato EXACTO:

HERRAMIENTA|nombre
ARGUMENTOS|{"arg1": "valor1"}
---
HERRAMIENTA|nombre2
ARGUMENTOS|{"arg1": "valor1"}
---

Herramientas disponibles:
- read_file: Lee un archivo. Args: {"path": "ruta/al/archivo"}
- write_file: Escribe un archivo. Args: {"path": "ruta/al/archivo", "content": "contenido"}
- edit_file: Edita un archivo (replace). Args: {"path": "...", "old": "texto a reemplazar", "new": "texto nuevo"}
- glob: Busca archivos por patrón. Args: {"pattern": "**/*.go"}
- grep: Busca contenido en archivos. Args: {"pattern": "func.*main", "include": "*.go"}
- run: Ejecuta un comando. Args: {"command": "go build ./..."}
- read_dir: Lista un directorio. Args: {"path": "."}
- run_test: Ejecuta tests. Args: {"command": "go test ./..."}
- leer_entrada: Pregunta algo al usuario. Args: {"pregunta": "..."}

Cuando termines una tarea, respondé sin formato de herramienta, solo texto normal explicando qué hiciste.

Directorio del proyecto: %s

IMPORTANTE: No uses el formato de herramienta para responder normalmente. Solo usalo cuando necesites leer/escribir archivos o ejecutar comandos. Si es una respuesta normal (explicar, conversar, etc.), respondé sin formato especial.`, workspaceRoot)
}

func (c *ClienteClaude) Charla(system string, historial []TurnoClaude) (string, error) {
	mensajes := make([]mensajeClaude, 0, len(historial))
	for _, t := range historial {
		mensajes = append(mensajes, mensajeClaude{Role: t.Role, Content: t.Content})
	}
	return c.Consultar(system, mensajes)
}
