package agents

import (
	"testing"
)

type chatIAFalsa struct {
	disponible bool
}

func (f *chatIAFalsa) Disponible() bool { return f.disponible }
func (f *chatIAFalsa) Chat(system, user string) (string, error) {
	return "objetivo", nil
}

func nuevoAgenteFalso(disponible bool, dir string) *AgenteProyecto {
	return NuevoAgenteProyecto(&chatIAFalsa{disponible: disponible}, dir, NuevoGestorPlan(dir+"/planes"))
}

func TestAgenteProyectoDisponible(t *testing.T) {
	dir := t.TempDir()
	if NuevoAgenteProyecto(nil, dir, NuevoGestorPlan(dir)).Disponible() {
		t.Error("sin IA no debería estar disponible")
	}
	if !nuevoAgenteFalso(true, dir).Disponible() {
		t.Error("con IA disponible debería estar disponible")
	}
	if nuevoAgenteFalso(false, dir).Disponible() {
		t.Error("IA no disponible → agente no disponible")
	}
}

func TestProcesarSinIADisponible(t *testing.T) {
	dir := t.TempDir()
	a := nuevoAgenteFalso(false, dir)
	respuesta := a.Procesar("creá una función")
	if respuesta == "" {
		t.Error("sin IA debería responder un mensaje claro")
	}
	if a.TieneTareaPendiente() {
		t.Error("sin IA no debería quedar tarea pendiente")
	}
}

func TestContinuarYCancelarSinPlan(t *testing.T) {
	dir := t.TempDir()
	a := nuevoAgenteFalso(true, dir)

	if r := a.ContinuarPlan(); r == "" {
		t.Error("ContinuarPlan sin plan debería responder algo")
	}
	if r := a.CancelarPlan(); r == "" {
		t.Error("CancelarPlan sin plan debería responder algo")
	}
	if d := a.PlanPendienteDescripcion(); d != "" {
		t.Errorf("sin plan la descripción debería ser vacía, fue %q", d)
	}
	if a.TieneTareaPendiente() {
		t.Error("sin plan no debería haber tarea pendiente")
	}
	a.Reset() // no debe panickear
	a.SetRespuestaUsuario("hola")
}
