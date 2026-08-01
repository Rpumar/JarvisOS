package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGestorTareas(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorTareas(filepath.Join(dir, "tareas.json"))

	t1 := g.Agregar("enviar el informe", "al gerente", "mañana")
	t2 := g.Agregar("revisar presupuesto", "", "")
	if t1.ID >= t2.ID {
		t.Fatalf("los IDs deben incrementarse: %d >= %d", t1.ID, t2.ID)
	}
	if g.ContarPendientes() != 2 {
		t.Fatalf("esperaba 2 pendientes, hay %d", g.ContarPendientes())
	}

	msg := g.MarcarHecha("enviar el informe")
	if !strings.Contains(msg, "hecha") {
		t.Fatalf("marcar por nombre falló: %q", msg)
	}
	if g.ContarPendientes() != 1 {
		t.Fatalf("esperaba 1 pendiente tras marcar, hay %d", g.ContarPendientes())
	}

	msg = g.MarcarHecha("#2")
	if !strings.Contains(msg, "hecha") {
		t.Fatalf("marcar por id falló: %q", msg)
	}
	if g.ContarPendientes() != 0 {
		t.Fatalf("esperaba 0 pendientes, hay %d", g.ContarPendientes())
	}

	if !g.Borrar("#1") {
		t.Fatal("esperaba borrar la tarea #1")
	}
	if len(g.ListarTodas()) != 1 {
		t.Fatalf("esperaba 1 tarea restante, hay %d", len(g.ListarTodas()))
	}

	g2 := NuevoGestorTareas(filepath.Join(dir, "tareas.json"))
	if len(g2.ListarTodas()) != 1 {
		t.Fatal("las tareas no persistieron entre instancias")
	}
}

func TestManejarTarea(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{tareas: NuevoGestorTareas(filepath.Join(dir, "tareas.json"))}

	got := h.manejarTarea("agendá una tarea enviar el informe para mañana")
	if !strings.Contains(got, "enviar el informe") || !strings.Contains(got, "mañana") {
		t.Fatalf("agendar tarea falló: %q", got)
	}
	if h.tareas.ContarPendientes() != 1 {
		t.Fatalf("esperaba 1 tarea, hay %d", h.tareas.ContarPendientes())
	}

	got = h.manejarTarea("qué tareas tengo")
	if !strings.Contains(got, "enviar el informe") {
		t.Fatalf("listar tareas falló: %q", got)
	}

	got = h.manejarTarea("marcar tarea #1 como hecha")
	if !strings.Contains(got, "hecha") {
		t.Fatalf("marcar tarea falló: %q", got)
	}
	if h.tareas.ContarPendientes() != 0 {
		t.Fatalf("esperaba 0 pendientes, hay %d", h.tareas.ContarPendientes())
	}

	got = h.manejarTarea("todas las tareas")
	if !strings.Contains(got, "hecha") {
		t.Fatalf("listar todas falló: %q", got)
	}

	got = h.manejarTarea("borrar tarea #1")
	if !strings.Contains(got, "eliminada") {
		t.Fatalf("borrar tarea falló: %q", got)
	}
}

func TestGestorProcedimientos(t *testing.T) {
	dir := t.TempDir()
	m := NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))
	m.Crear("hacer el informe", []string{"abrir word", "escribir el resumen"})

	p, ok := m.Obtener("hacer el informe")
	if !ok || len(p.Pasos) != 2 {
		t.Fatalf("esperaba procedimiento con 2 pasos, ok=%v pasos=%v", ok, p)
	}

	m2 := NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))
	if _, ok := m2.Obtener("hacer el informe"); !ok {
		t.Fatal("el procedimiento no persistió entre instancias")
	}

	if !m.Borrar("hacer el informe") {
		t.Fatal("esperaba borrar el procedimiento")
	}
	if _, ok := m.Obtener("hacer el informe"); ok {
		t.Fatal("el procedimiento sigue existiendo tras borrarlo")
	}
}

