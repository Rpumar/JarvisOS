package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"JarvisOS/core/audit"
)

// HoraInformeDiario es la hora (local, 24h) a partir de la cual el vigía
// emite el informe diario del día que termina.
const HoraInformeDiario = 19

// InformeDiarioDatos junta todos los datos del día para armar el resumen.
// Se aísla en un struct para poder testear la generación sin tocar
// Windows ni disco.
type InformeDiarioDatos struct {
	TareasPendientes   []Tarea
	TareasHechasHoy    []Tarea
	OrdenesActivas     []Orden
	OrdenesTerminadas  []Orden
	ActividadHoy       []audit.Entrada
	EventosManana      []EventoAgenda
}

// GenerarInformeDiario arma el texto del informe del día que termina. Es
// puro: no escribe ni habla, solo construye el mensaje.
func GenerarInformeDiario(d InformeDiarioDatos) string {
	var b strings.Builder
	b.WriteString("Informe diario, señor.\n")

	if len(d.OrdenesTerminadas) > 0 {
		b.WriteString(fmt.Sprintf("Órdenes cumplidas: %d\n", len(d.OrdenesTerminadas)))
		for _, o := range d.OrdenesTerminadas {
			b.WriteString(fmt.Sprintf("  - #%d %s\n", o.ID, o.Objetivo))
		}
	}

	if len(d.OrdenesActivas) > 0 {
		b.WriteString(fmt.Sprintf("Órdenes aún en juego: %d\n", len(d.OrdenesActivas)))
		for _, o := range d.OrdenesActivas {
			b.WriteString(fmt.Sprintf("  - #%d [%s] %s\n", o.ID, o.Estado, o.Objetivo))
		}
	}

	if len(d.TareasPendientes) > 0 {
		b.WriteString(fmt.Sprintf("Tareas pendientes: %d\n", len(d.TareasPendientes)))
		for _, t := range d.TareasPendientes {
			b.WriteString(fmt.Sprintf("  - %s\n", t.Nombre))
		}
	}

	if len(d.ActividadHoy) > 0 {
		b.WriteString(fmt.Sprintf("Acciones registradas hoy: %d\n", len(d.ActividadHoy)))
		m := d.ActividadHoy
		if len(m) > 10 {
			m = m[:10]
		}
		for _, e := range m {
			linea := strings.TrimSpace(e.Comando)
			if res := strings.TrimSpace(e.Resultado); res != "" && res != linea {
				linea += " -> " + res
			}
			b.WriteString(fmt.Sprintf("  - [%s] %s\n", e.Momento, linea))
		}
		if len(d.ActividadHoy) > 10 {
			b.WriteString(fmt.Sprintf("  ...y %d más.\n", len(d.ActividadHoy)-10))
		}
	}

	if len(d.EventosManana) > 0 {
		b.WriteString("Para mañana:\n")
		for _, e := range d.EventosManana {
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", e.Titulo, e.Inicio))
		}
	}

	return b.String()
}

// RecolectarInformeDiario junta los datos del día desde el Hands real,
// usando los mismos gestores que el resto del sistema.
func (h *Hands) RecolectarInformeDiario(hoy time.Time) InformeDiarioDatos {
	d := InformeDiarioDatos{}
	prefDia := hoy.Format("2006-01-02")

	if h.tareas != nil {
		d.TareasPendientes = h.tareas.ListarPendientes()
		for _, t := range h.tareas.ListarTodas() {
			if strings.HasPrefix(t.Completada, prefDia) {
				d.TareasHechasHoy = append(d.TareasHechasHoy, t)
			}
		}
	}

	if h.ordenes != nil {
		for _, o := range h.ordenes.Listar() {
			if o.Estado == OrdenTerminada {
				d.OrdenesTerminadas = append(d.OrdenesTerminadas, o)
			} else if o.Estado != OrdenCancelada {
				d.OrdenesActivas = append(d.OrdenesActivas, o)
			}
		}
	}

	if h.Auditoria != nil {
		for _, e := range h.Auditoria.Recientes(500) {
			if strings.HasPrefix(e.Momento, prefDia) {
				d.ActividadHoy = append(d.ActividadHoy, e)
			}
		}
	}

	if h.agenda != nil {
		manana := hoy.Add(24 * time.Hour)
		d.EventosManana = h.agenda.EventosEntre(manana, manana.Add(24*time.Hour))
	}

	return d
}

// GuardarInformeDiario escribe el texto del informe en disco y devuelve la
// ruta del archivo creado, para que quede auditable y consultable.
func GuardarInformeDiario(rutaDir, fecha, texto string) (string, error) {
	if err := os.MkdirAll(rutaDir, 0o700); err != nil {
		return "", err
	}
	ruta := filepath.Join(rutaDir, fecha+".txt")
	if err := os.WriteFile(ruta, []byte(texto), 0o600); err != nil {
		return "", err
	}
	return ruta, nil
}