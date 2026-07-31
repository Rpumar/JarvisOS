package core

import (
	"fmt"
	"strings"
	"time"
)

var frasesCerrarSinObjeto = []string{
	"ciérralo", "cierralo", "ciérrala", "cierrala", "cerrar eso", "cerralo",
}

var frasesRepetirBusquedaEnYoutube = []string{
	"lo mismo en youtube", "buscá lo mismo en youtube", "busca lo mismo en youtube", "eso mismo en youtube",
}

var frasesQueDijiste = []string{"qué dijiste", "que dijiste", "repetí eso", "repeti eso", "podés repetir", "podes repetir"}

func (b *Brain) resolverPronombres(entrada string) string {
	for _, f := range frasesCerrarSinObjeto {
		if entrada == f && b.ultimaApp != "" {
			return "cerrar " + b.ultimaApp
		}
	}
	for _, f := range frasesRepetirBusquedaEnYoutube {
		if strings.Contains(entrada, f) && b.ultimaBusqueda != "" {
			return "buscar en youtube " + b.ultimaBusqueda
		}
	}
	return entrada
}

func (b *Brain) actualizarMemoriaDeSesion(entrada string) {
	if strings.HasPrefix(entrada, "abrir ") {
		b.ultimaApp = strings.TrimSpace(strings.TrimPrefix(entrada, "abrir "))
		return
	}
	if strings.HasPrefix(entrada, "buscar en youtube ") {
		b.ultimaBusqueda = strings.TrimSpace(strings.TrimPrefix(entrada, "buscar en youtube "))
		return
	}
	if strings.HasPrefix(entrada, "buscar ") {
		b.ultimaBusqueda = strings.TrimSpace(strings.TrimPrefix(entrada, "buscar "))
	}
}

var prefijosRecordar = []string{
	"recordá que ", "recorda que ", "acordate que ", "acuérdate que ",
	"recordame que ", "recordame ", "avisame que ", "avisame ",
}

var patronesNombre = []string{"me llamo ", "mi nombre es ", "llamame ", "llámame ", "decime "}
var patronesCiudad = []string{"vivo en ", "mi ciudad es "}
var patronesCumpleanos = []string{"mi cumpleaños es ", "cumplo años el "}
var patronesTrabajo = []string{"trabajo en ", "trabajo de ", "mi trabajo es "}

var frasesPreguntaNombre = []string{"cuál es mi nombre", "cual es mi nombre", "cómo me llamo", "como me llamo"}
var frasesPreguntaCiudad = []string{"dónde vivo", "donde vivo", "cuál es mi ciudad", "cual es mi ciudad"}
var frasesPreguntaCumpleanos = []string{"cuándo es mi cumpleaños", "cuando es mi cumpleaños", "cuál es mi cumpleaños"}
var frasesPreguntaTrabajo = []string{"dónde trabajo", "donde trabajo", "cuál es mi trabajo", "cual es mi trabajo"}
var frasesPreguntaNotas = []string{"qué recordás", "que recordas", "qué notas tenés", "que notas tenes", "qué me pediste recordar"}
var frasesPreguntaRecordatorios = []string{"qué recordatorios tengo", "que recordatorios tengo", "qué avisos tengo"}
var frasesCancelarTodos = []string{"cancelá todos los recordatorios", "cancela todos los recordatorios", "cancelá mis recordatorios", "cancela mis recordatorios"}
var prefijosCancelarRecordatorio = []string{
	"cancelá el recordatorio de ", "cancela el recordatorio de ",
	"cancelá los recordatorios de ", "cancela los recordatorios de ",
	"cancelá el recordatorio sobre ", "cancela el recordatorio sobre ",
}

var prefijosCrearLista = []string{"creá una lista de ", "crea una lista de ", "creá la lista ", "crea la lista ", "nueva lista "}
var prefijosAgregarLista = []string{"agregá ", "agrega ", "añadí ", "añade ", "poné en la lista ", "pone en la lista "}
var prefijosMostrarListas = []string{"mostrame las listas", "mostrame mis listas", "qué listas tengo", "que listas tengo", "mostrar listas"}
var prefijosMostrarLista = []string{"mostrame la lista de ", "mostrame la lista ", "abrir lista "}
var prefijosMarcarHecho = []string{"marcá como hecho ", "marca como hecho ", "marcá como completado ", "marca como completado ", "marcá ", "marca "}
var prefijosBuscarNotas = []string{"buscá en mis notas ", "busca en mis notas ", "buscá en notas ", "busca en notas "}
var prefijosEliminarLista = []string{"eliminá la lista ", "elimina la lista ", "borrá la lista ", "borra la lista "}