func TestManejarProcedimiento_AprenderYEjecutar(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))}

	got := h.manejarProcedimiento("aprendé que para hacer saludar al cliente: abrir chrome y eco hola")
	if !strings.Contains(got, "saludar al cliente") || !strings.Contains(got, "2 pasos") {
		t.Fatalf("aprender procedimiento falló: %q", got)
	}
	if p, ok := h.procedimientos.Obtener("saludar al cliente"); !ok || len(p.Pasos) != 2 {
		t.Fatalf("el procedimiento no quedó guardado: %v ok=%v", p, ok)
	}

	got = h.manejarProcedimiento("ejecutá el procedimiento saludar al cliente")
	if !strings.Contains(got, "ejecutado") || !strings.Contains(got, "2 pasos") {
		t.Fatalf("ejecutar procedimiento falló: %q", got)
	}

	got = h.manejarProcedimiento("qué procedimientos sabés")
	if !strings.Contains(got, "saludar al cliente") {
		t.Fatalf("listar procedimientos falló: %q", got)
	}

	got = h.manejarProcedimiento("olvidate el procedimiento saludar al cliente")
	if !strings.Contains(got, "olvidado") {
		t.Fatalf("olvidar procedimiento falló: %q", got)
	}
	if _, ok := h.procedimientos.Obtener("saludar al cliente"); ok {
		t.Fatal("el procedimiento sigue existiendo tras olvidarlo")
	}
}

func TestManejarProcedimiento_PasosPendientes(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))}

	got := h.manejarProcedimiento("aprendé a hacer abrir la cuenta")
	if !strings.Contains(got, "¿Cuáles son los pasos?") {
		t.Fatalf("esperaba pedir los pasos, obtuve: %q", got)
	}
	if h.procedimientoPendiente != "abrir la cuenta" {
		t.Fatalf("pendiente inesperado: %q", h.procedimientoPendiente)
	}

	got = h.manejarProcedimiento("los pasos son: abrir chrome y eco listo")
	if !strings.Contains(got, "Aprendido") || !strings.Contains(got, "2 pasos") {
		t.Fatalf("capturar pasos falló: %q", got)
	}
	if h.procedimientoPendiente != "" {
		t.Fatalf("no debería quedar pendiente: %q", h.procedimientoPendiente)
	}
	if p, ok := h.procedimientos.Obtener("abrir la cuenta"); !ok || len(p.Pasos) != 2 {
		t.Fatalf("procedimiento no guardado correctamente: %v ok=%v", p, ok)
	}
}

func TestManejarProcedimiento_Consulta(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))}

	got := h.manejarProcedimiento("hacé la liquidación de sueldos")
	if !strings.Contains(got, "Enséñeme") {
		t.Fatalf("esperaba pedido de enseñanza, obtuve: %q", got)
	}
}

func TestTextoParaIA_Procedimientos(t *testing.T) {
	dir := t.TempDir()
	m := NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json"))
	m.Crear("hacer el informe", []string{"abrir word", "escribir el resumen"})

	texto := m.TextoParaIA("ayudame a hacer el informe de ventas")
	if !strings.Contains(texto, "hacer el informe") || !strings.Contains(texto, "escribir el resumen") {
		t.Fatalf("el texto para IA debía incluir el procedimiento, obtuve: %q", texto)
	}

	if m.TextoParaIA("cómo está el clima") != "" {
		t.Fatal("no debía inyectar procedimientos para un pedido sin coincidencia")
	}

	m2 := NuevoGestorProcedimientos(filepath.Join(dir, "vacio.json"))
	if m2.TextoParaIA("hacer algo") != "" {
		t.Fatal("sin procedimientos no debe inyectar nada")
	}
}

func TestProcess_ProcedimientosInyectadosEnIA(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	ia := &iaFalsa{disponible: true, respuesta: "Claro, uso el procedimiento aprendido."}
	procs := NuevoGestorProcedimientos(filepath.Join(t.TempDir(), "procedimientos.json"))
	procs.Crear("hacer el informe", []string{"abrir word", "escribir el resumen"})

	b := NewBrain(manos, BrainOpciones{IA: ia, Procedimientos: procs})
	got := b.Process("necesito hacer el informe de ventas")
	if got != "Claro, uso el procedimiento aprendido." {
		t.Fatalf("respuesta = %q, esperaba la de la IA", got)
	}
	if !strings.Contains(ia.consultaRecibida, "hacer el informe") ||
		!strings.Contains(ia.consultaRecibida, "escribir el resumen") {
		t.Fatalf("la IA debía recibir el procedimiento aprendido, obtuvo: %q", ia.consultaRecibida)
	}
}

func TestProcess_PedidoDeTrabajoConsulta(t *testing.T) {
	manos := &manosFalsas{respuesta: ComandoNoReconocido}
	procs := NuevoGestorProcedimientos(filepath.Join(t.TempDir(), "procedimientos.json"))
	b := NewBrain(manos, BrainOpciones{Procedimientos: procs})

	got := b.Process("hacé la liquidación de sueldos")
	if !strings.Contains(got, "Enséñeme") {
		t.Fatalf("esperaba el pedido de enseñanza, obtuve: %q", got)
	}
}
