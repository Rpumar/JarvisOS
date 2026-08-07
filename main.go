package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"JarvisOS/config"
	"JarvisOS/core"
	"JarvisOS/core/audit"
	"JarvisOS/ia"
	"JarvisOS/memoria"
	"JarvisOS/webui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--install":
			if err := core.InstalarServicio(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--uninstall":
			if err := core.DesinstalarServicio(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--start":
			if err := core.IniciarServicio(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--stop":
			if err := core.DetenerServicio(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--status":
			fmt.Println(core.EstadoServicio())
			return
		case "--service":
			ejecutarModoServicio()
			return
		case "--web":
			ejecutarWebUI()
			return
		}
	}

	cfg := config.Load()

	fmt.Println("=================================")
	fmt.Printf(" %s v%s ACTIVADO\n", cfg.AppName, cfg.Version)
	fmt.Println(" El que maneja el total de la PC")
	fmt.Println("=================================")
	fmt.Printf("[LICENCIA] %s\n", core.EstadoLicencia(cfg.LicenseKey))

	if ruta, err := core.RealizarBackup(config.DatosDir(), core.BackupsMax); err != nil {
		fmt.Printf("[BACKUP] No pude respaldar los datos: %v\n", err)
	} else {
		fmt.Printf("[BACKUP] Respaldo creado: %s\n", ruta)
	}

	prefs := memoria.NuevoGestorPreferencias(filepath.Join(config.DatosDir(), "preferencias.json"))
	rutinas := core.NuevoRutinaManager(filepath.Join(config.DatosDir(), "rutinas.json"))
	tareas := core.NuevoGestorTareas(filepath.Join(config.DatosDir(), "tareas.json"))
	agenda := core.NuevoGestorAgenda(filepath.Join(config.DatosDir(), "agenda.json"))
	ordenes := core.NuevoGestorOrdenes(filepath.Join(config.DatosDir(), "ordenes.json"))
	procedimientos := core.NuevoGestorProcedimientos(filepath.Join(config.DatosDir(), "procedimientos.json"))
	formularios := core.NuevoGestorFormularios(filepath.Join(config.DatosDir(), "formularios.json"))
	auditoria := audit.NuevoRegistro(filepath.Join(config.DatosDir(), "auditoria.jsonl"))

	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout, cfg.IAURL, cfg.IAAPIKey)
	gestorSkills := core.NuevoSkillsManager()
	gestorRoles := core.NuevoRolesManager()
	gestorEmpresa := core.NuevoGestorEmpresa(filepath.Join(config.DatosDir(), "empresa.json"))
	gestorPerfil := core.NuevoGestorPerfil(filepath.Join(config.DatosDir(), "perfil.json"))
	gestorPerfil.LimitePuestos = core.PuestosLicencia(cfg.LicenseKey)
	hands := core.NewHands(core.HandsOpciones{
		Apps:             cfg.Apps,
		ClimaKey:         cfg.OpenWeatherKey,
		NewsKey:          cfg.NewsAPIKey,
		Prefs:            prefs,
		Rutinas:          rutinas,
		Tareas:           tareas,
		Agenda:           agenda,
		Ordenes:          ordenes,
		Procedimientos:   procedimientos,
		Formularios:      formularios,
		WorkspaceRoot:    cfg.WorkspaceRoot,
		DatosDir:         config.DatosDir(),
		IA:               conectorIA,
		Skills:           gestorSkills,
		Auditoria:        auditoria,
		Perfil:           gestorPerfil,
		PINHash:          cfg.PINHash,
		PINSetter:        func(hash string) bool { cfg.PINHash = hash; return cfg.Save() == nil },
		ContrasenaHash:   cfg.LoginPasswordHash,
		ContrasenaSetter: func(hash string) bool { cfg.LoginPasswordHash = hash; return cfg.Save() == nil },
		LicenseKey:       cfg.LicenseKey,
		LicenseSetter:    func(clave string) bool { cfg.LicenseKey = clave; return cfg.Save() == nil },
		EmailEnabled:     cfg.EmailEnabled,
		EmailSmtpHost:    cfg.EmailSmtpHost,
		EmailSmtpPort:    cfg.EmailSmtpPort,
		EmailUsuario:     cfg.EmailUsuario,
		EmailPassword:    cfg.EmailPassword,
		EmailDesde:       cfg.EmailDesde,
		EmailImapHost:    cfg.EmailImapHost,
		EmailImapPort:    cfg.EmailImapPort,
		EmailImapMax:     cfg.EmailImapMax,
		XApiKey:          cfg.XApiKey,
		XApiSecret:       cfg.XApiSecret,
		XAccessToken:     cfg.XAccessToken,
		XAccessSecret:    cfg.XAccessSecret,
		LinkedInToken:    cfg.LinkedInToken,
		LinkedInAuthor:   cfg.LinkedInAuthor,
		LimiteComando:    time.Duration(cfg.ComandoTimeoutSegundos) * time.Second,
	})

	almacen, err := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR FATAL] No se pudo inicializar la memoria persistente: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = almacen.Cerrar() }()

	if cfg.WorkspaceRoot != "" {
		prefs.SetUltimoProyecto(cfg.WorkspaceRoot)
	}
	p := prefs.Get()
	nombre := p.Nombre
	if nombre != "" {
		fmt.Printf("[JARVIS] Bienvenido de nuevo, %s. %s\n", nombre, prefs.String())
	} else {
		fmt.Printf("[JARVIS] %s\n", prefs.String())
	}

	brain := core.NewBrain(hands, core.BrainOpciones{
		IA:             conectorIA,
		Memoria:        almacen,
		Prefs:          prefs,
		Skills:         gestorSkills,
		Roles:          gestorRoles,
		Procedimientos: procedimientos,
		Empresa:        gestorEmpresa,
		Perfil:         gestorPerfil,
		MaxHistorialIA: cfg.MaxHistorialIA,
	})

	if pendientes := tareas.TextoPendientes(); pendientes != "" {
		fmt.Printf("[TAREAS] %s\n", pendientes)
	}

	if pendientesOrdenes := ordenes.TextoPendientes(); pendientesOrdenes != "" {
		fmt.Printf("[ORDENES] %s\n", pendientesOrdenes)
		fmt.Println("[ORDENES] Las órdenes no se abandonan. Las retomaré automáticamente.")
	}

	if conectorIA.Disponible() {
		fmt.Println("[IA] Ollama activo (Llama 3.2 3B).")
	} else {
		fmt.Println("[IA] Sin Ollama. Solo comandos locales.")
	}
	fmt.Printf("[MEMORIA] Datos persistentes en: %s\n", cfg.RutaMemoria)

	oidos, err := core.NewEars("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR FATAL] No se pudo inicializar la entrada de texto: %v\n", err)
		os.Exit(1)
	}
	defer oidos.Cerrar()

	fmt.Println("[JARVIS] Sistemas en línea.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	apagar := make(chan struct{})
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\n[APAGANDO] Señal recibida. Cerrando sistemas...")
			close(apagar)
		case <-apagar:
		}
	}()

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		vigilarRecordatorios(almacen, hands, apagar)
	}()
	go func() {
		defer wg.Done()
		vigilarAprobaciones(hands, apagar)
	}()
	go func() {
		defer wg.Done()
		vigilarOrdenes(hands, apagar)
	}()
	go func() {
		defer wg.Done()
		vigilarInforme(hands, apagar)
	}()

