package memoria

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func nuevoAlmacenTest(t *testing.T) (*Almacen, string) {
	ruta := filepath.Join(t.TempDir(), "memoria.db")
	a, err := NuevoAlmacen(ruta)
	if err != nil {
		t.Fatalf("no se pudo crear almacén de prueba: %v", err)
	}
	return a, ruta
}

func TestNuevoAlmacen_ArchivoInexistente_ArrancaVacio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "memoria.db")
	a, err := NuevoAlmacen(ruta)
	if err != nil {
		t.Fatalf("no se esperaba error con archivo inexistente: %v", err)
	}
	defer a.Cerrar()
	if _, existe := a.ObtenerHecho("nombre"); existe {
		t.Error("no debería haber ningún hecho en un almacén recién creado")
	}
	if len(a.ObtenerNotas()) != 0 {
		t.Error("no debería haber ninguna nota en un almacén recién creado")
	}
}

func TestGuardarYObtenerHecho(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	if err := a.GuardarHecho("nombre", "Juan"); err != nil {
		t.Fatalf("no se esperaba error al guardar: %v", err)
	}

	valor, existe := a.ObtenerHecho("nombre")
	if !existe {
		t.Fatal("se esperaba que el hecho existiera")
	}
	if valor != "Juan" {
		t.Errorf("valor = %q, esperaba %q", valor, "Juan")
	}
}

func TestObtenerHecho_NoExistente(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	if _, existe := a.ObtenerHecho("ciudad"); existe {
		t.Error("no debería existir un hecho que nunca se guardó")
	}
}

