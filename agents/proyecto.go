package agents

import (
	"fmt"
	"strings"
	"time"

	"JarvisOS/core"
	"JarvisOS/ia"
)

type AgenteProyecto struct {
	claude    *ia.ClienteClaude
	herramientas *core.EjecutorHerramientas
	workspace string
	historial []ia.TurnoClaude
}

func NuevoAgenteProyecto(claude *ia.ClienteClaude, workspaceRoot string) *AgenteProyecto {
	return &AgenteProyecto{
		claude:       claude,
		herramientas: core.NuevoEjecutorHerramientas(workspaceRoot),
		workspace:    workspaceRoot,
	}
}

func (a *AgenteProyecto) Disponible() bool {
	return a.claude.Disponible()
}

func (a *AgenteProyecto) Procesar(peticion string) string {
	if !a.Disponible() {
		return "El agente de proyecto no está disponible, señor. Configure Claude API en config.json."
	}

	a.historial = append(a.historial, ia.TurnoClaude{Role: "user", Content: peticion})
	systemPrompt := ia.ClaudeSystemPrompt(a.workspace)

	maxIntentos := 10
	for intento := 0; intento < maxIntentos; intento++ {
		respuesta, err := a.claude.Charla(systemPrompt, a.historial)
		if err != nil {
			return fmt.Sprintf("Error al consultar a Claude: %v", err)
		}

		a.historial = append(a.historial, ia.TurnoClaude{Role: "assistant", Content: respuesta})

		if !strings.Contains(respuesta, "HERRAMIENTA|") {
			if len(a.historial) > 20 {
				a.historial = a.historial[len(a.historial)-10:]
			}
			return respuesta
		}

		herramientas := core.ParsearHerramientas(respuesta)
		if len(herramientas) == 0 {
			return respuesta
		}

		var resultados []string
		for _, h := range herramientas {
			resultado := a.herramientas.Ejecutar(h)

			if h.Nombre == "leer_entrada" {
				return fmt.Sprintf("Necesito preguntarle algo, señor: %s", h.Argumentos["pregunta"])
			}

			linea := fmt.Sprintf("Resultado de %s: %s", h.Nombre, resultado.Salida)
			resultados = append(resultados, linea)
		}

		mensajeResultado := strings.Join(resultados, "\n\n")
		a.historial = append(a.historial, ia.TurnoClaude{Role: "user", Content: mensajeResultado})
	}

	return "Se alcanzó el máximo de iteraciones. La tarea podría estar incompleta, señor."
}

func (a *AgenteProyecto) SetRespuestaUsuario(respuesta string) {
	a.historial = append(a.historial, ia.TurnoClaude{Role: "user", Content: respuesta})
}

func (a *AgenteProyecto) Reset() {
	a.historial = nil
}

func (a *AgenteProyecto) TieneTareaPendiente() bool {
	if len(a.historial) == 0 {
		return false
	}
	ultimo := a.historial[len(a.historial)-1]
	return ultimo.Role == "assistant" && strings.Contains(ultimo.Content, "leer_entrada")
}

type IngAgente interface {
	Disponible() bool
	Procesar(peticion string) string
	SetRespuestaUsuario(respuesta string)
	Reset()
	TieneTareaPendiente() bool
}

var _ IngAgente = (*AgenteProyecto)(nil)

// MonitorTarea ejecuta una tarea con timeout en una goroutine separada.
func (a *AgenteProyecto) MonitorTarea(peticion string, timeout time.Duration) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		resultado := a.Procesar(peticion)
		ch <- resultado
	}()
	return ch
}
