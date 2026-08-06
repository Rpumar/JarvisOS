package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGestorOrdenes(t *testing.T) {
	dir := t.TempDir()
	g := NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json"))

	o1 := g.Agregar("preparar la presentación mensual", "dueño")
	o2 := g.Agregar("revisar la contabilidad", "dueño")
	if o1.ID >= o2.ID {
		t.Fatalf("los IDs deben incrementarse: %d >= %d", o1.ID, o2.ID)
	}
	if len(g.Activas()) != 2 {
		t.Fatalf("esperaba 2 órdenes activas, hay %d", len(g.Activas()))
	}
	if len(g.Continuables()) != 2 {
		t.Fatalf("esperaba 2 continuables, hay %d", len(g.Continuables()))
	}

	if !g.CambiarEstado(o1.ID, OrdenEnProgreso) {
		t.Fatal("cambiar estado falló")
	}
	if !g.RegistrarAccion(o1.ID, "abrir word", "hecho") {
		t.Fatal("registrar acción falló")
	}
	if !g.Terminar(o1.ID, "Orden cumplida.") {
		t.Fatal("terminar falló")
	}

	got, ok := g.Obtener(o1.ID)
	if !ok || got.Estado != OrdenTerminada || got.Reporte != "Orden cumplida." {
		t.Fatalf("orden no terminada correctamente: %+v ok=%v", got, ok)
	}
	if len(got.Historial) != 1 || got.Historial[0].Accion != "abrir word" {
		t.Fatalf("historial no guardado: %+v", got.Historial)
	}
	if len(g.Activas()) != 1 {
		t.Fatalf("esperaba 1 activa tras terminar, hay %d", len(g.Activas()))
	}

	g2 := NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json"))
	if _, ok := g2.Obtener(o2.ID); !ok {
		t.Fatal("las órdenes no persistieron entre instancias")
	}
	if g2.ObtenerMaxID() < o2.ID {
		t.Fatalf("el nextID no se recuperó correctamente, max=%d", g2.ObtenerMaxID())
	}
}

func TestManejarOrden_AgendarYListar(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{ordenes: NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json"))}

	got := h.manejarOrden("agendá una orden preparar la presentación mensual")
	if !strings.Contains(got, "Orden #1") || !strings.Contains(got, "presentación mensual") {
		t.Fatalf("agendar orden falló: %q", got)
	}

	got = h.manejarOrden("qué órdenes tengo")
	if !strings.Contains(got, "presentación mensual") {
		t.Fatalf("listar órdenes falló: %q", got)
	}

	got = h.manejarOrden("todas las órdenes")
	if !strings.Contains(got, "pendiente") {
		t.Fatalf("listar todas falló: %q", got)
	}
}

func TestManejarOrden_EjecutarConProcedimiento(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:       NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
	}
	h.procedimientos.Crear("revisar la contabilidad", []string{"eco hola"})

	got := h.manejarOrden("agendá una orden revisar la contabilidad")
	if !strings.Contains(got, "Orden #1") {
		t.Fatalf("agendar falló: %q", got)
	}

	got = h.manejarOrden("ejecutá la orden #1")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("ejecutar orden falló: %q", got)
	}

	o, _ := h.ordenes.Obtener(1)
	if o.Estado != OrdenTerminada {
		t.Fatalf("la orden debería estar terminada, está: %s", o.Estado)
	}
	if len(o.Historial) < 2 {
		t.Fatalf("esperaba historial con acciones, tengo: %+v", o.Historial)
	}

	got = h.manejarOrden("reportá la orden #1")
	if !strings.Contains(got, "cumplida") {
		t.Fatalf("reportar orden falló: %q", got)
	}
}

func TestManejarOrden_EjecutarSinProcedimientoBloquea(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:       NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
	}

	h.manejarOrden("agendá una orden auditar los servidores")
	got := h.manejarOrden("ejecutá la orden #1")
	if !strings.Contains(got, "Enséñeme") {
		t.Fatalf("esperaba pedido de enseñanza, obtuve: %q", got)
	}
	o, _ := h.ordenes.Obtener(1)
	if o.Estado != OrdenBloqueada {
		t.Fatalf("la orden debería estar bloqueada, está: %s", o.Estado)
	}
}

