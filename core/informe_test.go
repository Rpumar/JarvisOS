package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"JarvisOS/core/audit"
)

func TestGenerarInformeDiario_Contenidos(t *testing.T) {
	d := InformeDiarioDatos{
		TareasPendientes: []Tarea{{Nombre: "enviar resumen"}},
		OrdenesActivas:   []Orden{{ID: 2, Objetivo: "auditar servidores", Estado: OrdenEnProgreso}},
		OrdenesTerminadas: []Orden{{ID: 1, Objetivo: "preparar informe", Estado: OrdenTerminada}},
		ActividadHoy: []audit.Entrada{
			{Momento: "10:00:00", Comando: "abrir word", Resultado: "OK"},
		},
		EventosManana: []EventoAgenda{{Titulo: "Reunión", Inicio: "09:00"}},
	}

	informe := GenerarInformeDiario(d)

	casos := []string{"Informe diario", "preparar informe", "auditar servidores", "enviar resumen", "abrir word", "Reunión"}
	for _, caso := range casos {
		if !strings.Contains(informe, caso) {
			t.Errorf("el informe debería mencionar %q, obtuve: %q", caso, informe)
		}
	}
}

func TestGenerarInformeDiario_Vacio(t *testing.T) {
	informe := GenerarInformeDiario(InformeDiarioDatos{})
	if strings.TrimSpace(informe) == "" {
		t.Fatal("el informe aún debe tener el encabezado")
	}
	if !strings.Contains(informe, "Informe diario") {
		t.Fatalf("falta encabezado: %q", informe)
	}
}

func TestGuardarInformeDiario(t *testing.T) {
	dir := t.TempDir()
	ruta, err := GuardarInformeDiario(dir, "2026-08-05", "Informe diario, señor.")
	if err != nil {
		t.Fatal(err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil || string(datos) != "Informe diario, señor." {
		t.Fatalf("contenido inesperado: %q err=%v", datos, err)
	}
	if filepath.Base(ruta) != "2026-08-05.txt" {
		t.Fatalf("nombre de archivo inesperado: %q", filepath.Base(ruta))
	}
}