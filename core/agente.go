package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ============================================================
// BUCLE DE AGENTE: la IA decide qué acciones ejecutar para
// cumplir una orden, ejecuta, observa el resultado, ajusta y
// aprende. Es el "empleado que no abandona": si conoce el
// procedimiento lo usa directo; si no, la IA lo resuelve paso
// a paso y queda aprendido para la próxima.
// ============================================================

const (
	maxIteracionesIA = 10
	maxErroresJSON   = 3
)

// respuestaAgente es el contrato JSON estricto que la IA debe
// devolver en cada paso del bucle.
type respuestaAgente struct {
	Accion string `json:"accion"`
	Fin    string `json:"fin"`
	Razon  string `json:"razon"`
}

var errRespuestaNoJSON = errors.New("la IA no respondió JSON válido")

// ejecutarOrdenConIA cumple una orden guiando a la IA con el
// catálogo de herramientas. Devuelve el mensaje final al dueño.
func (h *Hands) ejecutarOrdenConIA(orden Orden) string {
	return h.bucleAgente(orden, construirPromptAgente(orden.Objetivo), nil)
}

// verificarOrdenConIA reanuda el bucle del agente con pasos ya
// ejecutados (procedimiento conocido): la IA revisa el resultado
// y decide si falta algo más o si la orden quedó cumplida.
func (h *Hands) verificarOrdenConIA(orden Orden, prompt string, ejecutadas []string) string {
	return h.bucleAgente(orden, prompt, ejecutadas)
}

// bucleAgente ejecuta la iteración IA -> acción -> resultado hasta
// que la IA declara FIN o se alcanza el límite de pasos. Si la IA
// responde JSON malformado, se le re-pide con el error en el prompt
// (hasta maxErroresJSON veces antes de bloquear).
func (h *Hands) bucleAgente(orden Orden, prompt string, ejecutadasPrevias []string) string {
	h.ejecucionMu.Lock()
	defer h.ejecucionMu.Unlock()

	h.ordenes.CambiarEstado(orden.ID, OrdenEnProgreso)
	h.ordenes.RegistrarAccion(orden.ID, "iniciar orden con IA", orden.Objetivo)

	ejecutadas := make([]string, 0, 8)
	ejecutadas = append(ejecutadas, ejecutadasPrevias...)

	erroresJSON := 0

	for i := 0; i < maxIteracionesIA; i++ {
		resp, err := h.IA.Consultar(prompt, nil)
		if err != nil {
			h.ordenes.RegistrarAccion(orden.ID, "consulta IA", "error: "+err.Error())
			h.ordenes.CambiarEstado(orden.ID, OrdenBloqueada)
			return fmt.Sprintf("La IA falló al avanzar la orden #%d: %v, señor. Quedó en espera de revisión.", orden.ID, err)
		}
		resp = strings.TrimSpace(resp)
		if resp == "" {
			h.ordenes.CambiarEstado(orden.ID, OrdenBloqueada)
			return fmt.Sprintf("La IA no respondió, señor. La orden #%d quedó en espera.", orden.ID)
		}

		r, errParse := parsearRespuestaIA(resp)
		if errParse != nil {
			erroresJSON++
			prompt += fmt.Sprintf("\n[Error: tu respuesta no era JSON válido. Respondé SOLO con un objeto JSON, sin texto: {\"accion\":\"<comando del catálogo>\",\"razon\":\"<por qué>\"} o {\"fin\":\"<resumen>\"}. Respuesta recibida: %q]", resp)
			if erroresJSON >= maxErroresJSON {
				h.ordenes.RegistrarAccion(orden.ID, "consulta IA", "JSON inválido repetido")
				h.ordenes.CambiarEstado(orden.ID, OrdenBloqueada)
				return fmt.Sprintf("La IA insistió con respuestas que no eran JSON, señor. La orden #%d quedó en espera de revisión.", orden.ID)
			}
			continue
		}

		if r.Fin != "" {
			h.ordenes.RegistrarAccion(orden.ID, "reporte IA", r.Fin)
			if len(ejecutadas) > 0 && h.procedimientos != nil {
				h.procedimientos.Crear(orden.Objetivo, ejecutadas)
				h.ordenes.RegistrarAccion(orden.ID, "aprender", fmt.Sprintf("procedimiento '%s' aprendido (%d pasos)", orden.Objetivo, len(ejecutadas)))
			}
			h.ordenes.Terminar(orden.ID, r.Fin)
			return fmt.Sprintf("Orden #%d cumplida, señor. %s", orden.ID, r.Fin)
		}

		if descripcion, peligrosa := esAccionPeligrosa(r.Accion); peligrosa {
			h.aprobacionMu.Lock()
			h.aprobacionPendiente = &aprobacionOrden{
				Orden:      orden,
				Prompt:     prompt,
				Ejecutadas: ejecutadas,
				Accion:     r.Accion,
			}
			h.aprobacionMu.Unlock()
			h.ordenes.SolicitarAprobacion(orden.ID, r.Accion, descripcion)
			h.ordenes.RegistrarAccion(orden.ID, "accion sensible", r.Accion+" ("+descripcion+")")
			h.auditar(orden.ID, r.Accion, "requiere aprobación del dueño ("+descripcion+")")
			return fmt.Sprintf("La IA propuso %s (%s), señor. Eso requiere su aprobación: apruebe desde el panel, o diga 'aprobar la orden #%d' (y el PIN si lo configuró). Para rechazarla, 'denegar la orden #%d'.", r.Accion, descripcion, orden.ID, orden.ID)
		}

		resultado := h.RunCommand(r.Accion)
		h.auditar(orden.ID, r.Accion, resultado)
		if resultado == ComandoNoReconocido {
			h.ordenes.RegistrarAccion(orden.ID, r.Accion, "comando no reconocido")
			prompt += fmt.Sprintf("\n[Resultado de '%s']: comando no reconocido. Usá otro comando de la lista.", r.Accion)
			continue
		}
		h.ordenes.RegistrarAccion(orden.ID, r.Accion, resultado)
		ejecutadas = append(ejecutadas, r.Accion)
		prompt += fmt.Sprintf("\n[Resultado de '%s']: %s", r.Accion, resultado)
	}

	h.ordenes.CambiarEstado(orden.ID, OrdenBloqueada)
	return fmt.Sprintf("La IA alcanzó el límite de pasos sin cerrar la orden #%d, señor. La dejé en espera para su revisión.", orden.ID)
}

