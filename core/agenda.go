package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// === AGENDA (F3): calendario local de Jarvis ===
// Agenda persistida en JSON local (JarvisOS-datos/agenda.json), sin
// credenciales ni internet. Comandos: "agendá una reunión mañana a las 15",
// "qué tengo hoy", "cancelá el evento ...". Reutiliza patronHora/patronFecha
// de recordatorios.go.

// EventoAgenda es una entrada del calendario local.
type EventoAgenda struct {
	ID         int    `json:"id"`
	Titulo     string `json:"titulo"`
	Inicio     string `json:"inicio"` // RFC3339
	Fin        string `json:"fin"`     // RFC3339, puede estar vacío
	Ubicacion  string `json:"ubicacion,omitempty"`
	SincOutlook bool  `json:"sincOutlook,omitempty"`
}

// GestorAgenda persiste y consulta los eventos.
type GestorAgenda struct {
	mu      sync.RWMutex
	eventos []EventoAgenda
	ruta    string
}

func NuevoGestorAgenda(ruta string) *GestorAgenda {
	g := &GestorAgenda{ruta: ruta}
	g.cargar()
	return g
}

func (g *GestorAgenda) cargar() {
	contenido, err := os.ReadFile(g.ruta)
	if err != nil {
		g.eventos = []EventoAgenda{}
		return
	}
	if err := json.Unmarshal(contenido, &g.eventos); err != nil {
		g.eventos = []EventoAgenda{}
	}
}

func (g *GestorAgenda) guardar() error {
	dir := filepath.Dir(g.ruta)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	datos, err := json.MarshalIndent(g.eventos, "", "  ")
	if err != nil {
		return err
	}
	return escribirJSONAtomico(g.ruta, datos)
}

// Agregar crea un evento y devuelve su ID.
func (g *GestorAgenda) Agregar(titulo, inicio, fin string) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.siguienteIDLocked()
	g.eventos = append(g.eventos, EventoAgenda{ID: id, Titulo: titulo, Inicio: inicio, Fin: fin})
	return id, g.guardar()
}

func (g *GestorAgenda) siguienteIDLocked() int {
	max := 0
	for _, e := range g.eventos {
		if e.ID > max {
			max = e.ID
		}
	}
	return max + 1
}

// Listar devuelve los eventos ordenados por inicio.
func (g *GestorAgenda) Listar() []EventoAgenda {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cp := make([]EventoAgenda, len(g.eventos))
	copy(cp, g.eventos)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Inicio < cp[j].Inicio })
	return cp
}

// EventosEntre devuelve los eventos cuyo inicio cae en el rango [desde, hasta].
func (g *GestorAgenda) EventosEntre(desde, hasta time.Time) []EventoAgenda {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var resultado []EventoAgenda
	for _, e := range g.eventos {
		inicio, err := time.Parse(time.RFC3339, e.Inicio)
		if err != nil {
			continue
		}
		if !inicio.Before(desde) && inicio.Before(hasta) {
			resultado = append(resultado, e)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Inicio < resultado[j].Inicio })
	return resultado
}

// Cancelar elimina un evento por texto parcial del título. Devuelve cuántos
// se cancelaron y los títulos borrados.
func (g *GestorAgenda) Cancelar(termino string) (int, []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	termino = strings.ToLower(strings.TrimSpace(termino))
	var borrados []string
	quedan := g.eventos[:0]
	for _, e := range g.eventos {
		if termino != "" && strings.Contains(strings.ToLower(e.Titulo), termino) {
			borrados = append(borrados, e.Titulo)
			continue
		}
		quedan = append(quedan, e)
	}
	g.eventos = quedan
	if len(borrados) > 0 {
		_ = g.guardar()
	}
	return len(borrados), borrados
}

// Proximos devuelve hasta n eventos futuros (los más cercanos).
func (g *GestorAgenda) Proximos(n int, ahora time.Time) []EventoAgenda {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var resultado []EventoAgenda
	for _, e := range g.eventos {
		inicio, err := time.Parse(time.RFC3339, e.Inicio)
		if err != nil {
			continue
		}
		if inicio.After(ahora) {
			resultado = append(resultado, e)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Inicio < resultado[j].Inicio })
	if len(resultado) > n {
		resultado = resultado[:n]
	}
	return resultado
}

