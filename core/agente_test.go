package core

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type iaSecuencial struct {
	respuestas []string
	errs       []error
	i          int
}

func (i *iaSecuencial) Disponible() bool { return true }

func (i *iaSecuencial) Consultar(prompt string, historial []TurnoConversacion) (string, error) {
	if i.i < len(i.respuestas) {
		r := i.respuestas[i.i]
		var e error
		if i.i < len(i.errs) {
			e = i.errs[i.i]
		}
		i.i++
		return r, e
	}
	return "", nil
}

func TestParsearRespuestaIA(t *testing.T) {
	casos := []struct {
		resp   string
		accion string
		fin    string
		err    bool
	}{
		{`{"accion":"abrir chrome","razon":"test"}`, "abrir chrome", "", false},
		{`{"fin":"todo listo"}`, "", "todo listo", false},
		{"```json\n{\"fin\":\"resumen\"}\n```", "", "resumen", false},
		{`texto antes { "accion":"eco hola" } texto despues`, "eco hola", "", false},
		{"no es json", "", "", true},
		{`{"accion":"a","fin":"b"}`, "", "", true},
		{`{"otra_cosa":1}`, "", "", true},
		{"{}", "", "", true},
		{`{"fin":""}`, "", "", true},
	}
	for _, c := range casos {
		r, err := parsearRespuestaIA(c.resp)
		if c.err {
			if err == nil {
				t.Errorf("parsearRespuestaIA(%q): esperaba error, obtuve %+v", c.resp, r)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsearRespuestaIA(%q): error inesperado %v", c.resp, err)
			continue
		}
		if r.Accion != c.accion || r.Fin != c.fin {
			t.Errorf("parsearRespuestaIA(%q) = %+v, esperaba accion=%q fin=%q", c.resp, r, c.accion, c.fin)
		}
	}
}

func TestEjecutarOrdenConIA_CumpleYAprende(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"eco hola","razon":"saludar"}`,
			`{"accion":"eco listo","razon":"cerrar"}`,
			`{"fin":"presentación preparada y verificada"}`,
		}},
	}

	o := h.ordenes.Agregar("preparar la presentación", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("esperaba orden cumplida, obtuve: %q", got)
	}

	cerrada, _ := h.ordenes.Obtener(o.ID)
	if cerrada.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", cerrada.Estado)
	}
	if !strings.Contains(cerrada.Reporte, "presentación preparada") {
		t.Fatalf("reporte inesperado: %q", cerrada.Reporte)
	}
	if len(cerrada.Historial) < 4 {
		t.Fatalf("esperaba historial con acciones, tengo: %+v", cerrada.Historial)
	}

	if p, ok := h.procedimientos.Obtener("preparar la presentación"); !ok || len(p.Pasos) != 2 {
		t.Fatalf("la IA debió aprender el procedimiento: %v ok=%v", p, ok)
	}
}

func TestEjecutarOrdenConIA_RechazaAccionSensible(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"borrar la carpeta de respaldos","razon":"limpiar"}`,
		}},
	}

	o := h.ordenes.Agregar("limpiar respaldos", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "aprobación") {
		t.Fatalf("esperaba pedido de aprobación, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenEsperandoAprobacion {
		t.Fatalf("la orden debería estar esperando aprobación, está: %s", orden.Estado)
	}
	if orden.PendienteAccion != "borrar la carpeta de respaldos" {
		t.Fatalf("la acción pendiente no se guardó: %q", orden.PendienteAccion)
	}
}

func TestEjecutarOrdenConIA_ComandoDesconocidoYFin(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"comando inventado","razon":"probar"}`,
			`{"fin":"no pude con ese comando"}`,
		}},
	}

	o := h.ordenes.Agregar("tarea rara", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("esperaba cierre, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", orden.Estado)
	}
	if _, ok := h.procedimientos.Obtener("tarea rara"); ok {
		t.Fatal("no debía aprender un procedimiento sin acciones válidas")
	}
}

func TestEjecutarOrdenConIA_ErrorIA(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		IA: &iaSecuencial{
			respuestas: []string{`{"accion":"eco hola"}`},
			errs:       []error{errors.New("falla de conexión")},
		},
	}

	o := h.ordenes.Agregar("tarea con falla", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "en espera") {
		t.Fatalf("esperaba bloqueo por error de IA, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenBloqueada {
		t.Fatalf("la orden debería estar bloqueada, está: %s", orden.Estado)
	}
}

func TestEjecutarOrdenConIA_JSONInvalidoReintenta(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			"primero voy a pensar bien esto",
			`{"accion":"eco hola","razon":"corregido"}`,
			`{"fin":"reintento exitoso"}`,
		}},
	}

	o := h.ordenes.Agregar("tarea con json roto", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("esperaba cierre tras reintento, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", orden.Estado)
	}
	if p, ok := h.procedimientos.Obtener("tarea con json roto"); !ok || len(p.Pasos) != 1 {
		t.Fatalf("debió aprender el paso del reintento: %v ok=%v", p, ok)
	}
}

func TestEjecutarOrdenConIA_JSONInvalidoBloquea(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		IA: &iaSecuencial{respuestas: []string{
			"texto libre",
			"texto libre otra vez",
			"texto libre de nuevo",
		}},
	}

	o := h.ordenes.Agregar("tarea con json siempre roto", "dueño")
	got := h.ejecutarOrdenConIA(o)
	if !strings.Contains(got, "en espera") {
		t.Fatalf("esperaba bloqueo por JSON inválido repetido, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenBloqueada {
		t.Fatalf("la orden debería estar bloqueada, está: %s", orden.Estado)
	}
}

func TestProcesarOrden_ConIAResuelve(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"eco hola","razon":"iniciar"}`,
			`{"fin":"orden resuelta"}`,
		}},
	}

	got := h.manejarOrden("agendá una orden resolver el misterio")
	if !strings.Contains(got, "Orden #1") {
		t.Fatalf("agendar falló: %q", got)
	}
	got = h.manejarOrden("ejecutá la orden #1")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("ejecutar orden con IA falló: %q", got)
	}
	if p, ok := h.procedimientos.Obtener("resolver el misterio"); !ok || len(p.Pasos) != 1 {
		t.Fatalf("la IA debió aprender el procedimiento: %v ok=%v", p, ok)
	}
}

func TestProcesarOrden_ConIAVerificaProcedimientoConocido(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"eco paso extra","razon":"completar"}`,
			`{"fin":"verificado, quedó completa"}`,
		}},
	}
	h.procedimientos.Crear("armar el informe", []string{"eco base"})

	o := h.ordenes.Agregar("armar el informe", "dueño")
	got := h.procesarOrden(o.ID)
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("esperaba cierre verificado, obtuve: %q", got)
	}
	cerrada, _ := h.ordenes.Obtener(o.ID)
	if cerrada.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", cerrada.Estado)
	}
	if !strings.Contains(cerrada.Reporte, "verificado") {
		t.Fatalf("el reporte debía venir del fin de la IA: %q", cerrada.Reporte)
	}
	if p, ok := h.procedimientos.Obtener("armar el informe"); !ok || len(p.Pasos) != 2 {
		t.Fatalf("el procedimiento debía sumar el paso extra: %v ok=%v", p, ok)
	}
}