loop:
	for {
		select {
		case <-apagar:
			break loop
		default:
		}

		fmt.Print("\n> ")
		comando, err := oidos.EscucharTexto()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\nFin de la entrada. Hasta luego, señor.")
				return
			}
			fmt.Printf("[ADVERTENCIA] %v\n", err)
			continue
		}

		if comando == "" {
			continue
		}

		comandoLower := strings.ToLower(comando)

		if core.EsDespedida(comandoLower) {
			fmt.Println(brain.Despedirse())
			close(apagar)
			break loop
		}

		respuesta := brain.Process(comando)
		if respuesta != "" {
			fmt.Printf("[JARVIS] %s\n", respuesta)
		}
	}

	fmt.Println("[APAGANDO] Liberando memoria...")
	wg.Wait()
	fmt.Println("[APAGANDO] Sistemas fuera. Hasta luego, señor.")
}

func ejecutarModoServicio() {
	_ = os.Stdin.Close()
	logF, err := os.OpenFile(
		filepath.Join(config.DatosDir(), "service.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600,
	)
	if err == nil {
		os.Stdout = logF
		os.Stderr = logF
		defer logF.Close()
	}

	cfg := config.Load()
	if ruta, err := core.RealizarBackup(config.DatosDir(), core.BackupsMax); err != nil {
		fmt.Fprintf(os.Stderr, "[BACKUP] No pude respaldar los datos: %v\n", err)
	} else {
		fmt.Printf("[BACKUP] Respaldo creado: %s\n", ruta)
	}
	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout, cfg.IAURL, cfg.IAAPIKey)
	gestorSkills := core.NuevoSkillsManager()
	gestorRoles := core.NuevoRolesManager()
	gestorEmpresa := core.NuevoGestorEmpresa(filepath.Join(config.DatosDir(), "empresa.json"))
	gestorPerfil := core.NuevoGestorPerfil(filepath.Join(config.DatosDir(), "perfil.json"))
	gestorPerfil.LimitePuestos = core.PuestosLicencia(cfg.LicenseKey)
	tareas := core.NuevoGestorTareas(filepath.Join(config.DatosDir(), "tareas.json"))
	agenda := core.NuevoGestorAgenda(filepath.Join(config.DatosDir(), "agenda.json"))
	ordenes := core.NuevoGestorOrdenes(filepath.Join(config.DatosDir(), "ordenes.json"))
	procedimientos := core.NuevoGestorProcedimientos(filepath.Join(config.DatosDir(), "procedimientos.json"))
	formularios := core.NuevoGestorFormularios(filepath.Join(config.DatosDir(), "formularios.json"))
	auditoria := audit.NuevoRegistro(filepath.Join(config.DatosDir(), "auditoria.jsonl"))
	hands := core.NewHands(core.HandsOpciones{
		Apps:             cfg.Apps,
		ClimaKey:         cfg.OpenWeatherKey,
		NewsKey:          cfg.NewsAPIKey,
		WorkspaceRoot:    cfg.WorkspaceRoot,
		DatosDir:         config.DatosDir(),
		IA:               conectorIA,
		Skills:           gestorSkills,
		Tareas:           tareas,
		Agenda:           agenda,
		Ordenes:          ordenes,
		Procedimientos:   procedimientos,
		Formularios:      formularios,
		Auditoria:        auditoria,
		Perfil:           gestorPerfil,
		PINHash:          cfg.PINHash,
		PINSetter:        func(hash string) bool { cfg.PINHash = hash; return cfg.Save() == nil },
		ContrasenaHash:   cfg.LoginPasswordHash,
		ContrasenaSetter: func(hash string) bool { cfg.LoginPasswordHash = hash; return cfg.Save() == nil },
		LicenseKey:       cfg.LicenseKey,
		LicenseSetter:    func(clave string) bool { cfg.LicenseKey = clave; return cfg.Save() == nil },
		EmailEnabled:     cfg.EmailEnabled,
		EmailSmtpHost:    cfg.EmailSmtpHost,
		EmailSmtpPort:    cfg.EmailSmtpPort,
		EmailUsuario:     cfg.EmailUsuario,
		EmailPassword:    cfg.EmailPassword,
		EmailDesde:       cfg.EmailDesde,
		EmailImapHost:    cfg.EmailImapHost,
		EmailImapPort:    cfg.EmailImapPort,
		EmailImapMax:     cfg.EmailImapMax,
		XApiKey:          cfg.XApiKey,
		XApiSecret:       cfg.XApiSecret,
		XAccessToken:     cfg.XAccessToken,
		XAccessSecret:    cfg.XAccessSecret,
		LinkedInToken:    cfg.LinkedInToken,
		LinkedInAuthor:   cfg.LinkedInAuthor,
		LimiteComando:    time.Duration(cfg.ComandoTimeoutSegundos) * time.Second,
	})
	almacen, err := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SERVICE] Error fatal: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = almacen.Cerrar() }()
	brain := core.NewBrain(hands, core.BrainOpciones{
		IA: conectorIA, Memoria: almacen,
		Skills: gestorSkills, Roles: gestorRoles, Procedimientos: procedimientos, Empresa: gestorEmpresa, Perfil: gestorPerfil, MaxHistorialIA: cfg.MaxHistorialIA,
	})
	oidos, err := core.NewEars("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SERVICE] Error entrada de texto: %v\n", err)
		os.Exit(1)
	}
	defer oidos.Cerrar()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vigilarRecordatoriosService(almacen, hands)
	}()
	go vigilarAprobaciones(hands, nil)
	go vigilarOrdenes(hands, nil)
	go vigilarInforme(hands, nil)

	fmt.Println("[SERVICE] JarvisOS iniciado en modo servicio.")

	for {
		comando, err := oidos.EscucharTexto()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("[SERVICE] Entrada cerrada. Deteniendo modo servicio.")
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if comando == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		respuesta := brain.Process(comando)
		if respuesta != "" {
			fmt.Printf("[SERVICE] %s\n", respuesta)
		}
	}
}

