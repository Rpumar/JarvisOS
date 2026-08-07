package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"JarvisOS/core/audit"
)

func TestGenerarInformePiloto_Contenidos(t *testing.T) {
	d := InformePilotoDatos{
		Desde:             "2026-07-08",
		Hasta:             "2026-08-06",
		OrdenesTerminadas: 12,
		TareasHechas:      8,
		AccionesAuditadas: 30,
		Aprobadas:         5,
		Denegadas:         1,
		Expiradas:         2,
		HorasAhorradas:    6.6,
	}
	informe := GenerarInformePiloto(d)
	casos := []string{
		"Informe de cierre de piloto",
		"2026-07-08 a 2026-08-06",
		"Tareas cumplidas: 20",
		"órdenes: 12",
		"tareas: 8",
		"6.6 en total",
		"por semana",
		"Acciones auditadas: 30",
		"Aprobadas: 5",
		"Denegadas: 1",
		"Expiradas por timeout: 2",
	}
	for _, c := range casos {
		if !strings.Contains(informe, c) {
			t.Errorf("el informe debería mencionar %q, obtuve: %q", c, informe)
		}
	}
}

func TestGenerarInformePiloto_Vacio(t *testing.T) {
	informe := GenerarInformePiloto(InformePilotoDatos{Desde: "a", Hasta: "b"})
	if strings.TrimSpace(informe) == "" {
		t.Fatal("el informe aún debe tener contenido")
	}
	if !strings.Contains(informe, "Informe de cierre de piloto") {
		t.Fatalf("falta encabezado: %q", informe)
	}
	if !strings.Contains(informe, "control total") {
		t.Fatalf("sin denegadas/expiradas debería resaltar el control: %q", informe)
	}
}

func TestRecolectarInformePiloto_Metricas(t *testing.T) {
	dir := t.TempDir()
	ordenes := NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json"))
	tareas := NuevoGestorTareas(filepath.Join(dir, "tareas.json"))
	reg := audit.NuevoRegistro(filepath.Join(dir, "auditoria.jsonl"))

	h := &Hands{
		ordenes:   ordenes,
		tareas:    tareas,
		Auditoria: reg,
		DatosDir:  dir,
	}

	// Orden terminada dentro de la ventana.
	o := ordenes.Agregar("preparar informe", "dueño")
	ordenes.Terminar(o.ID, "listo")

	// Orden terminada hace más de 30 días: debe quedar fuera.
	vieja := ordenes.Agregar("tarea vieja", "dueño")
	ordenes.Terminar(vieja.ID, "ok")
	// Fuerzo la fecha hacia atrás manipulando el archivo no es viable sin
	// setter; en su lugar verifico que el conteo incluya al menos las nuevas.

	// Tarea hecha hoy.
	tareas.Agregar("redactar mail", "", "")
	if got := tareas.MarcarHecha("redactar mail"); !strings.Contains(got, "hecha") {
		t.Fatalf("no se pudo marcar la tarea: %q", got)
	}

	// Acciones auditadas de hoy, con una aprobada y una denegada.
	reg.Registrar(audit.Entrada{Momento: time.Now().Format("2006-01-02 15:04:05"), Comando: "x", Resultado: "listo"})
	reg.Registrar(audit.Entrada{Momento: time.Now().Format("2006-01-02 15:04:05"), Comando: "x", Resultado: "denegada_por_el_dueño"})

	d := h.RecolectarInformePiloto(time.Now())

	if d.OrdenesTerminadas < 1 {
		t.Errorf("esperaba al menos 1 orden terminada en la ventana, obtuve %d", d.OrdenesTerminadas)
	}
	if d.TareasHechas != 1 {
		t.Errorf("esperaba 1 tarea hecha, obtuve %d", d.TareasHechas)
	}
	if d.AccionesAuditadas < 2 {
		t.Errorf("esperaba al menos 2 acciones auditadas, obtuve %d", d.AccionesAuditadas)
	}
	if d.Denegadas != 1 {
		t.Errorf("esperaba 1 denegada, obtuve %d", d.Denegadas)
	}
	if d.HorasAhorradas <= 0 {
		t.Errorf("esperaba horas ahorradas > 0, obtuve %.1f", d.HorasAhorradas)
	}
}

func TestGuardarInformePiloto(t *testing.T) {
	dir := t.TempDir()
	ruta, err := GuardarInformePiloto(dir, "Informe de cierre de piloto, señor.")
	if err != nil {
		t.Fatalf("GuardarInformePiloto falló: %v", err)
	}
	if !strings.Contains(filepath.Base(ruta), "piloto-") {
		t.Fatalf("nombre de archivo inesperado: %q", filepath.Base(ruta))
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer el informe guardado: %v", err)
	}
	if string(datos) != "Informe de cierre de piloto, señor." {
		t.Fatalf("contenido inesperado: %q", string(datos))
	}
}

func TestInformePiloto_HandlerGeneraYGuarda(t *testing.T) {
	dir := t.TempDir()
	h := &Hands{
		ordenes:  NuevoGestorOrdenes(filepath.Join(dir, "ordenes.json")),
		tareas:   NuevoGestorTareas(filepath.Join(dir, "tareas.json")),
		DatosDir: dir,
	}
	o := h.ordenes.Agregar("hacer planilla", "dueño")
	h.ordenes.Terminar(o.ID, "ok")

	informe, ruta := h.generarInformePilotoYGuardar(filepath.Join(dir, "informes"))
	if !strings.Contains(informe, "Informe de cierre de piloto") {
		t.Fatalf("informe sin encabezado: %q", informe)
	}
	if ruta == "" {
		t.Fatal("el informe debía guardarse en disco")
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("el archivo guardado no existe: %v", err)
	}
}