func (b *Brain) procesarMemoria(original string) (string, bool) {
	entrada := strings.ToLower(original)

	if duracion, ok := extraerTimer(original); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		momento := time.Now().Add(duracion)
		if err := b.mem.AgregarRecordatorio("¡Timer terminado!", momento); err != nil {
			return fmt.Sprintf("No pude iniciar el timer, señor: %v", err), true
		}
		return fmt.Sprintf("Timer de %s en marcha, señor.", duracion), true
	}

	if texto, momento, ok := extraerRecordatorio(original); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		if err := b.mem.AgregarRecordatorio(texto, momento); err != nil {
			return fmt.Sprintf("No pude programar el recordatorio, señor: %v", err), true
		}
		return fmt.Sprintf("Anotado, señor. Le aviso el %s a las %s.", momento.Format("2006-01-02"), momento.Format("15:04")), true
	}

	if contieneAlguna(entrada, frasesCancelarTodos) {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		n, err := b.mem.CancelarRecordatorios("")
		if err != nil {
			return fmt.Sprintf("No pude cancelarlos, señor: %v", err), true
		}
		if n == 0 {
			return "No tenía ningún recordatorio pendiente, señor.", true
		}
		return fmt.Sprintf("Cancelé sus %d recordatorios, señor.", n), true
	}
	if busqueda, ok := primerPrefijoQueCoincide(entrada, prefijosCancelarRecordatorio); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		n, err := b.mem.CancelarRecordatorios(busqueda)
		if err != nil {
			return fmt.Sprintf("No pude cancelarlo, señor: %v", err), true
		}
		if n == 0 {
			return fmt.Sprintf("No encontré ningún recordatorio pendiente sobre '%s', señor.", busqueda), true
		}
		return "Cancelado, señor.", true
	}

	if clave, valor, esHecho := extraerHechoDirecto(original); esHecho {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		if err := b.mem.GuardarHecho(clave, valor); err != nil {
			return fmt.Sprintf("No pude guardarlo, señor: %v", err), true
		}
		b.sincronizarPrefs(clave, valor)
		return fmt.Sprintf("Anotado. Ya sé que su %s es %s.", clave, valor), true
	}

	if contenido, ok := extraerContenidoRecordar(original); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		if clave, valor, esHecho := extraerHecho(contenido); esHecho {
			if err := b.mem.GuardarHecho(clave, valor); err != nil {
				return fmt.Sprintf("No pude guardarlo, señor: %v", err), true
			}
			b.sincronizarPrefs(clave, valor)
			return fmt.Sprintf("Anotado. Ya sé que su %s es %s.", clave, valor), true
		}
		if err := b.mem.AgregarNota(contenido); err != nil {
			return fmt.Sprintf("No pude guardar la nota, señor: %v", err), true
		}
		return "Anotado, señor.", true
	}

	if contieneAlguna(entrada, frasesPreguntaNombre) {
		return b.responderHecho("nombre", "Se llama %s, señor. ¿O acaso lo olvidó?", "Todavía no me dijo su nombre, señor.")
	}
	if contieneAlguna(entrada, frasesPreguntaCiudad) {
		return b.responderHecho("ciudad", "Vive en %s, según lo que me contó.", "No sé dónde vive, señor. Todavía no me lo dijo.")
	}
	if contieneAlguna(entrada, frasesPreguntaCumpleanos) {
		return b.responderHecho("cumpleaños", "Su cumpleaños es el %s, señor.", "No sé cuándo es su cumpleaños. Todavía no me lo dijo.")
	}
	if contieneAlguna(entrada, frasesPreguntaTrabajo) {
		return b.responderHecho("trabajo", "Trabaja en %s, según lo que me contó.", "No sé dónde trabaja, señor. Todavía no me lo dijo.")
	}

	if contieneAlguna(entrada, frasesPreguntaNotas) {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		notas := b.mem.ObtenerNotas()
		if len(notas) == 0 {
			return "No tengo ninguna nota guardada, señor.", true
		}
		fmt.Println("Notas guardadas:")
		for _, n := range notas {
			fmt.Println(" -", n)
		}
		return fmt.Sprintf("Tengo %d notas guardadas. La última: %s. Mire la consola para ver todas.", len(notas), notas[len(notas)-1]), true
	}

	if contieneAlguna(entrada, frasesPreguntaRecordatorios) {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		pendientes := b.mem.ObtenerRecordatoriosPendientesTexto()
		if len(pendientes) == 0 {
			return "No tiene ningún recordatorio pendiente, señor.", true
		}
		fmt.Println("Recordatorios pendientes:")
		for _, p := range pendientes {
			fmt.Println(" -", p)
		}
		return fmt.Sprintf("Tiene %d recordatorios pendientes. Mire la consola para ver todos.", len(pendientes)), true
	}

	if texto, ok := primerPrefijoQueCoincide(entrada, prefijosBuscarNotas); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		resultados := b.mem.BuscarNotas(texto)
		if len(resultados) == 0 {
			return fmt.Sprintf("No encontré notas sobre '%s', señor.", texto), true
		}
		fmt.Println("Notas encontradas:")
		for _, n := range resultados {
			fmt.Println(" -", n)
		}
		return fmt.Sprintf("Encontré %d notas sobre '%s'. Mire la consola, señor.", len(resultados), texto), true
	}

	if contieneAlguna(entrada, prefijosMostrarListas) {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		listas := b.mem.ObtenerListas()
		if len(listas) == 0 {
			return "No tiene ninguna lista guardada, señor.", true
		}
		fmt.Println("Listas:")
		for _, l := range listas {
			fmt.Println(l)
		}
		return fmt.Sprintf("Tiene %d listas guardadas. Mire la consola, señor.", len(listas)), true
	}

	if nombreLista, ok := primerPrefijoQueCoincide(entrada, prefijosMostrarLista); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		lista, encontrada := b.mem.ObtenerLista(nombreLista)
		if !encontrada {
			return fmt.Sprintf("No encontré una lista llamada '%s', señor.", nombreLista), true
		}
		fmt.Println(lista)
		return fmt.Sprintf("Lista '%s', señor. Mire la consola.", nombreLista), true
	}

	if nombreLista, ok := primerPrefijoQueCoincide(entrada, prefijosEliminarLista); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		if err := b.mem.EliminarLista(nombreLista); err != nil {
			return fmt.Sprintf("No pude eliminar la lista: %v", err), true
		}
		return fmt.Sprintf("Lista '%s' eliminada, señor.", nombreLista), true
	}

	if nombreLista, ok := primerPrefijoQueCoincide(entrada, prefijosCrearLista); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		if err := b.mem.CrearLista(nombreLista); err != nil {
			return fmt.Sprintf("No pude crear la lista: %v", err), true
		}
		return fmt.Sprintf("Lista '%s' creada, señor.", nombreLista), true
	}

	if itemCompleto, ok := detectarAgregarALista(entrada); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		nombreLista, item := itemCompleto.nombre, itemCompleto.item
		if err := b.mem.AgregarItemLista(nombreLista, item); err != nil {
			return fmt.Sprintf("No pude agregar a la lista: %v", err), true
		}
		return fmt.Sprintf("Agregué '%s' a la lista '%s', señor.", item, nombreLista), true
	}

	if itemCompleto, ok := detectarMarcarHecho(entrada); ok {
		if b.mem == nil {
			return "No tengo memoria persistente configurada, señor.", true
		}
		nombreLista, item := itemCompleto.nombre, itemCompleto.item
		itemReal, err := b.mem.MarcarItemLista(nombreLista, item)
		if err != nil {
			return fmt.Sprintf("No pude marcar el item: %v", err), true
		}
		return fmt.Sprintf("'%s' marcado como hecho en la lista '%s', señor.", itemReal, nombreLista), true
	}

	return "", false
}

