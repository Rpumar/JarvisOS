package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"JarvisOS/core/audit"
)

// TiempoMaximoAprobacion es cuánto se espera el PIN del dueño antes de que
// la orden quede expirada automáticamente (5 minutos).
const TiempoMaximoAprobacion = 5 * time.Minute

// aprobacionOrden guarda el estado del bucle cuando este se detiene a
// esperar la aprobación del dueño, para reanudarlo tras el PIN.
type aprobacionOrden struct {
	Orden      Orden
	Prompt     string
	Ejecutadas []string
	Accion     string
}

// auditar registra cada comando ejecutado en el registro inmutable.
func (h *Hands) auditar(ordenID int, comando, resultado string) {
	if h.Auditoria == nil {
		return
	}
	h.Auditoria.Registrar(audit.Entrada{
		Momento:   time.Now().Format("2006-01-02 15:04:05"),
		Usuario:   "dueño",
		Rol:       "dueño",
		Orden:     ordenID,
		Comando:   comando,
		Resultado: resultado,
	})
}

// HashTexto devuelve el hash SHA-256 en hex de un texto (PIN, contraseña).
// Es la misma función para todo secreto del sistema: nunca se guarda en
// claro, solo su hash.
func HashTexto(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hashPIN(pin string) string {
	return HashTexto(pin)
}

// pinValido: si no hay PIN configurado, la aprobación explícita del dueño
// (panel o voz) alcanza. Si hay PIN, debe coincidir.
func (h *Hands) pinValido(pin string) bool {
	if h.PINHash == "" {
		return true
	}
	return hashPIN(pin) == h.PINHash
}

// AprobarOrden autoriza la acción sensible pendiente de una orden y
// reanuda el bucle del agente automáticamente.
func (h *Hands) AprobarOrden(id int, pin string) string {
	h.ExpirarAprobacionesAntiguas(TiempoMaximoAprobacion)

	h.aprobacionMu.Lock()
	pend := h.aprobacionPendiente
	h.aprobacionMu.Unlock()

	if pend == nil || pend.Orden.ID != id {
		return h.aprobarOrdenPersistida(id, pin)
	}

	if !h.pinValido(pin) {
		return "PIN incorrecto, señor. La acción sensible no se ejecutó."
	}

	h.aprobacionMu.Lock()
	h.aprobacionPendiente = nil
	h.aprobacionMu.Unlock()

	h.ordenes.LimpiarAprobacion(id)
	h.ordenes.CambiarEstado(id, OrdenEnProgreso)

	resultado := h.RunCommand(pend.Accion)
	h.ordenes.RegistrarAccion(id, pend.Accion, resultado+" (aprobada por el dueño)")
	h.auditar(id, pend.Accion, resultado)

	if resultado == ComandoNoReconocido {
		pend.Prompt += fmt.Sprintf("\n[Resultado de '%s']: comando no reconocido. Usá otro comando de la lista.", pend.Accion)
	} else {
		pend.Ejecutadas = append(pend.Ejecutadas, pend.Accion)
		pend.Prompt += fmt.Sprintf("\n[Acción aprobada por el dueño. Resultado de '%s']: %s", pend.Accion, resultado)
	}
	return h.bucleAgente(pend.Orden, pend.Prompt, pend.Ejecutadas)
}

// aprobarOrdenPersistida cubre el caso de una aprobación pendiente que
// sobrevivió a un reinicio (no hay estado del bucle en memoria).
func (h *Hands) aprobarOrdenPersistida(id int, pin string) string {
	o, ok := h.ordenes.Obtener(id)
	if !ok {
		return fmt.Sprintf("No encontré la orden #%d, señor.", id)
	}
	if o.Estado != OrdenEsperandoAprobacion {
		return fmt.Sprintf("La orden #%d no está esperando aprobación, señor. Está %s.", id, o.Estado)
	}
	if !h.pinValido(pin) {
		return "PIN incorrecto, señor. La acción sensible no se ejecutó."
	}
	if o.PendienteAccion == "" {
		return "No hay ninguna acción pendiente que aprobar, señor."
	}
	h.ordenes.LimpiarAprobacion(id)
	h.ordenes.CambiarEstado(id, OrdenEnProgreso)
	resultado := h.RunCommand(o.PendienteAccion)
	h.ordenes.RegistrarAccion(id, o.PendienteAccion, resultado+" (aprobada por el dueño)")
	h.auditar(id, o.PendienteAccion, resultado)
	return fmt.Sprintf("Acción '%s' aprobada y ejecutada, señor: %s. Diga 'ejecutá la orden #%d' para seguir trabajándola.", o.PendienteAccion, resultado, id)
}

// DenegarOrden descarta la aprobación pendiente y deja la orden bloqueada.
func (h *Hands) DenegarOrden(id int) string {
	h.ExpirarAprobacionesAntiguas(TiempoMaximoAprobacion)

	h.aprobacionMu.Lock()
	if h.aprobacionPendiente != nil && h.aprobacionPendiente.Orden.ID == id {
		h.aprobacionPendiente = nil
	}
	h.aprobacionMu.Unlock()

	if o, ok := h.ordenes.Obtener(id); ok {
		desc := o.PendienteDescripcion
		if h.ordenes.DenegarAprobacion(id) {
			if desc != "" {
				h.auditar(id, desc, "denegada_por_el_dueño")
			}
			return fmt.Sprintf("Orden #%d denegada, señor. La acción sensible no se ejecutó.", id)
		}
		return fmt.Sprintf("No encontré la orden #%d, señor.", id)
	}
	return fmt.Sprintf("No encontré la orden #%d, señor.", id)
}

// EstablecerPIN configura el PIN del dueño (4-6 dígitos) para las
// aprobaciones. Persiste a través de PINSetter (config).
func (h *Hands) EstablecerPIN(pin string) string {
	pin = strings.TrimSpace(pin)
	if len(pin) < 4 || len(pin) > 6 {
		return "El PIN debe tener entre 4 y 6 dígitos, señor."
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return "El PIN debe ser solo números, señor."
		}
	}
	hash := hashPIN(pin)
	if h.PINSetter != nil && !h.PINSetter(hash) {
		return "No pude guardar el PIN, señor. Verifique los permisos del archivo de configuración."
	}
	h.PINHash = hash
	return "PIN configurado, señor. A partir de ahora, las acciones sensibles requieren su PIN."
}