// PendientesOutlook devuelve los próximos eventos no sincronizados todavía
// con Outlook, ordenados por fecha de inicio.
func (g *GestorAgenda) PendientesOutlook(n int, ahora time.Time) []EventoAgenda {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var resultado []EventoAgenda
	for _, e := range g.eventos {
		if e.SincOutlook {
			continue
		}
		inicio, err := time.Parse(time.RFC3339, e.Inicio)
		if err != nil {
			continue
		}
		if inicio.After(ahora) {
			resultado = append(resultado, e)
		}
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Inicio < resultado[j].Inicio })
	if len(resultado) > n {
		resultado = resultado[:n]
	}
	return resultado
}

// MarcarSincOutlook marca un evento como exportado a Outlook.
func (g *GestorAgenda) MarcarSincOutlook(id int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.eventos {
		if g.eventos[i].ID == id {
			g.eventos[i].SincOutlook = true
			_ = g.guardar()
			return
		}
	}
}

// === Parseo de comandos de agenda ===

// fechaAgenda interpreta la fecha de un comando: hoy (default), mañana,
// pasado mañana, día de la semana, o "el N de MES". Devuelve la fecha base
// (a medianoche).
//
// Grupos de patronFecha: [1] (el|este), [2] día semana, [3] relativo
// (mañana|pasado mañana), [4] recurrencia, [5] (el|este), [6] día mes,
// [7] mes.
func fechaAgenda(contenido string, ahora time.Time) time.Time {
	lower := strings.ToLower(contenido)
	lower = strings.ReplaceAll(lower, "manana", "mañana")
	fechaMatch := patronFecha.FindStringSubmatch(lower)
	if fechaMatch != nil {
		switch {
		case fechaMatch[3] != "":
			switch fechaMatch[3] {
			case "mañana":
				return ahora.Add(24 * time.Hour)
			case "pasado mañana", "pasado manana":
				return ahora.Add(48 * time.Hour)
			}
		case fechaMatch[1] != "" && fechaMatch[2] != "":
			if dia, ok := diasSemana[fechaMatch[2]]; ok {
				diasHasta := (int(dia) - int(ahora.Weekday()) + 7) % 7
				if diasHasta == 0 {
					diasHasta = 7
				}
				return ahora.Add(time.Duration(diasHasta) * 24 * time.Hour)
			}
		case fechaMatch[6] != "" && fechaMatch[7] != "":
			dia, _ := strconv.Atoi(fechaMatch[6])
			if mes, ok := meses[fechaMatch[7]]; ok {
				base := time.Date(ahora.Year(), mes, dia, 0, 0, 0, 0, ahora.Location())
				if base.Before(ahora) {
					base = base.AddDate(1, 0, 0)
				}
				return base
			}
		}
	}
	return hoyMedianoche(ahora)
}

func hoyMedianoche(ahora time.Time) time.Time {
	return time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, ahora.Location())
}

// prefijosAgendar identifican el arranque de un comando de alta de evento.
var prefijosAgendar = []string{
	"agendá ", "agenda ", "agendame ", "agendáme ", "anotá en el calendario ",
	"anota en el calendario ", "marcá en el calendario ", "marca en el calendario ",
	"guardá en el calendario ", "guarda en el calendario ", "agregá al calendario ",
	"agrega al calendario ", "creá un evento ", "crea un evento ",
}

// extraerEvento interpreta "agendá <título> [mañana|el martes|el 5 de agosto]
// a las <hora> [en <lugar>]". Devuelve título, inicio, fin y ok.
func extraerEvento(comando string, ahora time.Time) (titulo, inicio, fin string, ok bool) {
	lower := strings.ToLower(comando)
	var contenido string
	encontrado := false
	for _, p := range prefijosAgendar {
		if strings.HasPrefix(lower, p) {
			contenido = strings.TrimSpace(comando[len(p):])
			encontrado = true
			break
		}
	}
	if !encontrado {
		return "", "", "", false
	}

	fechaBase := fechaAgenda(contenido, ahora)

	// Hora: "a las 15", "a las 9:30 de la tarde", "15:30"
	ubicacion := patronHora.FindStringSubmatchIndex(contenido)
	hora, minuto := 9, 0
	if ubicacion != nil {
		match := patronHora.FindStringSubmatch(contenido)
		if h, m, okHora := parsearHora(match); okHora {
			hora, minuto = h, m
		}
	}

	// El título es todo lo que está antes de la mención de fecha u hora.
	titulo = strings.TrimSpace(contenido)
	if ubicacion != nil {
		titulo = strings.TrimSpace(contenido[:ubicacion[0]])
	}
	titulo = strings.TrimSpace(patronFecha.ReplaceAllString(titulo, ""))
	titulo = strings.TrimSpace(strings.Trim(titulo, " ,-"))
	titulo = recortarArticulo(titulo)
	if titulo == "" {
		return "", "", "", false
	}

	inicioT := time.Date(fechaBase.Year(), fechaBase.Month(), fechaBase.Day(), hora, minuto, 0, 0, ahora.Location())
	if inicioT.Before(ahora) {
		return "", "", "", false
	}
	inicio = inicioT.Format(time.RFC3339)
	return titulo, inicio, "", true
}