type itemCompleto struct {
	nombre string
	item   string
}

func detectarAgregarALista(entrada string) (*itemCompleto, bool) {
	lower := strings.ToLower(entrada)
	for _, p := range prefijosAgregarLista {
		if strings.HasPrefix(lower, p) {
			resto := strings.TrimSpace(entrada[len(p):])
			idx := strings.LastIndex(resto, " a la lista ")
			if idx < 0 {
				idx = strings.LastIndex(resto, " a lista ")
			}
			if idx < 0 {
				idx = strings.LastIndex(resto, " en la lista ")
			}
			if idx < 0 {
				return nil, false
			}
			item := strings.TrimSpace(resto[:idx])
			nombreLista := strings.TrimSpace(resto[idx+len(" a la lista "):])
			if strings.HasPrefix(nombreLista, "de ") {
				nombreLista = strings.TrimPrefix(nombreLista, "de ")
			}
			if item == "" || nombreLista == "" {
				return nil, false
			}
			return &itemCompleto{nombre: nombreLista, item: item}, true
		}
	}
	return nil, false
}

func detectarMarcarHecho(entrada string) (*itemCompleto, bool) {
	lower := strings.ToLower(entrada)
	for _, p := range prefijosMarcarHecho {
		if strings.HasPrefix(lower, p) {
			resto := strings.TrimSpace(entrada[len(p):])
			idx := strings.LastIndex(resto, " en la lista ")
			if idx < 0 {
				idx = strings.LastIndex(resto, " de la lista ")
			}
			if idx < 0 {
				idx = strings.LastIndex(resto, " en lista ")
			}
			if idx < 0 {
				return nil, false
			}
			item := strings.TrimSpace(resto[:idx])
			nombreLista := strings.TrimSpace(resto[idx+len(" en la lista "):])
			if strings.HasPrefix(nombreLista, "de ") {
				nombreLista = strings.TrimPrefix(nombreLista, "de ")
			}
			if item == "" || nombreLista == "" {
				return nil, false
			}
			return &itemCompleto{nombre: nombreLista, item: item}, true
		}
	}
	return nil, false
}

