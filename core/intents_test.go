package core

import "testing"

func TestClasificarArmas(t *testing.T) {
	casos := []struct {
		entrada string
		esperado string
	}{
		{"hacé ping a google", "ping_sitio"},
		{"cuál es mi velocidad de internet", "velocidad_internet"},
		{"escanear la red local", "escanear_red"},
		{"limpiar la caché dns", "limpiar_dns"},
		{"mostrame los adaptadores de red", "info_red"},
		{"cuánta ram me queda", "uso_ram"},
		{"cuál es mi ip publica", "ip_publica"},
		{"cuales son mis dns", ""},
		{"red wifi", ""},
		{"limpiar archivos temporales", "limpiar_temporales"},
		{"organizá mis descargas", "organizar_descargas"},
		{"grabá la pantalla", "grabar_pantalla"},
		{"poné el modo oscuro", "modo_oscuro"},
		{"está activo el firewall", "firewall"},
		{"qué puertos están abiertos", "puertos_uso"},
		{"aplicaciones usando internet", "procesos_red"},
		{"sesiones abiertas", "sesiones_activas"},
		{"crear rutina trabajo", "rutina"},
		{"abrir descargas", ""},
		{"avisame que la comida está lista", "notificacion"},
		{"comprimí la carpeta descargas", "comprimir"},
		{"descomprimí el archivo backup", "descomprimir"},
		{"expulsá el usb", "expulsar_disco"},
		{"mantené la pc despierta", "mantener_despierto"},
		{"activar suspensión", "activar_suspension"},
		{"probá el sonido", "probar_sonido"},
		{"qué dispositivos de audio tengo", "listar_audio"},
		{"listar cámaras", "listar_camaras"},
		{"informe del sistema", "informe_sistema"},
		{"qué hay en el portapapeles", "ver_portapapeles"},
		{"crear carpeta informes", "crear_carpeta"},
		{"nueva carpeta de fotos", "crear_carpeta"},
		{"crear archivo listado", "crear_archivo"},
		{"buscame el archivo receta", "buscar_archivo"},
		{"abrir ubicación de mi documento", "abrir_ubicacion"},
		{"borrar archivo basura", "borrar_archivo"},
		{"tomá nota comprar leche", "tomar_nota"},
		{"leé mis notas", "leer_notas"},
		{"hablá sobre python", ""},
		{"carpeta", ""},
		{"diagnóstico completo", "diagnostico"},
		{"revisá mi sistema", "diagnostico"},
		{"salud de mi pc", "salud_sistema"},
		{"qué problemas tiene mi pc", "problemas_sistema"},
		{"limpiá mi pc", "mantenimiento"},
		{"verificar integridad", "integridad"},
		{"qué servicios fallaron", "servicios_caidos"},
		{"qué procesos consumen más", "top_procesos"},
		{"eventos de error recientes", "eventos_error"},
		{"programas de inicio", "programas_inicio"},
		{"qué ocupa espacio", "carpetas_grandes"},
		{"modo vigilante", "vigilante_on"},
		{"pará la vigilancia", "vigilante_off"},
		{"estás vigilando", "vigilante_estado"},
		{"plan de acción", "plan_accion"},
		{"generá un plan", "plan_accion"},
		{"qué debería hacer", "plan_accion"},
		{"ejecutá el plan", "ejecutar_plan"},
		{"aplicá el plan", "ejecutar_plan"},
		{"qué skills tenés", "listar_skills"},
		{"mostrame las skills", "listar_skills"},
		{"carpeta descargas", ""},
		{"cancelá la orden #1", "orden"},
		{"cancela la orden #2", "orden"},
		{"bloqueá la orden #3", "orden"},
		{"terminá la orden #4", "orden"},
		{"reportá la orden #5", "orden"},
		{"enviar un email a juan@gmail.com", "email"},
		{"enviá un correo a ana@empresa.com", "email"},
		{"mandar un mail de resumen", "email"},
		{"leer mis correos", "email"},
		{"revisar mi email", "email"},
		{"cuántos correos tengo", "email"},
		{"crear un documento word", "crear_documento_office"},
		{"creá un documento de word", "crear_documento_office"},
		{"crear un word", "crear_documento_office"},
		{"crear una planilla de excel", "crear_documento_office"},
		{"crear un excel", "crear_documento_office"},
		{"crear una hoja de calculo", "crear_documento_office"},
		{"crear una presentación de powerpoint", "crear_documento_office"},
		{"hacé un powerpoint", "crear_documento_office"},
		{"crear diapositivas", "crear_documento_office"},
		{"abrir word", ""},
		{"documento", ""},
		{"agendá una cita con el médico mañana a las 9", "agenda"},
		{"agenda la reunión del lunes a las 10", "agenda"},
		{"anotá en el calendario el cumpleaños de mamá el martes", "agenda"},
		{"agendame el dentista para el viernes", "agenda"},
		{"qué tengo hoy", "agenda"},
		{"qué tengo mañana", "agenda"},
		{"próximos eventos", "agenda"},
		{"cancelá el evento gimnasio", "agenda"},
		{"abrir calendario", ""},
		{"calendario", "agenda"},
		{"publicá en x que estoy trabajando", "redes"},
		{"publicá en linkedin un aviso de la promo", "redes"},
		{"twitteá un tuit para la gente", "redes"},
		{"publicá un post en linkedin", "redes"},
		{"publicar un tuit", "redes"},
		{"hacé un post en linkedin", "redes"},
		{"cuál es mi ip publica", "ip_publica"},
		{"decime la ip publica", "ip_publica"},
	}
	c := NuevoClasificador()
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}

