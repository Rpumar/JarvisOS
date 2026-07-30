package core

import (
	"testing"
	"time"
)

func TestExtraerRecordatorio_CasosValidos(t *testing.T) {
	casos := []struct {
		entrada       string
		textoEsperado string
		horaEsperada  int
		minEsperado   int
	}{
		{"recordame llamar a mamá a las 17", "llamar a mamá", 17, 0},
		{"recordame llamar a mamá a las 17:30", "llamar a mamá", 17, 30},
		{"recordame llamar a mamá a las 5 de la tarde", "llamar a mamá", 17, 0},
		{"recordame tomar la pastilla a las 8 de la mañana", "tomar la pastilla", 8, 0},
		{"avisame comprar pan a las 9 de la noche", "comprar pan", 21, 0},
		{"recordame que llame al banco a las 10", "llame al banco", 10, 0},
	}

	for _, c := range casos {
		texto, momento, ok := extraerRecordatorio(c.entrada)
		if !ok {
			t.Errorf("extraerRecordatorio(%q): esperaba ok=true, fue false", c.entrada)
			continue
		}
		if texto != c.textoEsperado {
			t.Errorf("extraerRecordatorio(%q): texto = %q, esperaba %q", c.entrada, texto, c.textoEsperado)
		}
		if momento.Hour() != c.horaEsperada || momento.Minute() != c.minEsperado {
			t.Errorf("extraerRecordatorio(%q): hora = %02d:%02d, esperaba %02d:%02d",
				c.entrada, momento.Hour(), momento.Minute(), c.horaEsperada, c.minEsperado)
		}
	}
}

func TestExtraerRecordatorio_CasosInvalidos(t *testing.T) {
	casos := []string{
		"recordá que me llamo Juan",     // no tiene prefijo de recordatorio con hora
		"recordame algo sin hora",       // no tiene "a las"
		"recordame a las 25 comer algo", // hora inválida (25)
		"abrir chrome",                  // no es un recordatorio en absoluto
		"",
	}

	for _, entrada := range casos {
		if _, _, ok := extraerRecordatorio(entrada); ok {
			t.Errorf("extraerRecordatorio(%q): esperaba ok=false, fue true", entrada)
		}
	}
}

func TestProximaOcurrencia_HoyOManana(t *testing.T) {
	ahora := time.Date(2026, time.July, 22, 15, 0, 0, 0, time.UTC)

	// Una hora más tarde en el mismo día: debería ser hoy.
	resultado := proximaOcurrencia(17, 0, ahora)
	if resultado.Day() != 22 || resultado.Hour() != 17 {
		t.Errorf("esperaba hoy a las 17:00, obtuve %v", resultado)
	}

	// Una hora que ya pasó: debería ser mañana.
	resultado = proximaOcurrencia(10, 0, ahora)
	if resultado.Day() != 23 || resultado.Hour() != 10 {
		t.Errorf("esperaba mañana a las 10:00, obtuve %v", resultado)
	}
}

func TestExtraerTimer_CasosValidos(t *testing.T) {
	casos := []struct {
		entrada           string
		duracionEsperada  time.Duration
	}{
		{"poné un timer de 5 minutos", 5 * time.Minute},
		{"temporizador de 30 segundos", 30 * time.Second},
		{"timer de 2 horas", 2 * time.Hour},
		{"avisame en 10 minutos", 10 * time.Minute},
		{"ponme un timer de 1 minuto", 1 * time.Minute},
	}

	for _, c := range casos {
		duracion, ok := extraerTimer(c.entrada)
		if !ok {
			t.Errorf("extraerTimer(%q): esperaba ok=true, fue false", c.entrada)
			continue
		}
		if duracion != c.duracionEsperada {
			t.Errorf("extraerTimer(%q): duración = %v, esperaba %v", c.entrada, duracion, c.duracionEsperada)
		}
	}
}

func TestExtraerTimer_CasosInvalidos(t *testing.T) {
	casos := []string{
		"recordame llamar a mamá a las 5", // es un recordatorio con hora, no un timer
		"poné un timer",                   // sin duración
		"timer de cero minutos",           // "cero" no es un número reconocido por \d+
		"abrir chrome",
		"",
	}
	for _, entrada := range casos {
		if _, ok := extraerTimer(entrada); ok {
			t.Errorf("extraerTimer(%q): esperaba ok=false, fue true", entrada)
		}
	}
}