func (b *Brain) responderHecho(clave, formatoConValor, mensajeSinValor string) (string, bool) {
	if b.mem == nil {
		return "No tengo memoria persistente configurada, señor.", true
	}
	if valor, existe := b.mem.ObtenerHecho(clave); existe {
		return fmt.Sprintf(formatoConValor, valor), true
	}
	return mensajeSinValor, true
}

func extraerContenidoRecordar(original string) (contenido string, ok bool) {
	lower := strings.ToLower(original)
	for _, p := range prefijosRecordar {
		if strings.HasPrefix(lower, p) {
			return strings.TrimSpace(original[len(p):]), true
		}
	}
	return "", false
}

func extraerHechoDirecto(original string) (clave, valor string, esHecho bool) {
	lower := strings.ToLower(original)
	allPatterns := []struct {
		clave   string
		patrons []string
	}{
		{"nombre", patronesNombre},
		{"ciudad", patronesCiudad},
		{"cumpleaños", patronesCumpleanos},
		{"trabajo", patronesTrabajo},
	}
	for _, group := range allPatterns {
		if v, ok := prefijoCoincideConOriginal(lower, original, group.patrons); ok {
			return group.clave, v, true
		}
	}
	return "", "", false
}

func prefijoCoincideConOriginal(lower, original string, prefijos []string) (resto string, ok bool) {
	for _, p := range prefijos {
		if strings.HasPrefix(lower, p) {
			return strings.TrimSpace(original[len(p):]), true
		}
	}
	return "", false
}

func extraerHecho(contenido string) (clave, valor string, esHecho bool) {
	if v, ok := primerPrefijoQueCoincide(contenido, patronesNombre); ok {
		return "nombre", v, true
	}
	if v, ok := primerPrefijoQueCoincide(contenido, patronesCiudad); ok {
		return "ciudad", v, true
	}
	if v, ok := primerPrefijoQueCoincide(contenido, patronesCumpleanos); ok {
		return "cumpleaños", v, true
	}
	if v, ok := primerPrefijoQueCoincide(contenido, patronesTrabajo); ok {
		return "trabajo", v, true
	}
	return "", "", false
}

func primerPrefijoQueCoincide(contenido string, prefijos []string) (resto string, ok bool) {
	contenidoLower := strings.ToLower(contenido)
	for _, p := range prefijos {
		if strings.HasPrefix(contenidoLower, p) {
			return strings.TrimSpace(contenido[len(p):]), true
		}
	}
	return "", false
}

func contieneAlguna(entrada string, frases []string) bool {
	entradaLower := strings.ToLower(entrada)
	for _, f := range frases {
		if strings.Contains(entradaLower, f) {
			return true
		}
	}
	return false
}