func TestAgregarYObtenerNotas(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	if err := a.AgregarNota("tengo reunión el jueves"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if err := a.AgregarNota("comprar pan"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	notas := a.ObtenerNotas()
	if len(notas) != 2 {
		t.Fatalf("len(notas) = %d, esperaba 2", len(notas))
	}
	if !strings.Contains(notas[0], "reunión") {
		t.Errorf("notas[0] = %q, esperaba que contuviera 'reunión'", notas[0])
	}
	if !strings.Contains(notas[1], "pan") {
		t.Errorf("notas[1] = %q, esperaba que contuviera 'pan'", notas[1])
	}
}

func TestBuscarNotas(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	if err := a.AgregarNota("tengo reunión el jueves"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if err := a.AgregarNota("comprar pan"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	notas := a.BuscarNotas("pan")
	if len(notas) != 1 {
		t.Fatalf("len(notas) = %d, esperaba 1", len(notas))
	}
	if !strings.Contains(notas[0], "pan") {
		t.Errorf("notas[0] = %q, esperaba que contuviera 'pan'", notas[0])
	}

	notas = a.BuscarNotas("reunión")
	if len(notas) != 1 {
		t.Fatalf("len(notas) = %d, esperaba 1", len(notas))
	}
	if !strings.Contains(notas[0], "reunión") {
		t.Errorf("notas[0] = %q, esperaba que contuviera 'reunión'", notas[0])
	}

	if notas = a.BuscarNotas("zzz"); len(notas) != 0 {
		t.Errorf("len(notas) = %d, esperaba 0", len(notas))
	}

	notas = a.BuscarNotas("PAN")
	if len(notas) != 1 {
		t.Errorf("len(notas) = %d, esperaba 1 (búsqueda case-insensitive)", len(notas))
	}
}

func TestPersistenciaRealEntreInstancias(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "memoria.db")

	primero, err := NuevoAlmacen(ruta)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if err := primero.GuardarHecho("nombre", "Ana"); err != nil {
		t.Fatalf("no se esperaba error al guardar: %v", err)
	}
	if err := primero.AgregarNota("primera nota"); err != nil {
		t.Fatalf("no se esperaba error al guardar: %v", err)
	}
	primero.Cerrar()

	segundo, err := NuevoAlmacen(ruta)
	if err != nil {
		t.Fatalf("no se esperaba error al recargar: %v", err)
	}
	defer segundo.Cerrar()

	nombre, existe := segundo.ObtenerHecho("nombre")
	if !existe || nombre != "Ana" {
		t.Errorf("el hecho no sobrevivió al recargar: existe=%v, nombre=%q", existe, nombre)
	}
	if len(segundo.ObtenerNotas()) != 1 {
		t.Errorf("las notas no sobrevivieron al recargar: %v", segundo.ObtenerNotas())
	}
}

func TestArchivoCorrupto_DevuelveError(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "memoria.db")
	if err := os.WriteFile(ruta, []byte("esto no es un archivo SQLite válido"), 0o600); err != nil {
		t.Fatalf("no se pudo preparar el archivo de prueba: %v", err)
	}

	if _, err := NuevoAlmacen(ruta); err == nil {
		t.Error("se esperaba un error al cargar un archivo corrupto")
	}
}

func TestAgregarRecordatorio_YObtenerlo(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()
	momento := time.Now().Add(1 * time.Hour)

	if err := a.AgregarRecordatorio("llamar a mamá", momento); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	pendientes := a.RecordatoriosPendientes(time.Now())
	if len(pendientes) != 0 {
		t.Errorf("no debería haber pendientes antes de la hora programada, hay %d", len(pendientes))
	}

	pendientes = a.RecordatoriosPendientes(momento.Add(1 * time.Minute))
	if len(pendientes) != 1 {
		t.Fatalf("esperaba 1 pendiente después de la hora programada, hay %d", len(pendientes))
	}
	if pendientes[0].Texto != "llamar a mamá" {
		t.Errorf("texto = %q, esperaba %q", pendientes[0].Texto, "llamar a mamá")
	}
}

func TestMarcarCumplido_YaNoAparecePendiente(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()
	momento := time.Now().Add(-1 * time.Hour)

	_ = a.AgregarRecordatorio("comprar pan", momento)
	pendientes := a.RecordatoriosPendientes(time.Now())
	if len(pendientes) != 1 {
		t.Fatalf("esperaba 1 pendiente, hay %d", len(pendientes))
	}

	if err := a.MarcarCumplido(pendientes[0].ID); err != nil {
		t.Fatalf("no se esperaba error al marcar cumplido: %v", err)
	}

	pendientes = a.RecordatoriosPendientes(time.Now())
	if len(pendientes) != 0 {
		t.Errorf("no debería quedar pendiente después de marcarlo cumplido, hay %d", len(pendientes))
	}
}

func TestRecordatorioRecurrente_MarcarCumplidoGeneraSiguiente(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	ahora := time.Now()
	hoy08 := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 8, 0, 0, 0, ahora.Location())
	hoy0805 := hoy08.Add(5 * time.Minute)
	manana0805 := hoy08.AddDate(0, 0, 1).Add(5 * time.Minute)

	if err := a.AgregarRecordatorioConPeriodo("pastilla", hoy08, "diario"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	pendientes := a.RecordatoriosPendientes(hoy0805)
	if len(pendientes) != 1 {
		t.Fatalf("a las 08:05 esperaba 1 pendiente, hay %d", len(pendientes))
	}
	id := pendientes[0].ID

	if err := a.MarcarCumplido(id); err != nil {
		t.Fatalf("no se esperaba error al marcar cumplido: %v", err)
	}

	var total int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM recordatorios WHERE id = ?", id).Scan(&total); err != nil {
		t.Fatalf("no se pudo contar las filas: %v", err)
	}
	if total != 1 {
		t.Errorf("COUNT(*) = %d, esperaba 1 (misma fila reprogramada, sin duplicados)", total)
	}

	pendientes = a.RecordatoriosPendientes(hoy0805)
	if len(pendientes) != 0 {
		t.Errorf("tras marcarlo cumplido a las 08:05 no debería quedar pendiente, hay %d", len(pendientes))
	}

	pendientes = a.RecordatoriosPendientes(manana0805)
	if len(pendientes) != 1 {
		t.Fatalf("esperaba 1 pendiente mañana a las 08:05, hay %d", len(pendientes))
	}
	if pendientes[0].Cumplido {
		t.Error("el recordatorio reprogramado no debería aparecer como cumplido")
	}
}

func TestRecordatorios_SobrevivenAlRecargar(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "memoria.db")
	momento := time.Now().Add(2 * time.Hour)

	primero, _ := NuevoAlmacen(ruta)
	if err := primero.AgregarRecordatorio("reunión importante", momento); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	primero.Cerrar()

	segundo, err := NuevoAlmacen(ruta)
	if err != nil {
		t.Fatalf("no se esperaba error al recargar: %v", err)
	}
	defer segundo.Cerrar()
	pendientes := segundo.RecordatoriosPendientes(momento.Add(1 * time.Minute))
	if len(pendientes) != 1 || pendientes[0].Texto != "reunión importante" {
		t.Errorf("el recordatorio no sobrevivió al recargar: %+v", pendientes)
	}
}

func TestAlmacen_AccesoConcurrente_NoRompe(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = a.AgregarNota(fmt.Sprintf("nota %d", n))
		}(i)
		go func(n int) {
			defer wg.Done()
			_ = a.AgregarRecordatorio(fmt.Sprintf("recordatorio %d", n), time.Now())
		}(i)
	}
	wg.Wait()

	if len(a.ObtenerNotas()) != 20 {
		t.Errorf("se esperaban 20 notas tras el acceso concurrente, hay %d", len(a.ObtenerNotas()))
	}
}

func TestObtenerRecordatoriosPendientesTexto_IncluyeFuturosYNoSoloVencidos(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.AgregarRecordatorio("cosa futura", time.Now().Add(2*time.Hour))
	_ = a.AgregarRecordatorio("cosa vencida", time.Now().Add(-1*time.Hour))

	textos := a.ObtenerRecordatoriosPendientesTexto()

	if len(textos) != 2 {
		t.Fatalf("esperaba 2 recordatorios pendientes (incluyendo el futuro), hay %d: %v", len(textos), textos)
	}
}

