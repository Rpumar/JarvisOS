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
		modelo = "qwen2.5-coder:7b"
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
	if !c.disponible {
		return "", "", fmt.Errorf("no hay IA disponible (%s no responde)", c.baseURL)
	}
	respuesta, err := c.chatConSistema(promptSistemaCodigo, peticion)
	if err != nil {
		return "", "", err
	}
	return parsearRespuestaCodigo(respuesta)
}

const promptSistemaDesarrollo = `Sos un ingeniero de software senior fullstack (backend, frontend, base de datos y DevOps), que trabaja como desarrollador de JarvisOS para su usuario.
El proyecto en el que trabajás es una aplicación web con stack fijo:
- Backend: Go usando SOLO la librería estándar (net/http, encoding/json, sync, time). Nada de frameworks ni dependencias externas.
- Frontend: HTML, CSS y JavaScript puro, sin frameworks ni CDN. El estilo actual es un panel oscuro (verde/cian sobre fondo #0b0f17).
- La app sirve el frontend desde la carpeta "frontend" y expone una API JSON bajo /api.
Reglas estrictas, sin excepciones:
- Generá UN SOLO archivo, nuevo o a modificar, que implemente la mejora pedida de forma completa y compilable.
- Solo se permiten archivos dentro del proyecto: main.go, frontend/index.html, frontend/style.css o frontend/app.js.
- No generes código que borre archivos, ejecute comandos del sistema, acceda a internet, o introduzca dependencias.
- Si la petición incluye un bloque [INSTRUCCIONES DE SKILL] o instrucciones de proyecto, seguilas por encima de estas reglas generales (salvo las de seguridad, que nunca se violan).
- Si te pasan un error de compilación y el archivo actual, corregilo y devolvé el archivo completo, nunca un parche o diff.
- Si la petición es peligrosa o fuera del alcance del proyecto, dejá CONTENIDO vacío y explicá por qué.
Respondé ÚNICAMENTE en este formato exacto, sin texto adicional antes ni después:
ARCHIVO:
<ruta relativa del archivo a escribir o reemplazar, ej: main.go o frontend/app.js>
CONTENIDO:
<el código completo del archivo>
EXPLICACION:
<resumen breve en español de qué hace el cambio y cómo se prueba>`

func (c *Conector) ConsultarDesarrollo(peticion string) (respuesta string, explicacion string, err error) {
	if !c.disponible {
		return "", "", fmt.Errorf("no hay IA disponible (%s no responde)", c.baseURL)
	}
	codigo, err := c.chatConSistema(promptSistemaDesarrollo, peticion)
	if err != nil {
		return "", "", err
	}
	return codigo, extraerExplicacionDesarrollo(codigo), nil
}

func extraerExplicacionDesarrollo(texto string) string {
	idx := strings.Index(texto, "EXPLICACION:")
	if idx == -1 {
		return strings.TrimSpace(texto)
	}
	return strings.TrimSpace(texto[idx+len("EXPLICACION:"):])
}

func (c *Conector) chatConSistema(system, user string) (string, error) {
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
