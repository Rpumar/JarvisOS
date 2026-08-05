package agents

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestNuevoPlan(t *testing.T) {
	plan := NuevoPlan("hacé X", []string{"objetivo", "paso 1", "paso 2"})
	if plan.Objetivo != "objetivo" {
		t.Errorf("Objetivo = %q, esperaba 'objetivo'", plan.Objetivo)
	}
	if len(plan.Pasos) != 2 {
		t.Fatalf("Pasos = %d, esperaba 2", len(plan.Pasos))
	}
	if plan.Pasos[0].Descripcion != "paso 1" || plan.Pasos[0].Estado != PasoPendiente {
		t.Errorf("primer paso incorrecto: %+v", plan.Pasos[0])
	}
	if plan.Completado {
		t.Error("un plan nuevo no debería estar completado")
	}

	planSolo := NuevoPlan("p", []string{"único"})
	if len(planSolo.Pasos) != 0 {
		t.Errorf("un solo elemento no debería generar pasos, fue %d", len(planSolo.Pasos))
	}
}

func TestGestorPlanGuardarYCargar(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)

	plan := NuevoPlan("petición", []string{"obj", "paso"})
	plan.ID = "test-1"
	if err := g.Guardar(plan); err != nil {
		t.Fatalf("Guardar: %v", err)
	}

	cargado, err := g.Cargar("test-1")
	if err != nil {
		t.Fatalf("Cargar: %v", err)
	}
	if cargado.Objetivo != "obj" || len(cargado.Pasos) != 1 {
		t.Errorf("carga incorrecta: %+v", cargado)
	}
	if cargado.PeticionOriginal != "petición" {
		t.Errorf("PeticionOriginal = %q", cargado.PeticionOriginal)
	}
}

func TestGestorPlanCargarInexistente(t *testing.T) {
	g := NuevoGestorPlan(t.TempDir())
	if _, err := g.Cargar("no-existe"); err == nil {
		t.Error("Cargar de un plan inexistente debería devolver error")
	}
}

func TestGestorPlanCargarCorrupto(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)
	escribir(t, dir, "roto.json", "{no es json")
	if _, err := g.Cargar("roto"); err == nil {
		t.Error("Cargar de un plan corrupto debería devolver error")
	}
}

func TestMarcarPaso(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)
	plan := NuevoPlan("p", []string{"obj", "a", "b"})
	plan.ID = "marcar"

	g.MarcarPaso(plan, 0, PasoCompletado, "resultado", "")
	if plan.Pasos[0].Estado != PasoCompletado || plan.Pasos[0].Resultado != "resultado" {
		t.Errorf("MarcarPaso no aplicó el estado: %+v", plan.Pasos[0])
	}

	g.MarcarPaso(plan, 5, PasoFallido, "", "fuera de rango") // no debe panickear
	g.MarcarPaso(plan, -1, PasoFallido, "", "negativo")

	cargado, err := g.Cargar("marcar")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if cargado.Pasos[0].Estado != PasoCompletado {
		t.Error("MarcarPaso debería persistir el estado en disco")
	}
}

func TestSiguientePaso(t *testing.T) {
	g := NuevoGestorPlan(t.TempDir())
	plan := NuevoPlan("p", []string{"obj", "a", "b", "c"})
	plan.Pasos[0].Estado = PasoCompletado

	idx, paso := g.SiguientePaso(plan)
	if idx != 1 || paso.Descripcion != "b" {
		t.Errorf("SiguientePaso = (%d, %q), esperaba (1, b)", idx, paso.Descripcion)
	}

	plan.Pasos[1].Estado = PasoCompletado
	plan.Pasos[2].Estado = PasoFallido
	idx2, _ := g.SiguientePaso(plan)
	if idx2 != -1 {
		t.Errorf("sin pasos pendientes SiguientePaso = %d, esperaba -1", idx2)
	}
}

func TestRegistrarArchivoDeduplica(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)
	plan := NuevoPlan("p", []string{"obj"})
	plan.ID = "archivos"

	g.RegistrarArchivo(plan, "a.go")
	g.RegistrarArchivo(plan, "a.go")
	g.RegistrarArchivo(plan, "b.go")

	if len(plan.ArchivosTocados) != 2 {
		t.Errorf("ArchivosTocados = %v, esperaba 2 únicos", plan.ArchivosTocados)
	}
	cargado, _ := g.Cargar("archivos")
	if len(cargado.ArchivosTocados) != 2 {
		t.Error("los archivos tocados deberían persistirse")
	}
}

