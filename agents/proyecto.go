package agents

import (
	"fmt"
	"strings"
	"time"

	"JarvisOS/core"
)

type ChatIA interface {
	Disponible() bool
	Chat(system, user string) (string, error)
}

type AgenteProyecto struct {
	ia           ChatIA
	herramientas *core.EjecutorHerramientas
	gestorPlan   *GestorPlan
	workspace    string
	historial    []core.TurnoConversacion
}

func NuevoAgenteProyecto(ia ChatIA, workspaceRoot string, gestor *GestorPlan) *AgenteProyecto {
	return &AgenteProyecto{
		ia:           ia,
		herramientas: core.NuevoEjecutorHerramientas(workspaceRoot),
		gestorPlan:   gestor,
		workspace:    workspaceRoot,
	}
}

func (a *AgenteProyecto) Disponible() bool {
	return a.ia != nil && a.ia.Disponible()
}

func (a *AgenteProyecto) Procesar(peticion string) string {
	if !a.Disponible() {
		return "La IA no está disponible, señor. Verifique que Ollama esté corriendo."
	}

	if a.gestorPlan.PlanPendiente() != nil {
		a.historial = nil
		if strings.Contains(strings.ToLower(peticion), "continuar") {
			return a.ejecutarPlan(a.gestorPlan.PlanPendiente(), nil)
		}
		if strings.Contains(strings.ToLower(peticion), "cancelar") {
			a.gestorPlan.Completar(a.gestorPlan.PlanPendiente())
			return "Plan cancelado, señor."
		}
	}

	plan, err := GenerarPlanConIA(a.ia, peticion)
	if err != nil {
		return fmt.Sprintf("No pude generar un plan: %v", err)
	}

	a.gestorPlan.Guardar(plan)
	fmt.Printf("[PLAN] Objetivo: %s\n", plan.Objetivo)
	fmt.Printf("[PLAN] %d pasos planificados.\n", len(plan.Pasos))
	return a.ejecutarPlan(plan, nil)
}

func (a *AgenteProyecto) ejecutarPlan(plan *PlanTrabajo, pasoInicial *int) string {
	systemPrompt := a.systemPrompt()
	_ = pasoInicial
	iteraciones := 0
	maxIter := 30
	pasosSinResumen := 0

	for {
		idx, paso := a.gestorPlan.SiguientePaso(plan)
		if idx < 0 {
			a.gestorPlan.Completar(plan)
			return fmt.Sprintf("Plan completado, señor. %d pasos ejecutados. Archivos modificados: %s",
				len(plan.Pasos), strings.Join(plan.ArchivosTocados, ", "))
		}

		if iteraciones >= maxIter {
			return "Se alcanzó el máximo de iteraciones. El plan está pausado, señor. Diga 'continuar plan' para retomarlo."
		}

		a.gestorPlan.MarcarPaso(plan, idx, PasoEjecutando, "", "")
		mensaje := fmt.Sprintf("Paso actual: %s\n\nEjecutá las herramientas necesarias para completar este paso. Cuando termines, explicá el resultado.", paso.Descripcion)

		respuesta, err := a.ia.Chat(systemPrompt+"\n\nContexto del plan:\n"+plan.Contexto, mensaje)
		if err != nil {
			a.gestorPlan.MarcarPaso(plan, idx, PasoFallido, "", err.Error())
			return fmt.Sprintf("Error ejecutando paso '%s': %v", paso.Descripcion, err)
		}

		if strings.Contains(respuesta, "HERRAMIENTA|") {
			herramientas := core.ParsearHerramientas(respuesta)
			var resultados []string
			for _, h := range herramientas {
				resultado := a.herramientas.Ejecutar(h)
				if resultado.Exito {
					plan.Contexto += fmt.Sprintf("\nEjecuté %s: OK", h.Nombre)
					if h.Nombre == "write_file" || h.Nombre == "edit_file" {
						a.gestorPlan.RegistrarArchivo(plan, h.Argumentos["path"])
					}
				}
				resultados = append(resultados, fmt.Sprintf("Resultado de %s: %s", h.Nombre, resultado.Salida))
			}
			a.historial = append(a.historial, core.TurnoConversacion{
				Usuario:   mensaje,
				Asistente: respuesta,
			})
			a.historial = append(a.historial, core.TurnoConversacion{
				Usuario:   strings.Join(resultados, "\n"),
				Asistente: "Continuo.",
			})

			pasosSinResumen++
			if pasosSinResumen >= 5 {
				resumen, _ := a.ia.Chat(
					"Resumí el estado actual del proyecto y lo que falta hacer en 3-4 líneas.",
					"Estado actual: "+plan.Contexto+"\nPlan: "+plan.Objetivo,
				)
				if resumen != "" {
					plan.Contexto = resumen
					a.historial = nil
				}
				pasosSinResumen = 0
			}
		} else {
			a.gestorPlan.MarcarPaso(plan, idx, PasoCompletado, respuesta, "")
			plan.Contexto += fmt.Sprintf("\nPaso '%s': %s", paso.Descripcion, respuesta)
		}

		plan.ActualizadoEl = time.Now().Format(time.RFC3339)
		a.gestorPlan.Guardar(plan)
		iteraciones++
	}
}

