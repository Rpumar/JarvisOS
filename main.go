package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"JarvisOS/agents"
	"JarvisOS/config"
	"JarvisOS/core"
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

	prefs := memoria.NuevoGestorPreferencias(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "preferencias.json"))
	rutinas := core.NuevoRutinaManager(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "rutinas.json"))

	hands := core.NewHands(core.HandsOpciones{
		Apps:     cfg.Apps,
		ClimaKey: cfg.OpenWeatherKey,
		NewsKey:  cfg.NewsAPIKey,
		Prefs:    prefs,
		Rutinas:  rutinas,
	})
	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout)
	coderAgent := agents.NewCoderAgent(conectorIA)
	gestorPlan := agents.NuevoGestorPlan(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "planes"))
	ingAgente := agents.NuevoAgenteProyecto(conectorIA, cfg.WorkspaceRoot, gestorPlan)

	almacen, err := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR FATAL] No se pudo inicializar la memoria persistente: %v\n", err)
		os.Exit(1)
	}
	defer almacen.Cerrar()

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
		Coder:          coderAgent,
		Memoria:        almacen,
		IngAgente:      ingAgente,
		Prefs:          prefs,
		MaxHistorialIA: cfg.MaxHistorialIA,
	})

	if plan := gestorPlan.PlanPendiente(); plan != nil {
		fmt.Printf("[PLAN] Tiene un plan pendiente: %s\n", plan.Objetivo)
		fmt.Println("[PLAN] Diga 'continuar plan' para retomarlo o 'cancelar plan' para descartarlo.")
	}

	if conectorIA.Disponible() {
		fmt.Println("[IA] Ollama activo (Llama 3.2 3B).")
	} else {
		fmt.Println("[IA] Sin Ollama. Solo comandos locales.")
	}
	fmt.Printf("[MEMORIA] Datos persistentes en: %s\n", cfg.RutaMemoria)
	fmt.Printf("[CONFIG] Escucha continua: %v | Palabras de activación: %v\n", cfg.ContinuousListening, cfg.WakeWords)

	oidos, err := core.NewEars(cfg.ModeloVoz)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR FATAL] No se pudo inicializar el micrófono: %v\n", err)
		fmt.Fprintf(os.Stderr, "Verifica que exista el modelo de voz en '%s' y que haya un micrófono conectado.\n", cfg.ModeloVoz)
		os.Exit(1)
	}
	defer oidos.Cerrar()

	fmt.Println("[JARVIS] Sistemas en línea.")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vigilarRecordatorios(almacen, hands)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	escuchaActiva := cfg.ContinuousListening

	apagar := make(chan struct{})
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\n[APAGANDO] Señal recibida. Cerrando sistemas...")
			close(apagar)
		case <-apagar:
		}
	}()

loop:
	for {
		select {
		case <-apagar:
			break loop
		default:
		}

		if escuchaActiva {
			fmt.Print("\n[VOZ] Diga su comando... ")
			comando, err := oidos.Escuchar()
			if err != nil {
				fmt.Printf("[ADVERTENCIA] %v\n", err)
			}
			escuchaActiva = false
			if comando == "" {
				continue
			}
			comandoLower := strings.ToLower(comando)
			if esPalabraDeActivacion(comandoLower, cfg.WakeWords) {
				hands.Hablar(brain.Saludar())
				escuchaActiva = true
				continue
			}
			respuesta := brain.Process(comando)
			if respuesta != "" {
				fmt.Printf("[JARVIS] %s\n", respuesta)
				hands.Hablar(respuesta)
			}
			continue
		}

		fmt.Print("\n> ")
		comando, err := oidos.EscucharTexto()
		if err != nil {
			fmt.Printf("[ADVERTENCIA] %v\n", err)
			continue
		}

		if comando == "" {
			continue
		}

		comandoLower := strings.ToLower(comando)

		if esPalabraDeActivacion(comandoLower, cfg.WakeWords) {
			escuchaActiva = true
			hands.Hablar(brain.Saludar())
			continue
		}

		if strings.Contains(comandoLower, "apagar") || strings.Contains(comandoLower, "adiós") {
			hands.Hablar(brain.Despedirse())
			break loop
		}

		respuesta := brain.Process(comando)
		if respuesta != "" {
			fmt.Printf("[JARVIS] %s\n", respuesta)
			hands.Hablar(respuesta)
		}
	}

	fmt.Println("[APAGANDO] Liberando memoria...")
	close(apagar)
	wg.Wait()
	fmt.Println("[APAGANDO] Sistemas fuera. Hasta luego, señor.")
}

