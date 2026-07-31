package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type EstadoPaso string

const (
	PasoPendiente EstadoPaso = "pendiente"
	PasoEjecutando EstadoPaso = "ejecutando"
	PasoCompletado EstadoPaso = "completado"
	PasoFallido    EstadoPaso = "fallido"
)

type PasoPlan struct {
	Descripcion string     `json:"descripcion"`
	Estado      EstadoPaso `json:"estado"`
	Resultado   string     `json:"resultado,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type PlanTrabajo struct {
	ID           string     `json:"id"`
	Objetivo     string     `json:"objetivo"`
	PeticionOriginal string `json:"peticion_original"`
	CreadoEl     string     `json:"creado_el"`
	ActualizadoEl string    `json:"actualizado_el"`
	Pasos        []PasoPlan `json:"pasos"`
	Contexto     string     `json:"contexto"`
	ArchivosTocados []string `json:"archivos_tocados"`
	Completado   bool       `json:"completado"`
}

func NuevoPlan(peticion string, pasos []string) *PlanTrabajo {
	plan := &PlanTrabajo{
		ID:               fmt.Sprintf("plan_%d", time.Now().Unix()),
		Objetivo:         pasos[0],
		PeticionOriginal: peticion,
		CreadoEl:         time.Now().Format(time.RFC3339),
		ActualizadoEl:    time.Now().Format(time.RFC3339),
		Completado:       false,
	}
	if len(pasos) > 1 {
		plan.Pasos = make([]PasoPlan, len(pasos)-1)
		for i, p := range pasos[1:] {
			plan.Pasos[i] = PasoPlan{Descripcion: p, Estado: PasoPendiente}
		}
	}
	return plan
}

type GestorPlan struct {
	rutaDir string
	planActual *PlanTrabajo
}

func NuevoGestorPlan(rutaDir string) *GestorPlan {
	return &GestorPlan{rutaDir: rutaDir}
}

func (g *GestorPlan) rutaPlan(id string) string {
	return filepath.Join(g.rutaDir, id+".json")
}

func (g *GestorPlan) Guardar(plan *PlanTrabajo) error {
	plan.ActualizadoEl = time.Now().Format(time.RFC3339)
	g.planActual = plan

	if err := os.MkdirAll(g.rutaDir, 0o700); err != nil {
		return fmt.Errorf("no se pudo crear directorio de planes: %w", err)
	}

	datos, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("no se pudo serializar el plan: %w", err)
	}

	ruta := g.rutaPlan(plan.ID)
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		return fmt.Errorf("no se pudo guardar el plan: %w", err)
	}

	return nil
}

func (g *GestorPlan) Cargar(id string) (*PlanTrabajo, error) {
	datos, err := os.ReadFile(g.rutaPlan(id))
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el plan: %w", err)
	}

	var plan PlanTrabajo
	if err := json.Unmarshal(datos, &plan); err != nil {
		return nil, fmt.Errorf("plan corrupto: %w", err)
	}

	return &plan, nil
}

func (g *GestorPlan) MarcarPaso(plan *PlanTrabajo, idx int, estado EstadoPaso, resultado string, errMsg string) {
	if idx < 0 || idx >= len(plan.Pasos) {
		return
	}
	plan.Pasos[idx].Estado = estado
	plan.Pasos[idx].Resultado = resultado
	plan.Pasos[idx].Error = errMsg
	g.Guardar(plan)
}

func (g *GestorPlan) SiguientePaso(plan *PlanTrabajo) (int, *PasoPlan) {
	for i, p := range plan.Pasos {
		if p.Estado == PasoPendiente {
			return i, &plan.Pasos[i]
		}
	}
	return -1, nil
}

func (g *GestorPlan) RegistrarArchivo(plan *PlanTrabajo, archivo string) {
	for _, a := range plan.ArchivosTocados {
		if a == archivo {
			return
		}
	}
	plan.ArchivosTocados = append(plan.ArchivosTocados, archivo)
	g.Guardar(plan)
}

func (g *GestorPlan) Completar(plan *PlanTrabajo) {
	plan.Completado = true
	g.Guardar(plan)
}

func (g *GestorPlan) PlanPendiente() *PlanTrabajo {
	if g.planActual != nil && !g.planActual.Completado {
		return g.planActual
	}

	entries, err := os.ReadDir(g.rutaDir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		plan, err := g.Cargar(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil || plan.Completado {
			continue
		}
		g.planActual = plan
		return plan
	}

	return nil
}

func EsPeticionProgramacion(entrada string) bool {
	claves := []string{
		"programa", "codigo", "código", "implementa", "implementá",
		"crea", "creá", "hace", "hacé", "modifica", "modificá",
		"agrega", "agregá", "refactoriza", "refactoreá",
		"arregla", "arreglá", "repara", "repará",
		"escribe", "escribí", "genera", "generá",
		"funcion", "función", "clase", "test",
		"script", "proyecto", "archivo",
	}
	e := strings.ToLower(entrada)
	for _, c := range claves {
		if strings.Contains(e, c) {
			return true
		}
	}
	return false
}

func GenerarPlanConIA(ia interface {
	Chat(system, user string) (string, error)
}, peticion string) (*PlanTrabajo, error) {
	system := `Sos un arquitecto de software. Generá un plan de trabajo paso a paso para cumplir la petición del usuario.
El plan debe ser concreto, sin código, solo pasos accionables.
Cada paso debe ser algo que se pueda ejecutar con herramientas: leer archivos, escribir código, ejecutar tests, etc.

Respondé ÚNICAMENTE en este formato:
OBJETIVO: <una línea describiendo el objetivo>
PASOS:
1. <primer paso>
2. <segundo paso>
...`

	respuesta, err := ia.Chat(system, peticion)
	if err != nil {
		return nil, fmt.Errorf("no se pudo generar el plan: %w", err)
	}

	lineas := strings.Split(respuesta, "\n")
	var objetivo string
	var pasos []string

	for _, l := range lineas {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "OBJETIVO:") {
			objetivo = strings.TrimSpace(l[len("OBJETIVO:"):])
		} else if strings.HasPrefix(l, "PASOS:") {
			continue
		} else if l != "" && (l[0] >= '1' && l[0] <= '9') && strings.Contains(l, ". ") {
			partes := strings.SplitN(l, ". ", 2)
			if len(partes) == 2 {
				pasos = append(pasos, partes[1])
			}
		}
	}

	if objetivo == "" {
		objetivo = pasos[0]
	}

	planPasos := make([]PasoPlan, len(pasos)-1)
	for i, p := range pasos[1:] {
		planPasos[i] = PasoPlan{Descripcion: p, Estado: PasoPendiente}
	}

	plan := &PlanTrabajo{
		ID:               fmt.Sprintf("plan_%d", time.Now().Unix()),
		Objetivo:         objetivo,
		PeticionOriginal: peticion,
		CreadoEl:         time.Now().Format(time.RFC3339),
		ActualizadoEl:    time.Now().Format(time.RFC3339),
		Pasos:            planPasos,
		Contexto:         respuesta,
		Completado:       false,
	}

	return plan, nil
}
