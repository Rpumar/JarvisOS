package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ============================================================
// LICENCIA LOCAL (decisión 4.1): activación por clave sin
// servidor. La clave codifica el plan y la cantidad de puestos,
// firmada con HMAC-SHA256 sobre un secreto embebido: sirve para
// control comercial (plan/puestos), no para resistir a alguien
// que desensamble el binario. Validada al arrancar.
// ============================================================

// Planes de licencia y sus puestos por defecto.
const (
	PlanLite    = "lite"
	PlanPro     = "pro"
	PlanEmpresa = "empresa"

	PuestosLite    = 1
	PuestosPro     = 5
	PuestosEmpresa = 50
)

// PuestosPorPlan devuelve la cantidad de puestos del plan, o 0 si es inválido.
func PuestosPorPlan(plan string) int {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case PlanLite:
		return PuestosLite
	case PlanPro:
		return PuestosPro
	case PlanEmpresa:
		return PuestosEmpresa
	}
	return 0
}

// PlanValido indica si el plan existe.
func PlanValido(plan string) bool {
	return PuestosPorPlan(plan) > 0
}

// secretoLicencia es el secreto embebido para firmar las claves. No es
// seguridad real (se puede extraer del binario), pero evita claves forjadas
// por error y ordena el negocio por plan/puestos.
const secretoLicencia = "JarvisOS-licencia-local-v1-sin-servidor"

// claveLicencia es una clave ya firmada. El campo Plan es el plan (lite/pro/
// empresa), Puestos su tope, Nonce un valor aleatorio y Firma el HMAC.
type claveLicencia struct {
	Plan    string
	Puestos int
	Nonce   string
	Firma   string
}

// GenerarLicencia crea una clave firmada para el plan y cantidad de puestos
// dados. Devuelve error si el plan no existe o los puestos no son válidos.
func GenerarLicencia(plan string, puestos int) (string, error) {
	plan = strings.ToLower(strings.TrimSpace(plan))
	if !PlanValido(plan) {
		return "", fmt.Errorf("plan de licencia inválido: %q", plan)
	}
	if puestos <= 0 {
		return "", fmt.Errorf("la cantidad de puestos debe ser mayor a 0")
	}
	nonceBytes := make([]byte, 6)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	nonce := strings.ToUpper(hex.EncodeToString(nonceBytes))
	k := claveLicencia{Plan: plan, Puestos: puestos, Nonce: nonce}
	k.Firma = firmarLicencia(k)
	return k.String(), nil
}

// LicenciaValida valida la firma y el formato de una clave. Devuelve la clave
// decodificada y true si es legítima.
func LicenciaValida(clave string) (claveLicencia, bool) {
	k, err := decodificarLicencia(clave)
	if err != nil {
		return claveLicencia{}, false
	}
	firmaEsperada := firmarLicencia(k)
	return k, hmac.Equal([]byte(k.Firma), []byte(firmaEsperada))
}

// PuestosLicencia devuelve el tope de puestos que habilita la clave (0 =
// sin licencia o inválida = modo piloto sin límite de puestos).
func PuestosLicencia(clave string) int {
	clave = strings.TrimSpace(clave)
	if clave == "" {
		return 0
	}
	k, ok := LicenciaValida(clave)
	if !ok {
		return 0
	}
	return k.Puestos
}

// EstadoLicencia describe una clave: plan + puestos, para el banner y los
// comandos de voz. Si la clave está vacía, avisa que se está en modo piloto.
func EstadoLicencia(clave string) string {
	if strings.TrimSpace(clave) == "" {
		return "Sin licencia: modo piloto (sin límite de puestos). Diga 'activá la licencia JARVIS-...' para activar su plan."
	}
	k, ok := LicenciaValida(clave)
	if !ok {
		return "La clave de licencia configurada no es válida. Verifíquela o pida una nueva."
	}
	return fmt.Sprintf("Licencia %s activa: %d puesto(s).", strings.ToUpper(k.Plan), k.Puestos)
}

// String serializa la clave al formato JARVIS-PLAN-PUESTOS-NONCE-FIRMA.
func (k claveLicencia) String() string {
	return fmt.Sprintf("JARVIS-%s-%03d-%s-%s", strings.ToUpper(k.Plan), k.Puestos, k.Nonce, k.Firma)
}

// firmarLicencia calcula el HMAC-SHA256 (8 bytes hex) de la clave sin firma.
func firmarLicencia(k claveLicencia) string {
	material := fmt.Sprintf("%s|%d|%s", strings.ToLower(k.Plan), k.Puestos, k.Nonce)
	mac := hmac.New(sha256.New, []byte(secretoLicencia))
	mac.Write([]byte(material))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil))[:16])
}

// decodificarLicencia parsea el formato JARVIS-PLAN-PUESTOS-NONCE-FIRMA.
func decodificarLicencia(clave string) (claveLicencia, error) {
	partes := strings.Split(strings.TrimSpace(clave), "-")
	if len(partes) != 5 || partes[0] != "JARVIS" {
		return claveLicencia{}, fmt.Errorf("formato de licencia inválido")
	}
	plan := strings.ToLower(partes[1])
	if !PlanValido(plan) {
		return claveLicencia{}, fmt.Errorf("plan inválido: %q", partes[1])
	}
	puestos := 0
	if _, err := fmt.Sscanf(partes[2], "%d", &puestos); err != nil || puestos <= 0 {
		return claveLicencia{}, fmt.Errorf("puestos inválidos: %q", partes[2])
	}
	if len(partes[3]) != 12 {
		return claveLicencia{}, fmt.Errorf("nonce inválido")
	}
	if len(partes[4]) != 16 {
		return claveLicencia{}, fmt.Errorf("firma inválida")
	}
	return claveLicencia{Plan: plan, Puestos: puestos, Nonce: strings.ToUpper(partes[3]), Firma: strings.ToUpper(partes[4])}, nil
}

// extraerLicenciaDelComando busca una clave tipo JARVIS-... en el texto del
// comando de voz (puede venir pegada al comando o sola).
func extraerLicenciaDelComando(entrada string) string {
	for _, campo := range strings.Fields(entrada) {
		campo = strings.Trim(campo, ".,;")
		if strings.HasPrefix(strings.ToUpper(campo), "JARVIS-") {
			return strings.ToUpper(campo)
		}
	}
	return ""
}

// ============================================================
// COMANDOS DE VOZ: LICENCIA
// ============================================================

// consultarLicencia reporta el estado de la licencia configurada.
func (h *Hands) consultarLicencia() string {
	return EstadoLicencia(h.LicenseKey)
}

// activarLicencia guarda una clave nueva y valida que sea legítima. Devuelve
// un mensaje amable; solo persiste si la firma es válida.
func (h *Hands) activarLicencia(entrada string) string {
	clave := extraerLicenciaDelComando(entrada)
	if clave == "" {
		return "No le entendí la clave, señor. Dígala completa, por ejemplo: 'activá la licencia JARVIS-PRO-005-ABC123DEF456-1234567890ABCDEF'."
	}
	_, ok := LicenciaValida(clave)
	if !ok {
		return "Esa clave de licencia no es válida, señor. Verifique el plan, los puestos y que esté completa. Puede pedir una nueva."
	}
	if h.LicenseSetter != nil && !h.LicenseSetter(clave) {
		return "No pude guardar la licencia, señor. Verifique los permisos del archivo de configuración."
	}
	h.LicenseKey = clave
	if h.perfil != nil {
		h.perfil.LimitePuestos = PuestosLicencia(clave)
	}
	return EstadoLicencia(clave) + " Actividad correcta."
}