func TestClasificarVariacionesNaturales(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"súbele al volumen", "volumen_subir"},
		{"subi el volumen", "volumen_subir"},
		{"bajale el volumen", "volumen_bajar"},
		{"bajame el volumen", "volumen_bajar"},
		{"dame menos volumen", "volumen_bajar"},
		{"súbele", "volumen_subir"},
		{"mas alto", "volumen_subir"},
		{"subilo", "volumen_subir"},
		{"bájale", "volumen_bajar"},
		{"mas bajo", "volumen_bajar"},
		{"continuá la musica", "play_pause"},
		{"reproducí la playlist", "play_pause"},
		{"pasa a la siguiente", "siguiente_cancion"},
		{"volve para atras", "cancion_anterior"},
		{"me decis la hora", "hora"},
		{"tenes hora", "hora"},
		{"que tiempo hace", "clima"},
		{"cuantos grados hace", "clima"},
		{"sube el brillo", ""},
		{"baja el brillo", ""},
		{"subi el brillo", ""},
		{"bajame el brillo", ""},
		{"subele el brillo", ""},
		{"bajele el brillo", ""},
		{"la hora del partido", ""},
		{"cuanto falta para las 8", ""},
		{"el pdf del contrato", ""},
	}
	c := NuevoClasificador()
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}

func TestClasificarHora(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"qué hora es", "hora"},
		{"decime la hora", "hora"},
		{"hora", "hora"},
		{"hora actual", "hora"},
		{"ahora mismo abrí chrome", ""},
		{"qué hora era cuando terminó", ""},
		{"cuántas horas ahorré", "informe_piloto"},
	}
	c := NuevoClasificador()
	for _, cso := range casos {
		nombre, ok := c.Clasificar(cso.entrada)
		if cso.esperado == "" {
			if ok {
				t.Errorf("entrada %q: esperado sin match, obtuve %q", cso.entrada, nombre)
			}
			continue
		}
		if !ok || nombre != cso.esperado {
			t.Errorf("entrada %q: esperado %q, obtuve ok=%v nombre=%q", cso.entrada, cso.esperado, ok, nombre)
		}
	}
}
