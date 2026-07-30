package core

import "testing"

func contiene(opciones []string, valor string) bool {
	for _, o := range opciones {
		if o == valor {
			return true
		}
	}
	return false
}

func TestFraseAlAzar_ListaVacia(t *testing.T) {
	if got := fraseAlAzar(nil); got != "" {
		t.Errorf("fraseAlAzar(nil) = %q, esperaba cadena vacía", got)
	}
}

func TestFraseAlAzar_SiempreDevuelveUnaOpcionValida(t *testing.T) {
	opciones := []string{"a", "b", "c"}
	// Se repite varias veces porque es aleatorio: una sola llamada no
	// alcanza para confiar en que nunca devuelve algo fuera de la lista.
	for i := 0; i < 50; i++ {
		got := fraseAlAzar(opciones)
		if !contiene(opciones, got) {
			t.Fatalf("fraseAlAzar devolvió %q, que no está en las opciones", got)
		}
	}
}

func TestSaludo_SiempreDevuelveAlgoDelBanco(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := Saludo(); !contiene(saludos, got) {
			t.Fatalf("Saludo() devolvió %q, fuera del banco esperado", got)
		}
	}
}

func TestDespedida_SiempreDevuelveAlgoDelBanco(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := Despedida(); !contiene(despedidas, got) {
			t.Fatalf("Despedida() devolvió %q, fuera del banco esperado", got)
		}
	}
}

func TestRespuestaConfusion_SiempreDevuelveAlgoDelBanco(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := RespuestaConfusion(); !contiene(respuestasConfusion, got) {
			t.Fatalf("RespuestaConfusion() devolvió %q, fuera del banco esperado", got)
		}
	}
}

func TestConfirmacionGenerica_SiempreDevuelveAlgoDelBanco(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := ConfirmacionGenerica(); !contiene(confirmacionesGenericas, got) {
			t.Fatalf("ConfirmacionGenerica() devolvió %q, fuera del banco esperado", got)
		}
	}
}
