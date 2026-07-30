package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"JarvisOS/agents"
	"JarvisOS/config"
	"JarvisOS/core"
	"JarvisOS/ia"
	"JarvisOS/memoria"
)

func main() {
	cfg := config.Load()

	fmt.Println("=================================")
	fmt.Printf(" %s v%s ACTIVADO\n", cfg.AppName, cfg.Version)
	fmt.Println(" El que maneja el total de la PC")
	fmt.Println("=================================")

	hands := core.NewHands()
	conectorIA := ia.NuevoConector(cfg.Timeout)
	coderAgent := agents.NewCoderAgent(conectorIA)

	almacen, err := memoria.NuevoAlmacen(cfg.RutaMemoria)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR FATAL] No se pudo inicializar la memoria persistente: %v\n", err)
		os.Exit(1)
	}
	defer almacen.Cerrar()

	brain := core.NewBrain(hands, core.BrainOpciones{
		IA:             conectorIA,
		Coder:          coderAgent,
		Memoria:        almacen,
		MaxHistorialIA: cfg.MaxHistorialIA,
	})

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
			fmt.Println("\n[OIDOS] Le escucho...")
		} else {
			fmt.Println("\n[OIDOS] Diga 'Jarvis' para hablarme...")
		}

		comando, err := oidos.Escuchar()
		if err != nil {
			fmt.Printf("[ADVERTENCIA] %v\n", err)
			continue
		}

		if comando == "" {
			continue
		}

		comandoLower := strings.ToLower(comando)

		if strings.Contains(comandoLower, "apagar") && !esPalabraDeActivacion(comandoLower, cfg.WakeWords) {
			hands.Hablar(brain.Despedirse())
			break loop
		}

		if esPalabraDeActivacion(comandoLower, cfg.WakeWords) {
			escuchaActiva = true
			hands.Hablar(brain.Saludar())
			continue
		}

		if escuchaActiva {
			respuesta := brain.Process(comando)
			if respuesta != "" {
				fmt.Printf("[JARVIS] %s\n", respuesta)
				hands.Hablar(respuesta)
			}
		}
	}

	fmt.Println("[APAGANDO] Liberando memoria...")
	close(apagar)
	wg.Wait()
	fmt.Println("[APAGANDO] Sistemas fuera. Hasta luego, señor.")
}

func esPalabraDeActivacion(comandoLower string, wakeWords []string) bool {
	for _, w := range wakeWords {
		if strings.Contains(comandoLower, strings.ToLower(w)) {
			return true
		}
	}
	return false
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
