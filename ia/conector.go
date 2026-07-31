package ia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"JarvisOS/core"
)

const endpointOllama = "http://localhost:11434/api/chat"

type Conector struct {
	httpClient *http.Client
	modelo     string
	ollama     bool
}

func NuevoConector(modelo string, timeout time.Duration) *Conector {
	if modelo == "" {
		modelo = "qwen2.5-coder:7b"
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	c := &Conector{
		httpClient: &http.Client{Timeout: timeout},
		modelo:     modelo,
	}
	pollOllama(c)
	return c
}

func pollOllama(c *Conector) {
	resp, err := c.httpClient.Get("http://localhost:11434/api/tags")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		c.ollama = true
	}
}

func (c *Conector) Disponible() bool {
	return c.ollama
}

func (c *Conector) Modelo() string {
	return c.modelo
}

type mensajeChat struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type peticionOllamaChat struct {
	Model    string         `json:"model"`
	Messages []mensajeChat  `json:"messages"`
	Stream   bool           `json:"stream"`
}

type respuestaOllamaChat struct {
	Message mensajeChat `json:"message"`
	Done    bool        `json:"done"`
}

func (c *Conector) Consultar(prompt string, historial []core.TurnoConversacion) (string, error) {
	if !c.ollama {
		return "", fmt.Errorf("no hay IA disponible (Ollama no esta corriendo)")
	}
	return c.consultarOllama(prompt, historial)
}

