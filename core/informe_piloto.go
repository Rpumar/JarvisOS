package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MinutosEstimadosPorTarea es una estimación conservadora del tiempo manual
// que ahorra cada tarea/orden cumplida (informes, emails, planillas,
// formularios, etc.). Es la base del cálculo de "horas ahorradas" del piloto.
const MinutosEstimadosPorTarea = 20

// DiasPiloto es la ventana de la prueba piloto: 30 días corridos.
const DiasPiloto = 30

// InformePilotoDatos junta las métricas del período para armar el resumen de
// ROI. Se aísla en un struct para poder testear la generación sin tocar
// Windows ni disco.
type InformePilotoDatos struct {
	Desde             string
	Hasta             string
	OrdenesTerminadas int
	TareasHechas      int
	AccionesAuditadas int
	Aprobadas         int
	Denegadas         int
	Expiradas         int
	HorasAhorradas    float64
}

// GenerarInformePiloto arma el texto del informe de cierre de piloto: qué se
// cumplió en el período, cuántas horas manuales se estiman ahorradas por
// semana y cómo se comportaron las acciones sensibles (aprobadas/denegadas/
// expiradas). Es puro: no escribe ni habla, solo construye el mensaje.
func GenerarInformePiloto(d InformePilotoDatos) string {
	var b strings.Builder
	b.WriteString("Informe de cierre de piloto, señor.\n")
	b.WriteString(fmt.Sprintf("Período analizado: %s a %s.\n", d.Desde, d.Hasta))
	b.WriteString("Métricas de resultado:\n")

	totalTareas := d.OrdenesTerminadas + d.TareasHechas
	b.WriteString(fmt.Sprintf("  - Tareas cumplidas: %d (órdenes: %d, tareas: %d)\n", totalTareas, d.OrdenesTerminadas, d.TareasHechas))

	semanas := float64(DiasPiloto) / 7.0
	if semanas <= 0 {
		semanas = 1
	}
	horasPorSemana := d.HorasAhorradas / semanas
	b.WriteString(fmt.Sprintf("  - Horas manuales ahorradas: %.1f en total (≈%.1f por semana).\n", d.HorasAhorradas, horasPorSemana))

	b.WriteString("Control del dueño:\n")
	b.WriteString(fmt.Sprintf("  - Acciones auditadas: %d\n", d.AccionesAuditadas))
	b.WriteString(fmt.Sprintf("  - Aprobadas: %d | Denegadas: %d | Expiradas por timeout: %d\n", d.Aprobadas, d.Denegadas, d.Expiradas))
	if d.Denegadas+d.Expiradas == 0 {
		b.WriteString("  - Sin acciones sensibles rechazadas ni expiradas: el dueño mantuvo el control total.\n")
	}

	conclusion := "El empleado digital cumplió y rindió cuentas."
	if d.HorasAhorradas > 0 && d.HorasAhorradas >= 5*semanas {
		conclusion += " El ahorro estimado justifica ampliar el piloto a más puestos."
	} else if d.HorasAhorradas > 0 {
		conclusion += " El ahorro estimado confirma el valor del puesto."
	} else {
		conclusion += " El período no alcanzó a acumular suficientes tareas; evalúe dar más órdenes al puesto."
	}
	b.WriteString(conclusion + "\n")

	return b.String()
}

// RecolectarInformePiloto junta las métricas del período desde el Hands real,
// usando los mismos gestores que el resto del sistema. La ventana va desde
// hasta - DiasPiloto + 1 hasta hasta (inclusive).
func (h *Hands) RecolectarInformePiloto(hasta time.Time) InformePilotoDatos {
	d := InformePilotoDatos{}
	desde := hasta.AddDate(0, 0, -(DiasPiloto - 1))
	d.Desde = desde.Format("2006-01-02")
	d.Hasta = hasta.Format("2006-01-02")
	prefDesde := desde.Format("2006-01-02")

	if h.ordenes != nil {
		for _, o := range h.ordenes.Listar() {
			if o.Estado != OrdenTerminada {
				continue
			}
			if f := o.FechaCreacion; len(f) >= 10 && f[:10] >= prefDesde {
				d.OrdenesTerminadas++
			}
		}
	}

	if h.tareas != nil {
		for _, t := range h.tareas.ListarTodas() {
			if !t.Hecha || t.Completada == "" {
				continue
			}
			if len(t.Completada) >= 10 && t.Completada[:10] >= prefDesde {
				d.TareasHechas++
			}
		}
	}

	aprobadas := 0
	if h.Auditoria != nil {
		for _, e := range h.Auditoria.Recientes(5000) {
			if len(e.Momento) < 10 || e.Momento[:10] < prefDesde {
				continue
			}
			d.AccionesAuditadas++
			switch {
			case strings.Contains(e.Resultado, "aprobada") || strings.Contains(e.Resultado, "aprobado"):
				aprobadas++
			case strings.Contains(e.Resultado, "denegada") || strings.Contains(e.Resultado, "denegado"):
				d.Denegadas++
			case strings.Contains(e.Resultado, "expirado_por_timeout"):
				d.Expiradas++
			}
		}
	}
	// Las aprobaciones quedan en la auditoría con el resultado de la acción;
	// las órdenes registran la marca "(aprobada por el dueño)" en su historial,
	// que es la fuente confiable para contarlas.
	if aprobadas == 0 && h.ordenes != nil {
		for _, o := range h.ordenes.Listar() {
			if len(o.FechaCreacion) < 10 || o.FechaCreacion[:10] < prefDesde {
				continue
			}
			for _, a := range o.Historial {
				if strings.Contains(a.Resultado, "aprobada por el dueño") {
					aprobadas++
				}
			}
		}
	}
	d.Aprobadas = aprobadas

	totalTareas := d.OrdenesTerminadas + d.TareasHechas
	d.HorasAhorradas = float64(totalTareas) * float64(MinutosEstimadosPorTarea) / 60.0

	return d
}

// GuardarInformePiloto escribe el texto del informe en disco y devuelve la
// ruta del archivo creado, para que quede auditable y consultable.
func GuardarInformePiloto(rutaDir, texto string) (string, error) {
	if err := os.MkdirAll(rutaDir, 0o700); err != nil {
		return "", err
	}
	ruta := filepath.Join(rutaDir, "piloto-"+time.Now().Format("2006-01-02")+".txt")
	if err := os.WriteFile(ruta, []byte(texto), 0o600); err != nil {
		return "", err
	}
	return ruta, nil
}

// informesDir devuelve el directorio donde se guardan los informes diarios y
// los de cierre de piloto. Respeta DatosDir (sandbox de demo si está fijado).
func (h *Hands) informesDir() string {
	if h.DatosDir != "" {
		return filepath.Join(h.DatosDir, "informes")
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "informes")
}

// generarInformePilotoYGuardar arma el informe de piloto desde el Hands real
// y lo guarda en el directorio dado. Devuelve el texto y la ruta guardada.
func (h *Hands) generarInformePilotoYGuardar(informesDir string) (string, string) {
	d := h.RecolectarInformePiloto(time.Now())
	informe := GenerarInformePiloto(d)
	ruta, err := GuardarInformePiloto(informesDir, informe)
	if err != nil {
		return informe, ""
	}
	return informe, ruta
}