// OrdenesParaPanel devuelve las órdenes activas para el panel del dueño.
func (h *Hands) OrdenesParaPanel() []Orden {
	h.ExpirarAprobacionesAntiguas(TiempoMaximoAprobacion)
	if h.ordenes == nil {
		return nil
	}
	return h.ordenes.Activas()
}

// ExpirarAprobacionesAntiguas revisa las órdenes que esperan el PIN del
// dueño y expira las que superaron el timeout dado: la orden queda en estado
// "expirada", la acción sensible se descarta y el hecho se audita como
// "expirado_por_timeout_aprobacion". Devuelve un resumen hablable, o vacío
// si no había nada que expirar.
func (h *Hands) ExpirarAprobacionesAntiguas(timeout time.Duration) string {
	if h.ordenes == nil {
		return ""
	}
	ahora := time.Now()
	expiradas := 0
	for _, o := range h.ordenes.EsperandoAprobacion() {
		desde, err := time.Parse(time.RFC3339, o.PendienteDesde)
		if err != nil || ahora.Sub(desde) < timeout {
			continue
		}
		h.aprobacionMu.Lock()
		if h.aprobacionPendiente != nil && h.aprobacionPendiente.Orden.ID == o.ID {
			h.aprobacionPendiente = nil
		}
		h.aprobacionMu.Unlock()
		if h.ordenes.ExpirarAprobacion(o.ID) {
			h.auditar(o.ID, o.PendienteAccion, "expirado_por_timeout_aprobacion")
			expiradas++
		}
	}
	if expiradas > 0 {
		return fmt.Sprintf("La aprobación de %d orden(es) expiró por superar el tiempo máximo de espera, señor. Ejecute de nuevo la orden para volver a solicitarla.", expiradas)
	}
	return ""
}
