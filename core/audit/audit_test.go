package audit

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRotacionPorTamano(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "auditoria.jsonl")
	r := NuevoRegistro(ruta)
	r.maxBytes = 120

	for i := 0; i < 10; i++ {
		r.Registrar(Entrada{
			Usuario:   "dueño",
			Rol:       "dueño",
			Orden:     i,
			Comando:   "comando de prueba con contenido largo para superar el límite",
			Resultado: "resultado de prueba con contenido largo",
		})
	}

	if len(r.Listar()) != 10 {
		t.Fatalf("el registro debe conservar todas las entradas en memoria, tengo %d", len(r.Listar()))
	}

	entradas, err := os.ReadDir(filepath.Dir(ruta))
	if err != nil {
		t.Fatalf("no pude leer el directorio: %v", err)
	}
	archivados := 0
	for _, e := range entradas {
		if strings.HasPrefix(e.Name(), "auditoria.jsonl.") {
			archivados++
		}
	}
	if archivados == 0 {
		t.Fatal("no hubo rotación: no existe ningún archivo archivado con sufijo de timestamp")
	}

	reabierto := NuevoRegistro(ruta)
	if len(reabierto.Listar()) == 0 {
		t.Fatal("al reabrir el archivo activo después de rotar debe seguir siendo un JSONL válido con entradas")
	}
}
