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
	fmt.Println("[MODO TEXTO] Usando entrada por teclado.")
	fmt.Println("[MODO TEXTO] Escribá su comando y presione Enter.")
	return &Ears{lector: bufio.NewReader(os.Stdin)}, nil
}

func (e *Ears) EscucharTexto() (string, error) {
	fmt.Print("> ")
	texto, err := e.lector.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error al leer entrada: %w", err)
	}
	return strings.TrimSpace(texto), nil
}

func (e *Ears) Escuchar() (string, error) {
	return e.EscucharTexto()
}

func (e *Ears) Cerrar() {}