func vigilarRecordatoriosService(almacen *memoria.Almacen, hands *core.Hands) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[SERVICE] Vigía de recordatorios detenido: %v\n", r)
		}
	}()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		pendientes := almacen.RecordatoriosPendientes(time.Now())
		for _, r := range pendientes {
			fmt.Printf("[SERVICE] Recordatorio, señor: %s\n", r.Texto)
			if err := almacen.MarcarCumplido(r.ID); err != nil {
				fmt.Printf("[SERVICE] Error marcando recordatorio: %v\n", err)
			}
		}
	}
}

func ejecutarWebUI() {
	cfg := config.Load()
	prefs := memoria.NuevoGestorPreferencias(filepath.Join(config.DatosDir(), "preferencias.json"))
	rutinas := core.NuevoRutinaManager(filepath.Join(config.DatosDir(), "rutinas.json"))
	tareas := core.NuevoGestorTareas(filepath.Join(config.DatosDir(), "tareas.json"))
	agenda := core.NuevoGestorAgenda(filepath.Join(config.DatosDir(), "agenda.json"))
	ordenes := core.NuevoGestorOrdenes(filepath.Join(config.DatosDir(), "ordenes.json"))
	procedimientos := core.NuevoGestorProcedimientos(filepath.Join(config.DatosDir(), "procedimientos.json"))
	formularios := core.NuevoGestorFormularios(filepath.Join(config.DatosDir(), "formularios.json"))
	auditoria := audit.NuevoRegistro(filepath.Join(config.DatosDir(), "auditoria.jsonl"))
	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout, cfg.IAURL, cfg.IAAPIKey)
	gestorSkills := core.NuevoSkillsManager()
	gestorRoles := core.NuevoRolesManager()
	gestorEmpresa := core.NuevoGestorEmpresa(filepath.Join(config.DatosDir(), "empresa.json"))
	gestorPerfil := core.NuevoGestorPerfil(filepath.Join(config.DatosDir(), "perfil.json"))
	gestorPerfil.LimitePuestos = core.PuestosLicencia(cfg.LicenseKey)
	hands := core.NewHands(core.HandsOpciones{
		Apps: cfg.Apps, ClimaKey: cfg.OpenWeatherKey, NewsKey: cfg.NewsAPIKey,
		Prefs: prefs, Rutinas: rutinas, Tareas: tareas, Agenda: agenda, Ordenes: ordenes, Procedimientos: procedimientos, Formularios: formularios,
		WorkspaceRoot: cfg.WorkspaceRoot, IA: conectorIA, Skills: gestorSkills, DatosDir: config.DatosDir(),
		Auditoria: auditoria, Perfil: gestorPerfil, PINHash: cfg.PINHash,
		PINSetter:        func(hash string) bool { cfg.PINHash = hash; return cfg.Save() == nil },
		ContrasenaHash:   cfg.LoginPasswordHash,
		ContrasenaSetter: func(hash string) bool { cfg.LoginPasswordHash = hash; return cfg.Save() == nil },
		LicenseKey:       cfg.LicenseKey,
		LicenseSetter:    func(clave string) bool { cfg.LicenseKey = clave; return cfg.Save() == nil },
		EmailEnabled:     cfg.EmailEnabled,
		EmailSmtpHost:    cfg.EmailSmtpHost,
		EmailSmtpPort:    cfg.EmailSmtpPort,
		EmailUsuario:     cfg.EmailUsuario,
		EmailPassword:    cfg.EmailPassword,
		EmailDesde:       cfg.EmailDesde,
		EmailImapHost:    cfg.EmailImapHost,
		EmailImapPort:    cfg.EmailImapPort,
		EmailImapMax:     cfg.EmailImapMax,
		XApiKey:          cfg.XApiKey,
		XApiSecret:       cfg.XApiSecret,
		XAccessToken:     cfg.XAccessToken,
		XAccessSecret:    cfg.XAccessSecret,
		LinkedInToken:    cfg.LinkedInToken,
		LinkedInAuthor:   cfg.LinkedInAuthor,
		LimiteComando:    time.Duration(cfg.ComandoTimeoutSegundos) * time.Second,
	})
	almacen, _ := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if almacen != nil {
		defer func() { _ = almacen.Cerrar() }()
	}
	brain := core.NewBrain(hands, core.BrainOpciones{
		IA: conectorIA, Memoria: almacen,
		Prefs: prefs, Skills: gestorSkills, Roles: gestorRoles,
		Procedimientos: procedimientos, Empresa: gestorEmpresa, Perfil: gestorPerfil, MaxHistorialIA: cfg.MaxHistorialIA,
	})
	if pendientes := tareas.TextoPendientes(); pendientes != "" {
		fmt.Printf("[TAREAS] %s\n", pendientes)
	}
	if pendientesOrdenes := ordenes.TextoPendientes(); pendientesOrdenes != "" {
		fmt.Printf("[ORDENES] %s\n", pendientesOrdenes)
		fmt.Println("[ORDENES] Las órdenes no se abandonan. Las retomaré automáticamente.")
	}
	go vigilarAprobaciones(hands, nil)
	go vigilarOrdenes(hands, nil)
	go vigilarInforme(hands, nil)
	servidor := webui.NuevoServidor(brain, 8080, webui.ServidorOpciones{
		Estado:           hands,
		Diagnostico:      hands,
		Aprobador:        hands,
		Auditor:          hands,
		Skills:           gestorSkills,
		Roles:            gestorRoles,
		Empresa:          gestorEmpresa,
		Perfil:           gestorPerfil,
		ContrasenaHash:   cfg.LoginPasswordHash,
		PINHash:          cfg.PINHash,
		PINSetter:        func(hash string) bool { cfg.PINHash = hash; return cfg.Save() == nil },
		ContrasenaSetter: func(hash string) bool { cfg.LoginPasswordHash = hash; return cfg.Save() == nil },
		RutaHistorial:    filepath.Join(config.DatosDir(), "historial-web.json"),
	})
	if err := servidor.Iniciar(); err != nil {
		fmt.Fprintf(os.Stderr, "[WEBUI] Error: %v\n", err)
		os.Exit(1)
	}
}

