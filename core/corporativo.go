package core

import (
	"fmt"
	"strings"
)

// prefijosComandoCorporativo es la whitelist de comandos que el traductor
// corporativo puede generar. Solo acciones de agenda, tareas, recordatorios,
// notas u órdenes: nunca comandos de sistema. Esto garantiza que la IA no
// pueda inducir a Jarvis a ejecutar algo peligroso.
var prefijosComandoCorporativo = []string{
	"agendá ", "agenda ", "agendame ", "agendáme ",
	"agendá una tarea", "agenda una tarea",
	"recordame ", "avisame ",
	"tomá nota ", "toma nota ", "anotá ", "anota ", "recordá que ", "recuerda que ",
	"agendá una orden", "agenda una orden",
}

// esComandoCorporativoSeguro valida que el comando generado por la IA arranque
// con uno de los prefijos permitidos.
func esComandoCorporativoSeguro(comando string) bool {
	lower := strings.ToLower(strings.TrimSpace(comando))
	for _, p := range prefijosComandoCorporativo {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// intencionCorporativa devuelve true si el rol asistente_corporativo está
// activo (modo persistente o sugerido por el turno) o la skill
// asistente-corporativo se disparó con esta petición.
func (b *Brain) intencionCorporativa(entrada string) bool {
	if b.roles != nil {
		if r := b.roles.RolActivo(); r != nil && r.Nombre == "asistente_corporativo" {
			return true
		}
		for _, s := range b.roles.Sugerir(entrada) {
			if s.Nombre == "asistente_corporativo" {
				return true
			}
		}
	}
	if b.skills != nil {
		for _, s := range b.skills.Buscar(entrada) {
			if s.Nombre == "asistente-corporativo" {
				return true
			}
		}
	}
	return false
}

// traducirInstruccionCorporativa pide a la IA convertir una instrucción vaga
// de un cliente en un comando Jarvis exacto (agenda, tarea, recordatorio,
// nota u orden). Devuelve el comando validado o false si no se pudo traducir
// a una acción soportada.
func (b *Brain) traducirInstruccionCorporativa(input string) (string, bool) {
	if b.ia == nil || !b.ia.Disponible() {
		return "", false
	}
	prompt := `Convertí la instrucción del usuario en un comando exacto que JarvisOS pueda ejecutar.
Usá únicamente uno de estos formatos, en ese orden de preferencia:
- agendá <evento> [mañana|el martes|el 5 de agosto] [a las HH:MM]   (reuniones, citas, eventos)
- agendá una tarea <nombre> [para <cuándo>]                          (tareas pendientes)
- recordame <texto> [a las HH:MM | mañana | el lunes | cada día]     (recordatorios)
- tomá nota <texto>                                                  (notas libres)
- agendá una orden <objetivo>                                        (órdenes abiertas)
Respondé SOLO con el comando, sin explicaciones, sin comillas ni viñetas.
Si la instrucción no se puede convertir en ninguna de esas acciones, respondé exactamente: NINGUNA

Instrucción: ` + input
	resp, err := b.ia.Consultar(prompt, nil)
	if err != nil || strings.TrimSpace(resp) == "" {
		return "", false
	}
	comando := strings.Trim(strings.TrimSpace(resp), `"'`)
	if strings.EqualFold(comando, "NINGUNA") || !esComandoCorporativoSeguro(comando) {
		return "", false
	}
	return comando, true
}

// manejarInstruccionCorporativa intenta interpretar y aplicar una instrucción
// vaga cuando hay intención corporativa. Devuelve una respuesta lista para
// mostrar y true si tomó la instrucción (queda pendiente de confirmación).
// Si no aplica, devuelve ("", false) y el flujo normal sigue.
func (b *Brain) manejarInstruccionCorporativa(input, entrada string) (string, bool) {
	if !b.intencionCorporativa(entrada) {
		return "", false
	}
	comando, ok := b.traducirInstruccionCorporativa(input)
	if !ok {
		return "", false
	}
	if _, peligroso := esAccionPeligrosa(comando); peligroso {
		return "", false
	}
	ejecutar := func() string {
		return b.hands.RunCommand(comando)
	}
	b.confirmacionPendiente = &accionConfirmable{
		ejecutar:    ejecutar,
		descripcion: "interpretar su instrucción y ejecutar: " + comando,
	}
	return fmt.Sprintf("Entendido, señor. Interpreté su instrucción como: \"%s\". Diga 'sí' para ejecutarla o 'no' para cancelar.", comando), true
}
