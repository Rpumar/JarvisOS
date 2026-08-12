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
		texto, _, momento, ok := extraerRecordatorio(c.entrada)
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
		if _, _, _, ok := extraerRecordatorio(entrada); ok {
			t.Errorf("extraerRecordatorio(%q): esperaba ok=false, fue true", entrada)
		}
	}
}

func TestExtraerRecordatorio_Fechas(t *testing.T) {
	now := time.Date(2026, time.July, 22, 15, 0, 0, 0, time.UTC)
	ahoraJueves := time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC)
	ahoraAgosto := time.Date(2026, time.August, 11, 15, 0, 0, 0, time.UTC)

	casos := []struct {
		entrada         string
		ahora           time.Time
		esperado        time.Time
		periodoEsperado string
		ok              bool
	}{
		{"recordame llamar a mamá el jueves a las 5", now, time.Date(2026, time.July, 23, 5, 0, 0, 0, time.UTC), "", true},
		{"recordame X el jueves a las 17", now, time.Date(2026, time.July, 23, 17, 0, 0, 0, time.UTC), "", true},
		{"recordame llamar a mamá el jueves a las 5", ahoraJueves, time.Date(2026, time.July, 30, 5, 0, 0, 0, time.UTC), "", true},
		{"recordame sacar la basura mañana a las 9", now, time.Date(2026, time.July, 23, 9, 0, 0, 0, time.UTC), "", true},
		{"recordame X pasado mañana a las 9", now, time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC), "", true},
		{"recordame X pasado manana a las 9", now, time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC), "", true},
		{"recordame llamar a mamá el 25 de julio a las 14:30", now, time.Date(2026, time.July, 25, 14, 30, 0, 0, time.UTC), "", true},
		{"recordame X el 25 de julio a las 14:30", ahoraAgosto, time.Date(2027, time.July, 25, 14, 30, 0, 0, time.UTC), "", true},
		{"recordame tomar agua cada día a las 8", now, time.Date(2026, time.July, 23, 8, 0, 0, 0, time.UTC), "diario", true},
		{"recordame X cada semana a las 9", now, time.Date(2026, time.July, 23, 9, 0, 0, 0, time.UTC), "semanal", true},
		{"recordame X cada día", now, time.Time{}, "diario", true},
	}

	for _, c := range casos {
		_, periodo, momento, ok := extraerRecordatorioEn(c.entrada, c.ahora)
		if ok != c.ok {
			t.Errorf("extraerRecordatorioEn(%q): ok = %v, esperaba %v", c.entrada, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if periodo != c.periodoEsperado {
			t.Errorf("extraerRecordatorioEn(%q): periodo = %q, esperaba %q", c.entrada, periodo, c.periodoEsperado)
		}
		if c.esperado.IsZero() {
			if momento.IsZero() {
				t.Errorf("extraerRecordatorioEn(%q): momento es zero, esperaba una fecha válida", c.entrada)
			}
			continue
		}
		if !momento.Equal(c.esperado) {
			t.Errorf("extraerRecordatorioEn(%q): momento = %v, esperaba %v", c.entrada, momento, c.esperado)
		}
	}
}

func TestExtraerRecordatorio_OrdenInvertido(t *testing.T) {
	texto, periodo, _, ok := extraerRecordatorio("todos los días a las 8, recordame la pastilla")
	if !ok {
		t.Fatal("esperaba ok=true para el orden invertido")
	}
	if periodo != "diario" {
		t.Errorf("periodo = %q, esperaba %q", periodo, "diario")
	}
	if texto != "la pastilla" {
		t.Errorf("texto = %q, esperaba %q", texto, "la pastilla")
	}

	now := time.Date(2026, time.July, 22, 15, 0, 0, 0, time.UTC)
	texto, periodo, momento, ok := extraerRecordatorioEn("recordame tomar la pastilla todos los días a las 8", now)
	if !ok {
		t.Fatal("esperaba ok=true para el orden normal")
	}
	if periodo != "diario" {
		t.Errorf("periodo = %q, esperaba %q", periodo, "diario")
	}
	esperado := time.Date(2026, time.July, 23, 8, 0, 0, 0, time.UTC)
	if !momento.Equal(esperado) {
		t.Errorf("momento = %v, esperaba %v", momento, esperado)
	}
	if texto != "tomar la pastilla" {
		t.Errorf("texto = %q, esperaba %q", texto, "tomar la pastilla")
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
