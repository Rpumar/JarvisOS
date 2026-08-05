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

const urlOllamaV1 = "http://localhost:11434/v1"

type Conector struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	modelo     string
	disponible bool
}

func NuevoConector(modelo string, timeout time.Duration, baseURL, apiKey string) *Conector {
	if modelo == "" {
		modelo = "mistral:latest"
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = urlOllamaV1
	}
	c := &Conector{
		httpClient: &http.Client{Timeout: timeout},
		modelo:     modelo,
		baseURL:    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
	}
	c.probe()
	return c
}

func (c *Conector) probe() {
	cliente := &http.Client{Timeout: 5 * time.Second}
	resp, err := cliente.Get(c.baseURL + "/models")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return
	}
	c.disponible = true
}

func (c *Conector) Disponible() bool {
	return c.disponible
}

func (c *Conector) Modelo() string {
	return c.modelo
}

type mensajeChat struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type peticionChat struct {
	Model    string        `json:"model"`
	Messages []mensajeChat `json:"messages"`
	Stream   bool          `json:"stream"`
}

type respuestaChat struct {
	Choices []struct {
		Message mensajeChat `json:"message"`
	} `json:"choices"`
}

func (c *Conector) Consultar(prompt string, historial []core.TurnoConversacion) (string, error) {
	if !c.disponible {
		return "", fmt.Errorf("no hay IA disponible (%s no responde)", c.baseURL)
	}
	return c.consultarIA(prompt, historial)
}

func (c *Conector) Chat(system, user string) (string, error) {
	if !c.disponible {
		return "", fmt.Errorf("la IA no está disponible")
	}
	cuerpo := peticionChat{
		Model: c.modelo,
		Messages: []mensajeChat{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: false,
	}
	datosJSON, err := json.Marshal(cuerpo)
	if err != nil {
		return "", fmt.Errorf("error al preparar la peticion: %w", err)
	}
	return c.enviarChat(datosJSON)
}

func (c *Conector) consultarIA(prompt string, historial []core.TurnoConversacion) (string, error) {
	cuerpo := peticionChat{
		Model:    c.modelo,
		Messages: buildMensajes(prompt, historial),
		Stream:   false,
	}
	datosJSON, err := json.Marshal(cuerpo)
	if err != nil {
		return "", fmt.Errorf("error al preparar la peticion: %w", err)
	}
	return c.enviarChat(datosJSON)
}

func (c *Conector) enviarChat(datosJSON []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(datosJSON))
	if err != nil {
		return "", fmt.Errorf("error al construir la peticion HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al contactar la IA: %w", err)
	}
	defer resp.Body.Close()

	cuerpoResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer la respuesta: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("la IA respondió %d: %s", resp.StatusCode, strings.TrimSpace(string(cuerpoResp)))
	}

	var r respuestaChat
	if err := json.Unmarshal(cuerpoResp, &r); err != nil {
		return "", fmt.Errorf("error al interpretar la respuesta: %w", err)
	}
	if len(r.Choices) == 0 || r.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("la IA no devolvió contenido")
	}

	return r.Choices[0].Message.Content, nil
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