func TestManejarOrden_RetomarYControl(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:       NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
	}
	h.procedimientos.Crear("enviar el resumen semanal", []string{"eco listo"})

	h.manejarOrden("agendá una orden enviar el resumen semanal")
	h.manejarOrden("agendá una orden auditar los servidores")

	got := h.manejarOrden("retomá las órdenes")
	if !strings.Contains(got, "1 cumplidas") {
		t.Fatalf("retomar órdenes falló: %q", got)
	}

	got = h.manejarOrden("cancelar la orden #2")
	if !strings.Contains(got, "cancelada") {
		t.Fatalf("cancelar orden falló: %q", got)
	}
	if len(h.ordenes.Activas()) != 0 {
		t.Fatalf("esperaba 0 activas (una terminada, una cancelada), hay %d", len(h.ordenes.Activas()))
	}

	got = h.manejarOrden("marcar la orden #2 como terminada")
	if !strings.Contains(got, "No encontré la orden #2") && !strings.Contains(got, "terminada") {
		t.Fatalf("control de orden falló: %q", got)
	}
}

func TestExtraerID(t *testing.T) {
	casos := map[string]int{
		"ejecutá la orden #3": 3,
		"reporte de la orden 42": 42,
		"hola":               0,
		"bloquear #7":        7,
	}
	for entrada, esperado := range casos {
		if got := extraerID(entrada); got != esperado {
			t.Errorf("extraerID(%q) = %d, esperaba %d", entrada, got, esperado)
		}
	}
}

func TestEscribirJSONAtomico(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "datos.json")
	if err := escribirJSONAtomico(ruta, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("escribirJSONAtomico falló: %v", err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil || string(datos) != `{"ok":true}` {
		t.Fatalf("contenido inesperado: %s err=%v", datos, err)
	}
}

// TestRetomarOrdenes_VerificaYRespetaAprobacion verifica que la pasada
// automática del bucle (RetomarOrdenes) termina su trabajo con
// procedimiento conocido y que una orden en espera de aprobación quede
// intacta (no se cierra sola).
func TestRetomarOrdenes_VerificaYRespetaAprobacion(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
	}
	h.procedimientos.Crear("preparar informe", []string{"eco hola"})

	o1 := h.ordenes.Agregar("preparar informe", "dueño")
	o2 := h.ordenes.Agregar("enviar correo sensible", "dueño")
	h.ordenes.SolicitarAprobacion(o2.ID, "enviar email", "enviar correo al cliente")

	resumen := h.RetomarOrdenes()

	if !strings.Contains(resumen, "cumplidas") {
		t.Fatalf("el resumen debería mencionar las cumplidas, obtuve: %q", resumen)
	}
	terminada, _ := h.ordenes.Obtener(o1.ID)
	if terminada.Estado != OrdenTerminada {
		t.Fatalf("la orden cumplible debería terminar, está: %s", terminada.Estado)
	}
	esperando, _ := h.ordenes.Obtener(o2.ID)
	if esperando.Estado != OrdenEsperandoAprobacion {
		t.Fatalf("la orden debe seguir esperando aprobación, está: %s", esperando.Estado)
	}
}

// TestRetomarOrdenes_SinIA_NoCierraFalso: sin procedimiento conocido y
// sin IA, la orden queda bloqueada para que el dueño la revise; el bucle
// nunca la marca como cumplida de forma falsa.
func TestRetomarOrdenes_SinIA_NoCierraFalso(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:        NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		procedimientos: NuevoGestorProcedimientos(filepath.Join(dir, "procedimientos.json")),
	}
	h.ordenes.Agregar("auditar infraestructura", "dueño")

	_ = h.RetomarOrdenes()

	o, _ := h.ordenes.Obtener(1)
	if o.Estado != OrdenBloqueada {
		t.Fatalf("sin procedimiento ni IA la orden debería bloquearse, está: %s", o.Estado)
	}
	if o.Estado == OrdenTerminada {
		t.Fatal("el bucle no debe marcar una orden como cumplida sin verificación")
	}
}
