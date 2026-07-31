package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const nombreServicio = "JarvisOS"

func InstalarServicio() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo determinar la ruta del ejecutable: %v", err)
	}
	exe, _ = filepath.Abs(exe)

	cmd := exec.Command("sc", "create", nombreServicio,
		"binPath=", fmt.Sprintf(`"%s" --service`, exe),
		"start=", "auto",
		"DisplayName=", "JarvisOS - Asistente Inteligente",
		"description=", "Asistente de voz y control del sistema operativo.",
	)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al crear servicio: %v\n%s", err, string(salida))
	}

	exec.Command("sc", "failure", nombreServicio, "reset=", "86400",
		"actions=", "restart/5000/restart/10000/restart/15000").Run()

	fmt.Printf("Servicio '%s' instalado exitosamente.\n", nombreServicio)
	return nil
}

func DesinstalarServicio() error {
	exec.Command("sc", "stop", nombreServicio).Run()

	cmd := exec.Command("sc", "delete", nombreServicio)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al desinstalar servicio: %v\n%s", err, string(salida))
	}

	fmt.Printf("Servicio '%s' desinstalado.\n", nombreServicio)
	return nil
}

func IniciarServicio() error {
	cmd := exec.Command("sc", "start", nombreServicio)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al iniciar servicio: %v\n%s", err, string(salida))
	}
	fmt.Printf("Servicio '%s' iniciado.\n", nombreServicio)
	return nil
}

func DetenerServicio() error {
	cmd := exec.Command("sc", "stop", nombreServicio)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al detener servicio: %v\n%s", err, string(salida))
	}
	fmt.Printf("Servicio '%s' detenido.\n", nombreServicio)
	return nil
}

func EstadoServicio() string {
	cmd := exec.Command("sc", "query", nombreServicio)
	salida, err := cmd.Output()
	if err != nil {
		return "No instalado"
	}
	texto := string(salida)
	for _, linea := range strings.Split(texto, "\n") {
		linea = strings.TrimSpace(linea)
		if strings.HasPrefix(linea, "STATE") {
			if strings.Contains(linea, "RUNNING") {
				return "Ejecutando"
			}
			if strings.Contains(linea, "STOPPED") {
				return "Detenido"
			}
			return linea
		}
	}
	return "Desconocido"
}