// manejarAgenda despacha un comando de calendario: agendar, listar, cancelar.
func (h *Hands) manejarAgenda(comando string) string {
	if h.agenda == nil {
		return "No tengo calendario configurado, señor."
	}
	ahora := time.Now()

	if titulo, inicio, _, ok := extraerEvento(comando, ahora); ok {
		id, err := h.agenda.Agregar(titulo, inicio, "")
		if err != nil {
			return "No pude guardar el evento, señor: " + err.Error()
		}
		return fmt.Sprintf("Agendado, señor: '%s' el %s (#%d).", titulo, formatearEventoFecha(inicio), id)
	}

	lower := strings.ToLower(comando)
	switch {
	case strings.Contains(lower, "qué tengo hoy") || strings.Contains(lower, "que tengo hoy") ||
		strings.Contains(lower, "eventos de hoy") || strings.Contains(lower, "agenda de hoy"):
		return h.eventosDelDia(ahora)
	case strings.Contains(lower, "qué tengo mañana") || strings.Contains(lower, "que tengo manana") ||
		strings.Contains(lower, "eventos de mañana") || strings.Contains(lower, "eventos de manana"):
		return h.eventosDelDia(ahora.Add(24 * time.Hour))
	case strings.Contains(lower, "próximos eventos") || strings.Contains(lower, "proximos eventos") ||
		strings.Contains(lower, "qué se viene") || strings.Contains(lower, "que se viene"):
		return h.proximosEventos(ahora)
	case strings.Contains(lower, "cancelá el evento") || strings.Contains(lower, "cancela el evento") ||
		strings.Contains(lower, "borrá el evento") || strings.Contains(lower, "borra el evento"):
		termino := extraerObjeto(lower, []string{"cancelá el evento ", "cancela el evento ", "borrá el evento ", "borra el evento "})
		n, borrados := h.agenda.Cancelar(termino)
		if n == 0 {
			return fmt.Sprintf("No encontré ningún evento con '%s', señor.", termino)
		}
		return fmt.Sprintf("Cancelé %d evento(s): %s, señor.", n, strings.Join(borrados, ", "))
	case strings.Contains(lower, "qué eventos tengo") || strings.Contains(lower, "que eventos tengo") ||
		strings.Contains(lower, "lista de eventos") || strings.Contains(lower, "mostrame la agenda"):
		return h.proximosEventos(ahora)
	}

	return "Puedo agendar eventos ('agendá una reunión mañana a las 15'), decir qué tengo hoy, los próximos eventos o cancelar uno, señor."
}

func (h *Hands) eventosDelDia(ahora time.Time) string {
	desde := hoyMedianoche(ahora)
	hasta := desde.Add(24 * time.Hour)
	eventos := h.agenda.EventosEntre(desde, hasta)
	if len(eventos) == 0 {
		return "No tenés eventos ese día, señor."
	}
	return "Eventos: " + strings.Join(formatearEventos(eventos), " | ") + "."
}

func (h *Hands) proximosEventos(ahora time.Time) string {
	eventos := h.agenda.Proximos(10, ahora)
	if len(eventos) == 0 {
		return "No hay eventos próximos en la agenda, señor."
	}
	return "Próximos eventos: " + strings.Join(formatearEventos(eventos), " | ") + "."
}

func formatearEventos(eventos []EventoAgenda) []string {
	out := make([]string, 0, len(eventos))
	for _, e := range eventos {
		out = append(out, fmt.Sprintf("%s (%s)", e.Titulo, formatearEventoFecha(e.Inicio)))
	}
	return out
}

func formatearEventoFecha(rfc string) string {
	t, err := time.Parse(time.RFC3339, rfc)
	if err != nil {
		return rfc
	}
	return t.Format("02/01 a las 15:04")
}

// recortarArticulo quita un artículo determinante inicial del título
// ("una reunión" -> "reunión"), sin tocar el resto.
func recortarArticulo(titulo string) string {
	palabras := strings.Fields(titulo)
	if len(palabras) == 0 {
		return titulo
	}
	switch strings.ToLower(palabras[0]) {
	case "un", "una", "el", "la", "los", "las":
		return strings.TrimSpace(strings.Join(palabras[1:], " "))
	}
	return titulo
}
