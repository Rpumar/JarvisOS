package core

import (
	"path/filepath"
	"strings"
	"testing"

	"JarvisOS/core/audit"
)

func TestAprobarOrden_ReanudaBucle(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		Auditoria:      audit.NuevoRegistro(filepath.Join(dir, "auditoria.jsonl")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"borrar la carpeta de respaldos","razon":"limpiar"}`,
			`{"fin":"respaldo limpio y verificado"}`,
		}},
	}

	o := h.ordenes.Agregar("limpiar respaldos", "dueño")
	msg := h.ejecutarOrdenConIA(o)
	if !strings.Contains(msg, "aprobar") {
		t.Fatalf("esperaba pedido de aprobación, obtuve: %q", msg)
	}

	got := h.AprobarOrden(o.ID, "")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("tras aprobar debía reanudarse y cumplirse, obtuve: %q", got)
	}

	cerrada, _ := h.ordenes.Obtener(o.ID)
	if cerrada.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", cerrada.Estado)
	}
	if !strings.Contains(cerrada.Reporte, "respaldo limpio") {
		t.Fatalf("reporte inesperado: %q", cerrada.Reporte)
	}
	aprobada := false
	for _, a := range cerrada.Historial {
		if strings.Contains(a.Accion, "borrar la carpeta de respaldos") && strings.Contains(a.Resultado, "aprobada") {
			aprobada = true
		}
	}
	if !aprobada {
		t.Fatalf("la acción aprobada debía quedar en el historial: %+v", cerrada.Historial)
	}

	if entradas := h.Auditoria.Listar(); len(entradas) < 2 {
		t.Fatalf("la auditoría debía registrar la solicitud y la ejecución, tengo %d", len(entradas))
	}
}

func TestAprobarOrden_PINIncorrecto(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		PINHash: hashPIN("1234"),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"borrar la carpeta de respaldos"}`,
		}},
	}

	o := h.ordenes.Agregar("limpiar respaldos", "dueño")
	h.ejecutarOrdenConIA(o)

	if got := h.AprobarOrden(o.ID, "9999"); !strings.Contains(got, "PIN incorrecto") {
		t.Fatalf("PIN incorrecto debía rechazarse, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenEsperandoAprobacion {
		t.Fatalf("la orden debía seguir esperando, está: %s", orden.Estado)
	}
}

func TestAprobarOrden_PINCorrectoEjecuta(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		PINHash: hashPIN("1234"),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"borrar la carpeta de respaldos"}`,
			`{"fin":"listo"}`,
		}},
	}

	o := h.ordenes.Agregar("tarea con pin", "dueño")
	h.ejecutarOrdenConIA(o)

	got := h.AprobarOrden(o.ID, "1234")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("con PIN correcto debía reanudarse, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenTerminada {
		t.Fatalf("la orden debía terminar, está: %s", orden.Estado)
	}
}

func TestDenegarOrden_Descarta(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"formatear el disco d"}`,
		}},
	}

	o := h.ordenes.Agregar("formatear", "dueño")
	h.ejecutarOrdenConIA(o)

	got := h.DenegarOrden(o.ID)
	if !strings.Contains(got, "denegada") {
		t.Fatalf("esperaba confirmación de denegación, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenBloqueada {
		t.Fatalf("la orden debía quedar bloqueada, está: %s", orden.Estado)
	}
	if orden.PendienteAccion != "" {
		t.Fatalf("la acción pendiente debía descartarse, quedó: %q", orden.PendienteAccion)
	}
}

func TestAprobarOrden_ViaComandoDeVoz(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
		PINHash:        hashPIN("1234"),
		IA: &iaSecuencial{respuestas: []string{
			`{"accion":"borrar la carpeta de respaldos"}`,
			`{"fin":"completado por voz"}`,
		}},
	}

	o := h.ordenes.Agregar("tarea por voz", "dueño")
	h.ejecutarOrdenConIA(o)

	got := h.manejarOrden("aprobar la orden #1 1234")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("aprobar por voz con PIN debía reanudar, obtuve: %q", got)
	}
	orden, _ := h.ordenes.Obtener(o.ID)
	if orden.Estado != OrdenTerminada {
		t.Fatalf("la orden debía terminar, está: %s", orden.Estado)
	}
}

func TestEstablecerPIN(t *testing.T) {
	h := &Hands{}
	if got := h.EstablecerPIN("123"); !strings.Contains(got, "4 y 6") {
		t.Fatalf("PIN corto debía rechazarse, obtuve: %q", got)
	}
	if got := h.EstablecerPIN("12ab"); !strings.Contains(got, "números") {
		t.Fatalf("PIN con letras debía rechazarse, obtuve: %q", got)
	}
	if got := h.EstablecerPIN("1234"); !strings.Contains(got, "configurado") {
		t.Fatalf("PIN válido debía configurarse, obtuve: %q", got)
	}
	if h.PINHash != hashPIN("1234") {
		t.Fatal("el hash del PIN no se guardó")
	}
}

func TestProcesarOrden_EsperandoAprobacionGuia(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
	}
	o := h.ordenes.Agregar("limpiar", "dueño")
	h.ordenes.SolicitarAprobacion(o.ID, "borrar respaldos", "borrar archivos o datos")

	got := h.procesarOrden(o.ID)
	if !strings.Contains(got, "esperando su aprobación") {
		t.Fatalf("esperaba guía de aprobación, obtuve: %q", got)
	}
}
