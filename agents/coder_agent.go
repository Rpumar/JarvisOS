// Package agents reúne agentes especializados de JarvisOS que van más allá
// de comandos fijos.
package agents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GeneradorDeCodigo es la interfaz mínima que CoderAgent necesita de un
// conector de IA. Se define acá (no se importa el tipo concreto de ia) para
// no acoplar este paquete a un proveedor específico.
type GeneradorDeCodigo interface {
	Disponible() bool
	ConsultarCodigo(peticion string) (codigo string, explicacion string, err error)
}

// CoderAgent genera pequeños scripts de PowerShell a pedido usando un LLM,
// y los ejecuta ÚNICAMENTE después de una confirmación explícita del usuario.
//
// DECISIÓN DE SEGURIDAD (no es un detalle menor, es central al diseño):
// ejecutar código generado por una IA sin que un humano lo revise -aunque
// sea brevemente- es un riesgo real. Un error del modelo, una petición mal
// reconocida por el micrófono, o texto manipulado pueden traducirse en una
// acción destructiva sobre el equipo. Por eso:
//  1. Proponer() NUNCA ejecuta nada: genera el script, lo guarda en disco
//     (para que se pueda inspeccionar) y deja una propuesta pendiente.
//  2. Confirmar() es el único camino a la ejecución, y requiere que el
//     usuario lo pida explícitamente en un turno de voz aparte.
//  3. Además de la confirmación, se bloquean patrones conocidos de alto
//     riesgo (formatear, borrar carpetas del sistema, apagar el equipo,
//     deshabilitar seguridad, descargar y ejecutar cosas de internet)
//     incluso si el usuario confirmaría: eso ni siquiera llega a
//     proponerse. No es configurable a propósito.
//  4. config.RequireApproval NO controla este comportamiento: acá la
//     confirmación es obligatoria siempre, no una opción para desactivar.
//
// LIMITACIÓN HONESTA: la lista de patrones bloqueados es una defensa
// adicional (defensa en profundidad), no una garantía completa — un texto
// suficientemente distinto podría evadir coincidencias exactas de substring.
// La protección principal sigue siendo el paso de confirmación humana.
type CoderAgent struct {
	ia        GeneradorDeCodigo
	propuesta *propuestaCodigo
}

type propuestaCodigo struct {
	ruta        string
	explicacion string
}

// patronesBloqueados son fragmentos (en minúscula) que, si aparecen en un
// script generado, hacen que CoderAgent se niegue a proponerlo. No es una
// lista exhaustiva; ver la nota de "limitación honesta" arriba.
var patronesBloqueados = []string{
	"format-volume", "format ",
	"diskpart",
	"shutdown", "restart-computer", "stop-computer",
	"disable-windowsoptionalfeature",
	"set-mppreference",
	"reg delete",
	"bcdedit",
	"cipher /w",
	"net user",
	"invoke-webrequest", "invoke-restmethod", "start-bitstransfer",
	"iex ", "invoke-expression",
	"remove-item -recurse -force $env:windir",
	"remove-item -recurse -force c:\\windows",
	"remove-item -recurse -force c:\\users",
	"remove-item -recurse -force $env:systemroot",
	// Los mismos procesos críticos que core.esProcesoProtegido bloquea en
	// cerrarApp (Fase 4): el mismo riesgo existe acá si un script generado
	// los mata por su cuenta en vez de pasar por Hands.
	"stop-process -name explorer", "stop-process -name winlogon", "stop-process -name csrss",
	"stop-process -name lsass", "stop-process -name smss", "stop-process -name services",
	"stop-process -name wininit", "stop-process -name svchost",
	"taskkill /im explorer", 	"taskkill /im winlogon", "taskkill /im csrss",
	"taskkill /im lsass", "taskkill /im smss", "taskkill /im services",
	"taskkill /im wininit", "taskkill /im svchost",
	// Bloqueo adicional: instalación, desinstalación, descargas y compras
	"install-package", "install-module", "install-windowsfeature",
	"uninstall-package", "uninstall-module",
	"choco install", "choco uninstall", "choco upgrade",
	"winget install", "winget uninstall",
	"invoke-webrequest -outfile", "invoke-webrequest -uri",
	"start-bitstransfer -destination",
	"add-type -assemblyname system.windows.forms",
	"invoke-item",
	// Comandos que alteran el sistema de forma crítica
	"clear-item", "remove-item -recurse",
	"remove-item -force",
	"start-sleep -seconds 0",
	"register-scheduledjob", "register-scheduledtask",
	"set-executionpolicy",
	"new-itemproperty -path",
	"add-localgroupmember", "remove-localgroupmember",
}