// parsearRespuestaIA interpreta la respuesta del agente, que debe
// ser un objeto JSON de la forma:
//   - {"accion":"<comando>","razon":"<por qué>"} -> ejecutar herramienta
//   - {"fin":"<resumen>"}                          -> la orden está cumplida
//
// Toleran texto y marcas de código alrededor del JSON. Devuelve
// errRespuestaNoJSON si no hay un objeto válido.
func parsearRespuestaIA(resp string) (respuestaAgente, error) {
	obj := extraerJSON(resp)
	if obj == "" {
		return respuestaAgente{}, errRespuestaNoJSON
	}
	var r respuestaAgente
	if err := json.Unmarshal([]byte(obj), &r); err != nil {
		return respuestaAgente{}, errRespuestaNoJSON
	}
	r.Accion = strings.TrimSpace(r.Accion)
	r.Fin = strings.TrimSpace(r.Fin)
	r.Razon = strings.TrimSpace(r.Razon)
	if r.Accion == "" && r.Fin == "" {
		return respuestaAgente{}, errRespuestaNoJSON
	}
	if r.Accion != "" && r.Fin != "" {
		return respuestaAgente{}, errRespuestaNoJSON
	}
	return r, nil
}

// extraerJSON toma el primer objeto JSON de la respuesta, tolerando
// código markdown y texto alrededor.
func extraerJSON(resp string) string {
	inicio := strings.Index(resp, "{")
	if inicio < 0 {
		return ""
	}
	fin := strings.Index(resp[inicio:], "}")
	if fin < 0 {
		return ""
	}
	return resp[inicio : inicio+fin+1]
}

// construirPromptAgente arma las instrucciones que la IA recibe
// junto con el catálogo de herramientas disponibles.
func construirPromptAgente(objetivo string) string {
	var b strings.Builder
	b.WriteString("Sos Jarvis, el empleado digital de la empresa. Tenés una orden que cumplir:\n")
	b.WriteString(fmt.Sprintf("ORDEN: %s\n\n", objetivo))
	b.WriteString("Solo podés actuar usando los comandos de este catálogo:\n")
	b.WriteString(catalogoAgente())
	b.WriteString("\nReglas:\n")
	b.WriteString("- Respondé SOLO con un objeto JSON, sin texto adicional ni marcas de código.\n")
	b.WriteString("- Para ejecutar una herramienta: {\"accion\":\"<comando del catálogo>\",\"razon\":\"<por qué>\"}\n")
	b.WriteString("- Cuando la orden esté cumplida: {\"fin\":\"<resumen de lo hecho y el resultado>\"}\n")
	b.WriteString("- No inventes comandos. Si no sabés cómo hacer algo, usá {\"fin\":\"<explicación de qué necesitás>\"}\n")
	b.WriteString("- Nada de acciones de instalar, borrar, formatear o comprar.\n")
	return b.String()
}

func catalogoAgente() string {
	return `- abrir [chrome | word | excel | powerpoint | vscode | notepad | calculadora | outlook | teams | zoom | paint | terminal | cmd | powershell | fotos | música | videos | calendario]
- cerrar [app]
- buscar archivo [nombre]
- crear archivo [ruta]
- crear carpeta [ruta]
- abrir carpeta [ruta]
- abrir página web [url]
- buscar en internet [consulta]
- tomar nota [texto]
- recordá que [texto]
- captura de pantalla
- enviar notificación [texto]
- copiar [texto] al portapapeles
- qué hora es / qué fecha es
- cuánta ram me queda / espacio en disco / batería / procesador
- listar procesos / matar proceso [nombre]
- listar tareas / qué tareas tengo
- listar órdenes / qué órdenes tengo
- ejecutá el procedimiento [nombre]
- crear rutina [nombre] que [pasos]
- ejecutar rutina [nombre]
- ver clima / ver noticias
- enviar email a [dirección] con asunto [asunto] y el texto [texto] (exige aprobación del dueño)`
}