func TestObtenerRecordatoriosPendientesTexto_NoIncluyeCumplidos(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.AgregarRecordatorio("ya avisado", time.Now().Add(-1*time.Hour))
	pendientes := a.RecordatoriosPendientes(time.Now())
	_ = a.MarcarCumplido(pendientes[0].ID)

	textos := a.ObtenerRecordatoriosPendientesTexto()
	if len(textos) != 0 {
		t.Errorf("un recordatorio cumplido no debería aparecer como pendiente, hay %d", len(textos))
	}
}

func TestCancelarRecordatorios_PorTexto(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.AgregarRecordatorio("llamar a mamá", time.Now().Add(1*time.Hour))
	_ = a.AgregarRecordatorio("comprar pan", time.Now().Add(2*time.Hour))

	n, err := a.CancelarRecordatorios("mamá")

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba cancelar 1 recordatorio, canceló %d", n)
	}
	restantes := a.ObtenerRecordatoriosPendientesTexto()
	if len(restantes) != 1 {
		t.Errorf("esperaba 1 recordatorio restante, hay %d: %v", len(restantes), restantes)
	}
}

func TestCancelarRecordatorios_TodosConTextoVacio(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.AgregarRecordatorio("uno", time.Now().Add(1*time.Hour))
	_ = a.AgregarRecordatorio("dos", time.Now().Add(2*time.Hour))

	n, err := a.CancelarRecordatorios("")

	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if n != 2 {
		t.Errorf("esperaba cancelar 2 recordatorios, canceló %d", n)
	}
	if len(a.ObtenerRecordatoriosPendientesTexto()) != 0 {
		t.Error("no debería quedar ningún recordatorio pendiente")
	}
}

func TestListas_CrearYAgregarItems(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	if err := a.CrearLista("compras"); err != nil {
		t.Fatalf("no se pudo crear la lista: %v", err)
	}
	if err := a.CrearLista("compras"); err == nil {
		t.Error("crear lista duplicada debería fallar")
	}
	if err := a.AgregarItemLista("compras", "pan"); err != nil {
		t.Fatalf("no se pudo agregar item: %v", err)
	}
	if err := a.AgregarItemLista("compras", "leche"); err != nil {
		t.Fatalf("no se pudo agregar item: %v", err)
	}
	if err := a.AgregarItemLista("inexistente", "x"); err == nil {
		t.Error("agregar a lista inexistente debería fallar")
	}

	lista, ok := a.ObtenerLista("compras")
	if !ok {
		t.Fatal("no se encontró la lista compras")
	}
	if !strings.Contains(lista, "pan") || !strings.Contains(lista, "leche") {
		t.Errorf("la lista debería contener los items: %q", lista)
	}
}

func TestListas_MarcarItem(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.CrearLista("tareas")
	_ = a.AgregarItemLista("tareas", "pagar impuestos")
	_ = a.AgregarItemLista("tareas", "revisar email")

	marcado, err := a.MarcarItemLista("tareas", "impuestos")
	if err != nil {
		t.Fatalf("no se pudo marcar: %v", err)
	}
	if !strings.Contains(marcado, "impuestos") {
		t.Errorf("marcado = %q", marcado)
	}
	lista, _ := a.ObtenerLista("tareas")
	if !strings.Contains(lista, "☑") {
		t.Errorf("esperaba item marcado con ☑: %q", lista)
	}
	if _, err := a.MarcarItemLista("tareas", "no-existe"); err == nil {
		t.Error("marcar item inexistente debería fallar")
	}
}

func TestListas_EliminarYListar(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.CrearLista("compras")
	_ = a.CrearLista("ideas")
	listas := a.ObtenerListas()
	if len(listas) != 2 {
		t.Errorf("esperaba 2 listas, hay %d", len(listas))
	}
	if err := a.EliminarLista("compras"); err != nil {
		t.Fatalf("no se pudo eliminar: %v", err)
	}
	if err := a.EliminarLista("compras"); err == nil {
		t.Error("eliminar lista inexistente debería fallar")
	}
	if _, ok := a.ObtenerLista("compras"); ok {
		t.Error("la lista eliminada no debería existir")
	}
	if len(a.ObtenerListas()) != 1 {
		t.Errorf("esperaba 1 lista tras eliminar, hay %d", len(a.ObtenerListas()))
	}
}

func TestListaVacia(t *testing.T) {
	a, _ := nuevoAlmacenTest(t)
	defer a.Cerrar()

	_ = a.CrearLista("vacia")
	lista, ok := a.ObtenerLista("vacia")
	if !ok || !strings.Contains(lista, "vacía") {
		t.Errorf("lista vacía debería decirlo: %q ok=%v", lista, ok)
	}
}