func ejecutarModoServicio() {
	_ = os.Stdin.Close()
	logF, err := os.OpenFile(
		filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "service.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600,
	)
	if err == nil {
		os.Stdout = logF
		os.Stderr = logF
		defer logF.Close()
	}

	cfg := config.Load()
	hands := core.NewHands(core.HandsOpciones{
		Apps:     cfg.Apps,
		ClimaKey: cfg.OpenWeatherKey,
		NewsKey:  cfg.NewsAPIKey,
	})
	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout)
	coderAgent := agents.NewCoderAgent(conectorIA)
	gestorPlan := agents.NuevoGestorPlan(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "planes"))
	ingAgente := agents.NuevoAgenteProyecto(conectorIA, cfg.WorkspaceRoot, gestorPlan)
	almacen, err := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SERVICE] Error fatal: %v\n", err)
		os.Exit(1)
	}
	defer almacen.Cerrar()
	brain := core.NewBrain(hands, core.BrainOpciones{
		IA: conectorIA, Coder: coderAgent, Memoria: almacen, IngAgente: ingAgente,
		MaxHistorialIA: cfg.MaxHistorialIA,
	})
	oidos, err := core.NewEars(cfg.ModeloVoz)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SERVICE] Error voz: %v\n", err)
		os.Exit(1)
	}
	defer oidos.Cerrar()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vigilarRecordatoriosService(almacen, hands)
	}()

	fmt.Println("[SERVICE] JarvisOS iniciado en modo servicio.")

	for {
		comando, err := oidos.EscucharTexto()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if comando == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		comandoLower := strings.ToLower(comando)
		if esPalabraDeActivacion(comandoLower, cfg.WakeWords) {
			escuchaActiva := true
			for escuchaActiva {
				comando, err := oidos.Escuchar()
				if err != nil || comando == "" {
					escuchaActiva = false
					continue
				}
				respuesta := brain.Process(comando)
				if respuesta != "" {
					fmt.Printf("[SERVICE] %s\n", respuesta)
					hands.Hablar(respuesta)
				}
				escuchaActiva = false
			}
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
			hands.Hablar(fmt.Sprintf("Recordatorio, señor: %s", r.Texto))
			if err := almacen.MarcarCumplido(r.ID); err != nil {
				fmt.Printf("[SERVICE] Error marcando recordatorio: %v\n", err)
			}
		}
	}
}

func esPalabraDeActivacion(comandoLower string, wakeWords []string) bool {
	for _, w := range wakeWords {
		if strings.Contains(comandoLower, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

func ejecutarWebUI() {
	cfg := config.Load()
	prefs := memoria.NuevoGestorPreferencias(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "preferencias.json"))
	rutinas := core.NuevoRutinaManager(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "rutinas.json"))
	hands := core.NewHands(core.HandsOpciones{
		Apps: cfg.Apps, ClimaKey: cfg.OpenWeatherKey, NewsKey: cfg.NewsAPIKey,
		Prefs: prefs, Rutinas: rutinas,
	})
	conectorIA := ia.NuevoConector(cfg.ModeloIA, cfg.Timeout)
	coderAgent := agents.NewCoderAgent(conectorIA)
	gestorPlan := agents.NuevoGestorPlan(filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "planes"))
	ingAgente := agents.NuevoAgenteProyecto(conectorIA, cfg.WorkspaceRoot, gestorPlan)
	almacen, _ := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if almacen != nil {
		defer almacen.Cerrar()
	}
	brain := core.NewBrain(hands, core.BrainOpciones{
		IA: conectorIA, Coder: coderAgent, Memoria: almacen,
		IngAgente: ingAgente, Prefs: prefs, MaxHistorialIA: cfg.MaxHistorialIA,
	})
	servidor := webui.NuevoServidor(brain, 8080, webui.ServidorOpciones{
		Estado:        hands,
		RutaHistorial: filepath.Join(os.Getenv("USERPROFILE"), "JarvisOS-datos", "historial-web.json"),
	})
	if err := servidor.Iniciar(); err != nil {
		fmt.Fprintf(os.Stderr, "[WEBUI] Error: %v\n", err)
		os.Exit(1)
	}
}

func vigilarRecordatorios(almacen *memoria.Almacen, hands *core.Hands) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[ADVERTENCIA] El vigía de recordatorios se detuvo por un error inesperado: %v\n", r)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pendientes := almacen.RecordatoriosPendientes(time.Now())
		for _, r := range pendientes {
			hands.Hablar(fmt.Sprintf("Recordatorio, señor: %s", r.Texto))
			if err := almacen.MarcarCumplido(r.ID); err != nil {
				fmt.Printf("[ADVERTENCIA] No pude marcar el recordatorio como cumplido: %v\n", err)
			}
		}
	}
}