func vigilarAprobaciones(hands *core.Hands, done <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ADVERTENCIA] El vigía de aprobaciones se detuvo por un error inesperado: %v\n", r)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			hands.ExpirarAprobacionesAntiguas(core.TiempoMaximoAprobacion)
		}
	}
}

// vigilarOrdenes es el bucle de recuperación del "empleado que no
// abandona". Cada cierto periodo (y una vez al arrancar) retoma las
// órdenes que siguen pendientes, en progreso o bloqueadas: las cumple
// con los pasos conocidos o con el agente IA, y solo las cierra cuando
// el resultado queda verificado. Las órdenes que esperan aprobación
// del dueño se dejan intactas hasta que él decida.
func vigilarOrdenes(hands *core.Hands, done <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ADVERTENCIA] El vigía de órdenes se detuvo por un error inesperado: %v\n", r)
		}
	}()

	// Al arrancar se retoman las pendientes (principio de F1: una orden
	// sigue viva aunque la PC se haya reiniciado).
	hands.RetomarOrdenes()

	ticker := time.NewTicker(core.IntervaloRetomarOrdenes)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			hands.RetomarOrdenes()
		}
	}
}

func vigilarRecordatorios(almacen *memoria.Almacen, hands *core.Hands, done <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ADVERTENCIA] El vigía de recordatorios se detuvo por un error inesperado: %v\n", r)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			pendientes := almacen.RecordatoriosPendientes(time.Now())
			for _, r := range pendientes {
				fmt.Printf("[RECORDATORIO] %s\n", r.Texto)
				if err := almacen.MarcarCumplido(r.ID); err != nil {
					fmt.Printf("[ADVERTENCIA] No pude marcar el recordatorio como cumplido: %v\n", err)
				}
			}
		}
	}
}

