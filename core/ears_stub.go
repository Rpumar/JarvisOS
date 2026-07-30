//go:build !cgo && !windows

package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Ears struct {
	lector *bufio.Reader
}

func NewEars(rutaModelo string) (*Ears, error) {
	fmt.Println("[MODO TEXTO] Sin CGo: usando entrada por teclado en vez de micrófono.")
	fmt.Println("[MODO TEXTO] Para activarme, escribí 'jarvis' y presioná Enter.")
	return &Ears{lector: bufio.NewReader(os.Stdin)}, nil
}

func (e *Ears) Escuchar() (string, error) {
	fmt.Print("> ")
	texto, err := e.lector.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error al leer entrada: %w", err)
	}
	return strings.TrimSpace(texto), nil
}

func (e *Ears) Cerrar() {}
