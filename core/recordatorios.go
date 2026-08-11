package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var patronHora = regexp.MustCompile(`(?:(?:a las|a la) (\d{1,2})(?::(\d{2}))?\s*(?:de la (mañana|tarde|noche))?|(\d{1,2}):(\d{2})\s*(?:hrs?)?)`)

var prefijosRecordatorio = []string{"recordame que ", "recordame ", "avisame que ", "avisame "}

func contenidoRecordatorio(entrada string) (string, bool) {
	lower := strings.ToLower(entrada)
	for _, p := range prefijosRecordatorio {
		if strings.HasPrefix(lower, p) {
			return strings.TrimSpace(entrada[len(p):]), true
		}
	}
	return "", false
}

var diasSemana = map[string]time.Weekday{
	"domingo": time.Sunday, "dom": time.Sunday,
	"lunes": time.Monday, "lun": time.Monday,
	"martes": time.Tuesday, "mar": time.Tuesday,
	"miercoles": time.Wednesday, "miércoles": time.Wednesday, "mie": time.Wednesday,
	"jueves": time.Thursday, "jue": time.Thursday,
	"viernes": time.Friday, "vie": time.Friday,
	"sabado": time.Saturday, "sábado": time.Saturday, "sab": time.Saturday,
}

var patronFecha = regexp.MustCompile(`\b(el|este)\s+(domingo|dom|lunes|lun|martes|mar|miercoles|miércoles|mie|jueves|jue|viernes|vie|sabado|sábado|sab)\b|\b(mañana|pasado mañana|pasado manana)\b|\b(cada día|cada semana|cada mes|todos los días|todas las semanas)\b|\b(el|este)\s+(\d{1,2})\s*de\s+(enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|octubre|noviembre|diciembre)\b`)

var meses = map[string]time.Month{
	"enero": time.January, "febrero": time.February, "marzo": time.March,
	"abril": time.April, "mayo": time.May, "junio": time.June,
	"julio": time.July, "agosto": time.August, "septiembre": time.September,
	"octubre": time.October, "noviembre": time.November, "diciembre": time.December,
}

func parsearHora(match []string) (hora, minuto int, ok bool) {
	if len(match) < 6 {
		return 0, 0, false
	}
	if match[1] != "" {
		h, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, 0, false
		}
		m := 0
		if match[2] != "" {
			m, err = strconv.Atoi(match[2])
			if err != nil {
				return 0, 0, false
			}
		}
		switch match[3] {
		case "tarde", "noche":
			if h < 12 {
				h += 12
			}
		case "mañana":
			if h == 12 {
				h = 0
			}
		}
		if h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, 0, false
		}
		return h, m, true
	}
	if match[4] != "" && match[5] != "" {
		h, err := strconv.Atoi(match[4])
		if err != nil {
			return 0, 0, false
		}
		m, err := strconv.Atoi(match[5])
		if err != nil {
			return 0, 0, false
		}
		if h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, 0, false
		}
		return h, m, true
	}
	return 0, 0, false
}

func proximaOcurrencia(hora, minuto int, ahora time.Time) time.Time {
	candidato := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), hora, minuto, 0, 0, ahora.Location())
	if candidato.Before(ahora) {
		candidato = candidato.Add(24 * time.Hour)
	}
	return candidato
}

func extraerRecordatorio(entrada string) (texto string, momento time.Time, ok bool) {
	return extraerRecordatorioEn(entrada, time.Now())
}