func TestCompletar(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)
	plan := NuevoPlan("p", []string{"obj"})
	plan.ID = "completar"

	g.Completar(plan)
	if !plan.Completado {
		t.Error("Completar debería marcar el plan como completado")
	}
	if g.PlanPendiente() != nil {
		t.Error("un plan completado no debería estar pendiente")
	}
}

func TestPlanPendiente(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorPlan(dir)

	if g.PlanPendiente() != nil {
		t.Error("sin planes no debería haber pendiente")
	}

	completo := NuevoPlan("p", []string{"obj"})
	completo.ID = "completo"
	g.Completar(completo)

	pend := NuevoPlan("p", []string{"obj", "x"})
	pend.ID = "pend"
	if err := g.Guardar(pend); err != nil {
		t.Fatal(err)
	}

	plan := g.PlanPendiente()
	if plan == nil || plan.ID != "pend" {
		t.Fatalf("PlanPendiente = %+v, esperaba 'pend'", plan)
	}

	g.Completar(plan)
	if g.PlanPendiente() != nil {
		t.Error("tras completar el único pendiente no debería quedar nada")
	}
}

func TestPlanPendienteDirectorioInexistente(t *testing.T) {
	g := NuevoGestorPlan(filepath.Join(t.TempDir(), "no-existe"))
	if g.PlanPendiente() != nil {
		t.Error("directorio inexistente debería devolver nil")
	}
}

func TestEsPeticionProgramacion(t *testing.T) {
	positivas := []string{
		"creá un script",
		"implementá la función suma",
		"arreglá este código",
		"generá un proyecto",
		"escribí una clase",
		"refactorizá el módulo",
	}
	for _, p := range positivas {
		if !EsPeticionProgramacion(p) {
			t.Errorf("EsPeticionProgramacion no detectó: %q", p)
		}
	}

	negativas := []string{
		"qué hora es",
		"reproducí música",
		"abrí el navegador",
		"cuál es la capital de Francia",
	}
	for _, n := range negativas {
		if EsPeticionProgramacion(n) {
			t.Errorf("EsPeticionProgramacion falso positivo: %q", n)
		}
	}
}

type iaFalsaPlan struct {
	respuesta string
	err       error
}

func (i *iaFalsaPlan) Chat(system, user string) (string, error) { return i.respuesta, i.err }

func TestGenerarPlanConIA(t *testing.T) {
	ia := &iaFalsaPlan{respuesta: "OBJETIVO: Crear el módulo X\nPASOS:\n1. Leer main.go\n2. Implementar la función\n3. Ejecutar tests"}
	plan, err := GenerarPlanConIA(ia, "implementá X")
	if err != nil {
		t.Fatalf("GenerarPlanConIA: %v", err)
	}
	if plan.Objetivo != "Crear el módulo X" {
		t.Errorf("Objetivo = %q", plan.Objetivo)
	}
	if len(plan.Pasos) != 2 {
		t.Fatalf("Pasos = %d, esperaba 2", len(plan.Pasos))
	}
	if plan.Pasos[0].Descripcion != "Implementar la función" {
		t.Errorf("paso 0 = %q", plan.Pasos[0].Descripcion)
	}
	if plan.PeticionOriginal != "implementá X" {
		t.Errorf("PeticionOriginal = %q", plan.PeticionOriginal)
	}
	if plan.Contexto != ia.respuesta {
		t.Error("Contexto debería conservar la respuesta completa de la IA")
	}
}

func TestGenerarPlanConIAError(t *testing.T) {
	ia := &iaFalsaPlan{err: errors.New("red caída")}
	if _, err := GenerarPlanConIA(ia, "x"); err == nil {
		t.Error("error del conector debería propagarse")
	}
}

func TestGenerarPlanConIAObjetivoVacio(t *testing.T) {
	ia := &iaFalsaPlan{respuesta: "PASOS:\n1. Hacer algo\n2. Verificar"}
	plan, err := GenerarPlanConIA(ia, "x")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Objetivo != "Hacer algo" {
		t.Errorf("objetivo vacío debería caer al primer paso, fue %q", plan.Objetivo)
	}
	if len(plan.Pasos) != 1 {
		t.Errorf("Pasos = %d, esperaba 1", len(plan.Pasos))
	}
}

func TestGenerarPlanConIAFormatoDesordenado(t *testing.T) {
	ia := &iaFalsaPlan{respuesta: "un texto\nPASOS:\n1. Uno\n2. Dos\nOBJETIVO: Meta"}
	plan, err := GenerarPlanConIA(ia, "x")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Objetivo != "Meta" {
		t.Errorf("Objetivo = %q, esperaba Meta", plan.Objetivo)
	}
	if len(plan.Pasos) != 1 {
		t.Errorf("Pasos = %d, esperaba 1", len(plan.Pasos))
	}
}