// ultimoInformeLeido guarda la fecha del último informe emitido, para que el
// vigía no lo repita (persiste a disco entre reinicios).
var ultimoInformeMu sync.Mutex
var ultimoInformeFech = ""

// vigilarInforme emite una vez al día, pasadas las HH:00, el informe del día
// que termina: qué órdenes se cumplieron, qué sigue en juego, tareas
// pendientes, actividad auditada y agenda de mañana. Lo guarda en el
// historial de informes para que quede auditable.
func vigilarInforme(hands *core.Hands, done <-chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ADVERTENCIA] El vigía de informe diario se detuvo por un error inesperado: %v\n", r)
		}
	}()

	datosDir := config.DatosDir()
	informesDir := filepath.Join(datosDir, "informes")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	ultimoInformeMu.Lock()
	ultimoInformeFech = leerArchivoInforme(filepath.Join(informesDir, "ultimo-emitido.txt"))
	ultimoInformeMu.Unlock()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ahora := time.Now()
			hoy := ahora.Format("2006-01-02")
			if ahora.Hour() < core.HoraInformeDiario {
				continue
			}
			ultimoInformeMu.Lock()
			if ultimoInformeFech == hoy {
				ultimoInformeMu.Unlock()
				continue
			}
			ultimoInformeMu.Unlock()

			datos := hands.RecolectarInformeDiario(ahora)
			informe := core.GenerarInformeDiario(datos)

			ruta, err := core.GuardarInformeDiario(informesDir, hoy, informe)
			if err != nil {
				fmt.Printf("[INFORME] No pude guardar el informe: %v\n", err)
			} else {
				fmt.Printf("[INFORME] Informe diario emitido -> %s\n", ruta)
			}
			if err := os.WriteFile(filepath.Join(informesDir, "ultimo-emitido.txt"), []byte(hoy), 0o600); err != nil {
				fmt.Printf("[INFORME] No pude registrar la emisión: %v\n", err)
			}
			ultimoInformeMu.Lock()
			ultimoInformeFech = hoy
			ultimoInformeMu.Unlock()
		}
	}
}

func leerArchivoInforme(ruta string) string {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(datos))
}