func (c *Conector) Chat(system, user string) (string, error) {
	if !c.ollama {
		return "", fmt.Errorf("Ollama no disponible")
	}

	mensajes := []mensajeChat{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	cuerpo := peticionOllamaChat{
		Model:    c.modelo,
		Messages: mensajes,
		Stream:   false,
	}

	datosJSON, _ := json.Marshal(cuerpo)
	req, _ := http.NewRequest(http.MethodPost, endpointOllama, bytes.NewReader(datosJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al contactar Ollama: %w", err)
	}
	defer resp.Body.Close()

	cuerpoResp, _ := io.ReadAll(resp.Body)
	var r respuestaOllamaChat
	if err := json.Unmarshal(cuerpoResp, &r); err != nil {
		return "", fmt.Errorf("error al interpretar respuesta: %w", err)
	}

	if r.Message.Content == "" {
		return "", fmt.Errorf("Ollama no devolvió contenido")
	}

	return r.Message.Content, nil
}

func (c *Conector) consultarOllama(prompt string, historial []core.TurnoConversacion) (string, error) {
	mensajes := buildMensajes(prompt, historial)

	cuerpo := peticionOllamaChat{
		Model:    c.modelo,
		Messages: mensajes,
		Stream:   false,
	}

	datosJSON, err := json.Marshal(cuerpo)
	if err != nil {
		return "", fmt.Errorf("error al preparar la peticion: %w", err)
	}

	peticion, err := http.NewRequest(http.MethodPost, endpointOllama, bytes.NewReader(datosJSON))
	if err != nil {
		return "", fmt.Errorf("error al construir la peticion HTTP: %w", err)
	}
	peticion.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(peticion)
	if err != nil {
		return "", fmt.Errorf("error al contactar Ollama: %w", err)
	}
	defer resp.Body.Close()

	cuerpoResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer la respuesta de Ollama: %w", err)
	}

	var respuesta respuestaOllamaChat
	if err := json.Unmarshal(cuerpoResp, &respuesta); err != nil {
		return "", fmt.Errorf("error al interpretar la respuesta de Ollama: %w", err)
	}

	if respuesta.Message.Content == "" {
		return "", fmt.Errorf("Ollama no devolvio contenido")
	}

	return respuesta.Message.Content, nil
}

func buildMensajes(prompt string, historial []core.TurnoConversacion) []mensajeChat {
	systemContent := `Eres JARVIS, el asistente de voz personal de tu usuario. Personalidad: seguro de vos mismo, con ingenio seco y calidez, siempre competente -nunca payaso ni exagerado-. Tratás al usuario de 'señor' de forma cercana, no fría. Respondé siempre en español, en una o dos frases como máximo: tu respuesta se lee en voz alta, así que la brevedad importa más que el ingenio. IMPORTANTE: Nunca le digas al usuario que borre, elimine, desinstale, descargue, instale, compre o venda nada. Si te pregunta cómo hacer algo peligroso, decile que no puedes ayudar con eso.`

	mensajes := []mensajeChat{
		{Role: "system", Content: systemContent},
	}
	for _, turno := range historial {
		mensajes = append(mensajes,
			mensajeChat{Role: "user", Content: turno.Usuario},
			mensajeChat{Role: "assistant", Content: turno.Asistente},
		)
	}
	mensajes = append(mensajes, mensajeChat{Role: "user", Content: prompt})

	return mensajes
}

const promptSistemaCodigo = `Sos un generador de scripts de PowerShell para Windows, usado por un asistente de voz local llamado JarvisOS.
Reglas estrictas, sin excepciones:
- Generá ÚNICAMENTE scripts de PowerShell simples y seguros para tareas cotidianas de automatización (organizar archivos, leer información del sistema, tareas repetitivas, cálculos).
- NUNCA generes código que: formatee discos, borre carpetas amplias o del sistema, deshabilite seguridad o el firewall, modifique el registro de Windows, descargue o ejecute archivos de internet, apague o reinicie el equipo, o cree/modifique usuarios y permisos.
- NUNCA generes código que instale, desinstale, descargue, compre, venda, o modifique programas o paquetes del sistema.
- NUNCA generes código que elimine archivos, carpetas o datos del usuario sin su confirmación explícita.
- Si la petición es peligrosa, ambigua, o pide algo fuera de este alcance, dejá CODIGO completamente vacío y explicá por qué en EXPLICACION.
Respondé ÚNICAMENTE en este formato exacto, sin texto adicional antes ni después:
CODIGO:
<el script de PowerShell, o nada si no corresponde>
EXPLICACION:
<una explicación breve en español, en una o dos frases, de qué hace el script o por qué no se generó>`

func (c *Conector) ConsultarCodigo(peticion string) (codigo string, explicacion string, err error) {
	if !c.ollama {
		return "", "", fmt.Errorf("no hay IA disponible (Ollama no esta corriendo)")
	}

	cuerpo := peticionOllamaChat{
		Model: c.modelo,
		Messages: []mensajeChat{
			{Role: "system", Content: promptSistemaCodigo},
			{Role: "user", Content: peticion},
		},
		Stream: false,
	}
	datosJSON, err := json.Marshal(cuerpo)
	if err != nil {
		return "", "", fmt.Errorf("error al preparar la peticion: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpointOllama, bytes.NewReader(datosJSON))
	if err != nil {
		return "", "", fmt.Errorf("error al construir la peticion HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error al contactar Ollama: %w", err)
	}
	defer resp.Body.Close()
	cuerpoResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error al leer la respuesta: %w", err)
	}
	var r respuestaOllamaChat
	if err := json.Unmarshal(cuerpoResp, &r); err != nil {
		return "", "", fmt.Errorf("error al interpretar respuesta de Ollama: %w", err)
	}
	return parsearRespuestaCodigo(r.Message.Content)
}

func parsearRespuestaCodigo(texto string) (codigo string, explicacion string, err error) {
	idxCodigo := strings.Index(texto, "CODIGO:")
	idxExplicacion := strings.Index(texto, "EXPLICACION:")

	if idxCodigo == -1 || idxExplicacion == -1 || idxExplicacion < idxCodigo {
		return "", strings.TrimSpace(texto), nil
	}

	codigo = strings.TrimSpace(texto[idxCodigo+len("CODIGO:") : idxExplicacion])
	explicacion = strings.TrimSpace(texto[idxExplicacion+len("EXPLICACION:"):])
	return codigo, explicacion, nil
}