func (a *AgenteProyecto) systemPrompt() string {
	info := DetectarProyecto(a.workspace)
	contexto := info.PromptContexto(a.workspace)

	return fmt.Sprintf(`Sos un ingeniero de software senior integrado en JARVIS, un asistente que controla Windows.

Cuando necesites ejecutar una herramienta, usá este formato EXACTO:

HERRAMIENTA|nombre
ARGUMENTOS|{"arg1": "valor1"}
---

Herramientas:
- read_file: {"path": "..."}
- write_file: {"path": "...", "content": "..."}  — SOLO para archivos NUEVOS
- edit_file: {"path": "...", "old": "...", "new": "..."}  — USA ESTA para modificar archivos existentes
- glob: {"pattern": "**/*.go"}
- grep: {"pattern": "...", "include": "*.go"}
- run: {"command": "go build ./..."}
- read_dir: {"path": "."}
- run_test: {"command": "go test ./..."}

REGLAS ESTRICTAS:
1. Antes de hacer cualquier cambio, PRIMERO leé el archivo que vas a modificar.
2. Si el cambio es grande o riesgoso (borrar código, refactor mayor), preguntá primero con "leer_entrada".
3. Preferí edit_file sobre write_file para archivos existentes: así solo cambiás lo necesario.
4. Después de cada cambio, ejecutá los tests para verificar que no rompiste nada.
5. Si los tests fallan, leé el error, corregí, y volvé a ejecutar.

Respondé SIEMPRE en español argentino, tratando al usuario de "señor".
Si no necesitás herramientas, respondé normal sin formato especial.

	%s`, contexto)
}

func (a *AgenteProyecto) SetRespuestaUsuario(respuesta string) {
	a.historial = append(a.historial, core.TurnoConversacion{Usuario: respuesta, Asistente: ""})
}

func (a *AgenteProyecto) Reset() {
	a.historial = nil
}

func (a *AgenteProyecto) TieneTareaPendiente() bool {
	return a.gestorPlan.PlanPendiente() != nil
}

func (a *AgenteProyecto) PlanPendienteDescripcion() string {
	plan := a.gestorPlan.PlanPendiente()
	if plan == nil {
		return ""
	}
	return plan.Objetivo
}

func (a *AgenteProyecto) ContinuarPlan() string {
	plan := a.gestorPlan.PlanPendiente()
	if plan == nil {
		return "No hay ningún plan pendiente, señor."
	}
	return a.ejecutarPlan(plan, nil)
}

func (a *AgenteProyecto) CancelarPlan() string {
	plan := a.gestorPlan.PlanPendiente()
	if plan == nil {
		return "No hay ningún plan pendiente, señor."
	}
	a.gestorPlan.Completar(plan)
	return "Plan cancelado, señor."
}