func extraerRecordatorioEn(entrada string, ahora time.Time) (texto string, momento time.Time, ok bool) {
	contenido, tieneContenido := contenidoRecordatorio(entrada)
	if !tieneContenido {
		return "", time.Time{}, false
	}

	periodo := ""
	fechaBase := ahora

	fechaLower := strings.ToLower(contenido)
	fechaMatch := patronFecha.FindStringSubmatch(fechaLower)

	if fechaMatch != nil {
		switch {
		case fechaMatch[1] != "" && fechaMatch[2] != "":
			if dia, ok := diasSemana[fechaMatch[2]]; ok {
				diasHasta := (int(dia) - int(ahora.Weekday()) + 7) % 7
				if diasHasta == 0 {
					// la regla diasHasta==0 → +7 significa "el jueves hoy agenda para la semana que viene"
					diasHasta = 7
				}
				fechaBase = ahora.Add(time.Duration(diasHasta) * 24 * time.Hour)
			}
		case fechaMatch[3] != "":
			switch fechaMatch[3] {
			case "mañana":
				fechaBase = ahora.Add(24 * time.Hour)
			case "pasado mañana", "pasado manana":
				fechaBase = ahora.Add(48 * time.Hour)
			}
		case fechaMatch[4] != "":
			// periodo es declarativo; el consumidor procesarMemoria (core/memoria_sesion.go) llama
			// AgregarRecordatorio sin periodo; el wiring a AgregarRecordatorioConPeriodo está fuera de alcance
			switch fechaMatch[4] {
			case "cada día", "todos los días":
				periodo = "diario"
			case "cada semana", "todas las semanas":
				periodo = "semanal"
			case "cada mes":
				periodo = "mensual"
			}
		case fechaMatch[6] != "" && fechaMatch[7] != "":
			dia, err := strconv.Atoi(fechaMatch[6])
			if err != nil || dia < 1 || dia > 31 {
				break
			}
			if mes, ok := meses[fechaMatch[7]]; ok {
				fechaBase = time.Date(ahora.Year(), mes, dia, 0, 0, 0, 0, ahora.Location())
				if fechaBase.Before(ahora) {
					fechaBase = fechaBase.AddDate(1, 0, 0)
				}
			}
		}
	}

	ubicacion := patronHora.FindStringSubmatchIndex(contenido)
	if ubicacion == nil {
		if periodo != "" {
			texto = strings.TrimSpace(contenido)
			_ = patronFecha.ReplaceAllString(texto, "")
			texto = strings.TrimSpace(patronFecha.ReplaceAllString(texto, ""))
			if texto == "" {
				return "", time.Time{}, false
			}
			return texto, proximaOcurrencia(9, 0, fechaBase), true
		}
		return "", time.Time{}, false
	}

	match := patronHora.FindStringSubmatch(contenido)
	hora, minuto, okHora := parsearHora(match)
	if !okHora {
		return "", time.Time{}, false
	}

	texto = strings.TrimSpace(contenido[:ubicacion[0]])
	texto = strings.TrimSpace(patronFecha.ReplaceAllString(texto, ""))
	if texto == "" {
		return "", time.Time{}, false
	}

	momento = time.Date(fechaBase.Year(), fechaBase.Month(), fechaBase.Day(), hora, minuto, 0, 0, ahora.Location())
	if !fechaBase.Equal(ahora) && momento.Before(ahora) {
		return "", time.Time{}, false
	}
	if fechaBase.Equal(ahora) && momento.Before(ahora) {
		momento = momento.Add(24 * time.Hour)
	}

	if periodo != "" {
		return texto, momento, true
	}

	return texto, momento, true
}

var patronDuracion = regexp.MustCompile(`(\d+)\s*(minutos?|segundos?|horas?)`)

var prefijosTimer = []string{
	"poné un timer de ", "pone un timer de ", "ponme un timer de ",
	"temporizador de ", "timer de ", "avisame en ",
}

func extraerTimer(entrada string) (duracion time.Duration, ok bool) {
	lower := strings.ToLower(entrada)
	var contenido string
	var encontrado bool
	for _, p := range prefijosTimer {
		if strings.HasPrefix(lower, p) {
			contenido = strings.TrimSpace(entrada[len(p):])
			encontrado = true
			break
		}
	}
	if !encontrado {
		return 0, false
	}

	match := patronDuracion.FindStringSubmatch(contenido)
	if match == nil {
		return 0, false
	}

	cantidad, err := strconv.Atoi(match[1])
	if err != nil || cantidad <= 0 {
		return 0, false
	}

	var unidad time.Duration
	switch {
	case strings.HasPrefix(match[2], "minuto"):
		unidad = time.Minute
	case strings.HasPrefix(match[2], "segundo"):
		unidad = time.Second
	case strings.HasPrefix(match[2], "hora"):
		unidad = time.Hour
	default:
		return 0, false
	}

	return time.Duration(cantidad) * unidad, true
}
