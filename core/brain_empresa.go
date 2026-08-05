package core

import (
	"fmt"
	"regexp"
	"strings"
)

// manejarEmpresa atiende los comandos de voz sobre el perfil de empresa:
// consultar, cargar campos escalares, agregar/borrar elementos de listas.
// Usa la entrada original (con mayúsculas) para preservar el valor y la
// versión en minúsculas para detectar la frase. Devuelve false si el texto no
// era un comando de empresa.
func (b *Brain) manejarEmpresa(original string) (string, bool) {
	if b.empresa == nil {
		return "", false
	}
	original = strings.TrimSpace(original)
	entrada := strings.ToLower(original)
	if entrada == "" {
		return "", false
	}

	// Consultas.
	if contieneAlguna(entrada, []string{
		"que sabes de mi empresa", "qué sabés de mi empresa",
		"perfil de mi empresa", "perfil de la empresa", "perfil de empresa",
		"mostrame mi perfil de empresa", "mostrame mi perfil", "decime mi perfil",
		"que tenes de mi empresa", "qué tenés de mi empresa", "como esta mi empresa",
	}) {
		return b.empresa.Resumen(), true
	}

	// Guía de carga.
	if contieneAlguna(entrada, []string{
		"configura mi empresa", "configurá mi empresa", "arma mi perfil de empresa",
		"armá mi perfil de empresa", "empeza con mi perfil", "empezá con mi perfil",
		"cargame mi empresa", "cargame mi perfil", "configura mi perfil",
	}) {
		return "Vamos a cargar tu empresa, señor. Por ejemplo: 'mi empresa se llama X y es rubro Y', 'somos Z empleados', 'mi producto principal es C', 'agendá un objetivo: bater'. También puedo guardar clientes, competidores y redes.", true
	}

	// Borrar elementos de una lista.
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"borra mi producto", "borrá mi producto", "saca mi producto", "sacá mi producto",
	}); ok && clave == "productos" {
		if err := b.empresa.BorrarItem("productos", valor); err == nil {
			return fmt.Sprintf("Saqué '%s' de los productos, señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"borra mi cliente", "borrá mi cliente", "saca mi cliente", "sacá mi cliente",
	}); ok && clave == "clientes" {
		if err := b.empresa.BorrarItem("clientes", valor); err == nil {
			return fmt.Sprintf("Saqué '%s' de los clientes, señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"borra mi objetivo", "borrá mi objetivo", "olvida mi objetivo", "olvidá mi objetivo",
	}); ok && clave == "objetivos" {
		if err := b.empresa.BorrarItem("objetivos", valor); err == nil {
			return fmt.Sprintf("Saqué '%s' de los objetivos, señor.", valor), true
		}
	}

	// Listas: productos, objetivos, clientes, competidores, redes.
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"mi producto principal es", "mi producto principal", "mi producto es", "nuestro producto es",
		"agrega mi producto", "agregá mi producto", "agrega un producto", "agregá un producto",
		"guarda mi producto", "guardá mi producto", "guarda el producto", "guardá el producto",
	}); ok && clave == "productos" {
		if err := b.empresa.AgregarItem("productos", valor); err == nil {
			return fmt.Sprintf("Agregué '%s' a los productos, señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"mi objetivo es", "nuestro objetivo es", "agenda un objetivo", "agendá un objetivo",
		"agrega un objetivo", "agregá un objetivo", "guarda un objetivo", "guardá un objetivo",
		"guarda el objetivo", "guardá el objetivo", "una meta es", "mi meta es",
		"agenda esta meta", "agendá esta meta",
	}); ok && clave == "objetivos" {
		if err := b.empresa.AgregarItem("objetivos", valor); err == nil {
			return fmt.Sprintf("Agendé el objetivo '%s', señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"mis clientes son", "mis clientes tipicos son", "mis clientes típicos son", "mi publico es",
		"agrega mi cliente", "agregá mi cliente", "guarda mi cliente", "guardá mi cliente",
		"mi cliente tipico es", "mi cliente típico es",
	}); ok && clave == "clientes" {
		if err := b.empresa.AgregarItem("clientes", valor); err == nil {
			return fmt.Sprintf("Agregué '%s' a los clientes típicos, señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"mi competidor es", "mi competencia es", "agrega mi competidor", "agregá mi competidor",
	}); ok && clave == "competidores" {
		if err := b.empresa.AgregarItem("competidores", valor); err == nil {
			return fmt.Sprintf("Agregué '%s' a los competidores, señor.", valor), true
		}
	}
	if clave, valor, ok := extraerPreposicion(entrada, original, []string{
		"mi red es", "mi red social es", "agrega mi red", "agregá mi red",
		"estoy en", "estamos en", "estamos activos en",
	}); ok && clave == "redes" {
		if err := b.empresa.AgregarItem("redes", valor); err == nil {
			return fmt.Sprintf("Agregué '%s' a las redes sociales, señor.", valor), true
		}
	}

	// Campos escalares.
	if valor, ok := extraerValor(entrada, original, []string{
		"el rubro de mi empresa es", "mi rubro es", "es del rubro", "es de rubro",
		"mi empresa se dedica a", "me dedico a",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("rubro", valor); err == nil {
			return fmt.Sprintf("Rubro guardado: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerValor(entrada, original, []string{
		"mi empresa se llama", "la empresa se llama", "la empresa es",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("nombre", valor); err == nil {
			return fmt.Sprintf("Nombre guardado: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerEmpleados(entrada, original); ok {
		if err := b.empresa.SetCampo("tamano", valor); err == nil {
			return fmt.Sprintf("Tamaño guardado: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerValor(entrada, original, []string{
		"facturamos aproximadamente", "facturamos alrededor de", "facturamos por mes",
		"facturamos", "facturo por mes",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("facturacion", valor); err == nil {
			return fmt.Sprintf("Facturación guardada: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerValor(entrada, original, []string{
		"mi email de contacto es", "mi email es", "mi correo es", "mi mail es",
		"el email de contacto es", "contactame en", "contacto al email",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("email", valor); err == nil {
			return fmt.Sprintf("Email guardado: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerValor(entrada, original, []string{
		"mi telefono es", "mi teléfono es", "mi whatsapp es", "el telefono de contacto es",
		"contactame al", "contacto al telefono",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("telefono", valor); err == nil {
			return fmt.Sprintf("Teléfono guardado: %s, señor.", valor), true
		}
	}
	if valor, ok := extraerValor(entrada, original, []string{
		"el dueño se llama", "el dueno se llama", "el dueño soy yo", "el dueno soy yo",
	}); ok && valor != "" {
		if err := b.empresa.SetCampo("dueno", valor); err == nil {
			return fmt.Sprintf("Dueño guardado: %s, señor.", valor), true
		}
	}

	return "", false
}

// extraerPreposicion detecta cuál lista se toca y captura el valor completo que
// le sigue (en el original), recortado en separadores razonables.
func extraerPreposicion(entrada, original string, prefijos []string) (clave, valor string, ok bool) {
	for _, p := range prefijos {
		idx := strings.Index(entrada, p)
		if idx < 0 {
			continue
		}
		// El índice pertenece a la minúscula; igual sufijo de prefijo original.
		resto := strings.TrimSpace(original[idx+len(p):])
		resto = strings.TrimLeft(resto, " :;.,-_")
		resto = recortarAfuera(resto)
		if resto == "" {
			continue
		}
		switch {
		case contieneAlguna(p, []string{"producto", "productos"}):
			return "productos", resto, true
		case contieneAlguna(p, []string{"objetivo", "objetivos", "meta"}):
			return "objetivos", resto, true
		case contieneAlguna(p, []string{"cliente", "clientes", "publico"}):
			return "clientes", resto, true
		case contieneAlguna(p, []string{"competidor", "competencia"}):
			return "competidores", resto, true
		case contieneAlguna(p, []string{"red", "redes", "estamos en", "estoy en"}):
			return "redes", resto, true
		}
	}
	return "", "", false
}

// extraerValor captura el valor que sigue a un prefijo de campo escalar.
func extraerValor(entrada, original string, prefijos []string) (string, bool) {
	for _, p := range prefijos {
		idx := strings.Index(entrada, p)
		if idx < 0 {
			continue
		}
		resto := strings.TrimSpace(original[idx+len(p):])
		resto = strings.TrimLeft(resto, " :;.,-_")
		resto = recortarAfuera(resto)
		if resto == "" {
			return "", false
		}
		return resto, true
	}
	return "", false
}

// recortarAfuera corta el texto en separadores narrativos (" y ", ".", ",",
// " porque ", " para ", " con "), pero conserva emails y URLs enteras.
func recortarAfuera(s string) string {
	abre := func(i int) bool {
		r := s[:i]
		return strings.Contains(r, "@") || strings.Contains(r, ".com") || strings.Contains(r, ".net")
	}
	seps := []string{" y ", ". ", ", ", " porque ", " para ", " con ", " también "}
	idx := -1
	for _, sep := range seps {
		i := strings.Index(s, sep)
		if i >= 0 && (idx < 0 || i < idx) && !abre(i) {
			idx = i
		}
	}
	if idx < 0 {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(s[:idx])
}

// extraerEmpleados captura "somos N empleados" / "mi empresa tiene N
// empleados" y lo normaliza como "N empleados".
func extraerEmpleados(entrada, original string) (string, bool) {
	re := regexp.MustCompile(`(?i)(somos|trabajamos|tenemos|mi empresa tiene)\s+([\d\s]+)\s+empleados?`)
	m := re.FindStringSubmatch(original)
	if len(m) < 3 {
		return "", false
	}
	return strings.TrimSpace(m[2]) + " empleados", true
}