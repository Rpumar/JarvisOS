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
		{"no hables", "voz_desactivar"},
		{"activá tu voz", "voz_activar"},
		{"qué voces tenés", "voz_listar"},
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
		{"crear proyecto web panel", "crear_proyecto"},
		{"creá un proyecto web llamado stock", "crear_proyecto"},
		{"hacé un proyecto de finanzas", "crear_proyecto"},
		{"nueva app web para mis clientes", "crear_proyecto"},
		{"scaffold panel de control", "crear_proyecto"},
		{"mis proyectos", "listar_proyectos"},
		{"qué proyectos tengo", "listar_proyectos"},
		{"lista de proyectos", "listar_proyectos"},
		{"compilar el proyecto panel", "compilar_proyecto"},
		{"build del proyecto panel", "compilar_proyecto"},
		{"ejecutar proyecto panel", "ejecutar_proyecto"},
		{"corré el proyecto panel", "ejecutar_proyecto"},
		{"levantá el proyecto stock", "ejecutar_proyecto"},
		{"detener proyecto panel", "detener_proyecto"},
		{"cerrá el proyecto stock", "detener_proyecto"},
		{"pará el proyecto panel", "detener_proyecto"},
		{"estado del proyecto panel", "estado_proyecto"},
		{"está corriendo el proyecto stock", "estado_proyecto"},
		{"mejorar el proyecto panel", "mejorar_proyecto"},
		{"agregá una feature al proyecto panel", "mejorar_proyecto"},
		{"agregar un botón a la app", "mejorar_proyecto"},
		{"qué skills tenés", "listar_skills"},
		{"mostrame las skills", "listar_skills"},
		{"carpeta descargas", ""},
		{"cancelá la orden #1", "orden"},
		{"cancela la orden #2", "orden"},
		{"bloqueá la orden #3", "orden"},
		{"terminá la orden #4", "orden"},
		{"reportá la orden #5", "orden"},
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
