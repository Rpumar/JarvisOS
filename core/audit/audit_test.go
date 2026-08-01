package audit

import (
	"path/filepath"
	"testing"
)

func TestRegistrarYReabrir(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "auditoria.jsonl")
	r := NuevoRegistro(ruta)

	r.Registrar(Entrada{Usuario: "dueño", Rol: "dueño", Orden: 1, Comando: "eco hola", Resultado: "hola"})
	r.Registrar(Entrada{Usuario: "dueño", Rol: "dueño", Orden: 1, Comando: "borrar backups", Resultado: "aprobado"})

	if len(r.Listar()) != 2 {
		t.Fatalf("esperaba 2 entradas, tengo %d", len(r.Listar()))
	}

	reabierto := NuevoRegistro(ruta)
	entradas := reabierto.Listar()
	if len(entradas) != 2 {
		t.Fatalf("al reabrir esperaba 2 entradas, tengo %d", len(entradas))
	}
	if entradas[0].Comando != "eco hola" {
		t.Errorf("primera entrada = %+v, esperaba 'eco hola'", entradas[0])
	}
	if entradas[1].Resultado != "aprobado" {
		t.Errorf("segunda entrada resultado = %q, esperaba 'aprobado'", entradas[1].Resultado)
	}
}

func TestRecientes(t *testing.T) {
	r := NuevoRegistro("")
	for i := 0; i < 5; i++ {
		r.Registrar(Entrada{Comando: "cmd"})
	}
	recientes := r.Recientes(2)
	if len(recientes) != 2 {
		t.Fatalf("Recientes(2) devolvió %d, esperaba 2", len(recientes))
	}
}