// NewCoderAgent crea el agente. ia normalmente es un *ia.Conector, pero
// puede ser cualquier implementación de GeneradorDeCodigo.
func NewCoderAgent(ia GeneradorDeCodigo) *CoderAgent {
	return &CoderAgent{ia: ia}
}

// Proponer le pide a la IA un script para cumplir peticion. Nunca ejecuta
// nada: como mucho, guarda el script en disco y deja una propuesta a la
// espera de Confirmar().
func (c *CoderAgent) Proponer(peticion string) (string, error) {
	if !c.ia.Disponible() {
		return "No tengo una IA configurada para generar código, señor. Hace falta tener Ollama funcionando.", nil
	}

	codigo, explicacion, err := c.ia.ConsultarCodigo(peticion)
	if err != nil {
		return "", fmt.Errorf("no pude generar el script: %w", err)
	}

	if codigo == "" {
		c.propuesta = nil
		if explicacion == "" {
			explicacion = "No pude generar un script para eso, señor."
		}
		return explicacion, nil
	}

	if patronPeligrosoEn(codigo) {
		c.propuesta = nil
		return "Ese script incluye una operación que considero demasiado riesgosa (podría borrar datos, afectar la seguridad o apagar el equipo) y prefiero no proponerlo, señor. Si de verdad lo necesita, hágalo manualmente.", nil
	}

	carpeta := filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-scripts")
	if err := os.MkdirAll(carpeta, 0o700); err != nil {
		return "", fmt.Errorf("no pude preparar la carpeta de scripts: %w", err)
	}
	ruta := filepath.Join(carpeta, fmt.Sprintf("script_%d.ps1", time.Now().Unix()))
	if err := os.WriteFile(ruta, []byte(codigo), 0o600); err != nil {
		return "", fmt.Errorf("no pude guardar el script: %w", err)
	}

	c.propuesta = &propuestaCodigo{ruta: ruta, explicacion: explicacion}
	return fmt.Sprintf("%s Lo guardé en %s por si quiere revisarlo antes. ¿Confirmo y lo ejecuto, señor? Diga 'confirmar' o 'cancelar'.", explicacion, ruta), nil
}

// Confirmar ejecuta la propuesta pendiente, si hay una. Sea cual sea el
// resultado, la propuesta se descarta después: no hay reintentos implícitos.
func (c *CoderAgent) Confirmar() string {
	if c.propuesta == nil {
		return "No tengo ninguna propuesta pendiente, señor."
	}
	propuesta := c.propuesta
	c.propuesta = nil

	salida, err := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", propuesta.ruta).CombinedOutput()
	resultado := strings.TrimSpace(string(salida))
	if len(resultado) > 200 {
		resultado = resultado[:200] + "... (salida truncada, ver el archivo)"
	}

	if err != nil {
		return fmt.Sprintf("El script falló al ejecutarlo: %v. Salida: %s", err, resultado)
	}
	if resultado == "" {
		return fmt.Sprintf("Listo, señor. Se ejecutó sin errores. Script guardado en %s.", propuesta.ruta)
	}
	return fmt.Sprintf("Listo, señor. Resultado: %s", resultado)
}

// Cancelar descarta la propuesta pendiente sin ejecutar nada. El archivo
// .ps1 queda guardado igual, como rastro de lo que se propuso y se rechazó.
func (c *CoderAgent) Cancelar() string {
	if c.propuesta == nil {
		return "No tengo ninguna propuesta pendiente, señor."
	}
	c.propuesta = nil
	return "Descartado, señor. No se ejecutó nada."
}

// TienePropuestaPendiente indica si hay una propuesta esperando confirmación.
func (c *CoderAgent) TienePropuestaPendiente() bool {
	return c.propuesta != nil
}

func patronPeligrosoEn(codigo string) bool {
	codigoMin := strings.ToLower(codigo)
	for _, patron := range patronesBloqueados {
		if strings.Contains(codigoMin, patron) {
			return true
		}
	}
	return false
}
