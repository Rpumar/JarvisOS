package core

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

// === EMAIL (F3): lectura de bandeja por IMAP ===
// Cliente IMAP mínimo con la librería estándar: LOGIN, SELECT y FETCH de
// cabeceras. Sin dependencias externas (no hay red para go-imap en esta
// máquina). Cubre el protocolo básico para leer los últimos N correos.

// CorreoBandeja es un correo resumido de la bandeja.
type CorreoBandeja struct {
	Numero int    `json:"numero"`
	De     string `json:"de"`
	Asunto string `json:"asunto"`
	Fecha  string `json:"fecha"`
}

// imapConexion encapsula una conexión IMAP con su tag de comandos.
type imapConexion struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
	tag  int
}

// imapDial abre la conexión al servidor. Es una variable para poder
// reemplazarla en tests por un dial sin TLS contra un servidor falso.
var imapDial = conectarIMAP

// leerBandejaIMAP conecta por TLS, hace LOGIN/SELECT INBOX y devuelve los
// últimos n correos (por número de secuencia). Devuelve los correos de
// más reciente a más antiguo.
func leerBandejaIMAP(host string, puerto int, usuario, clave string, n int) ([]CorreoBandeja, error) {
	conn, err := imapDial(host, puerto)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return leerBandejaIMAPCon(conn, usuario, clave, n)
}

// leerBandejaIMAPCon ejecuta el protocolo IMAP sobre una conexión ya
// establecida. Separado del dial para poder testear contra un servidor falso
// sin TLS.
func leerBandejaIMAPCon(conn net.Conn, usuario, clave string, n int) ([]CorreoBandeja, error) {
	if n <= 0 {
		n = 10
	}
	c := &imapConexion{conn: conn, r: bufio.NewReader(conn), w: bufio.NewWriter(conn), tag: 1}
	defer c.cerrar()

	if err := c.comando("LOGIN "+citar(usuario)+" "+citar(clave), nil); err != nil {
		return nil, fmt.Errorf("login rechazado (¿clave de aplicación correcta?): %w", err)
	}

	// SELECT INBOX → "TOTAL EXISTS" en una línea "* N EXISTS".
	total := 0
	if err := c.comando("SELECT INBOX", func(linea string) bool {
		if n, ok := extraerExists(linea); ok {
			total = n
		}
		return true
	}); err != nil {
		return nil, fmt.Errorf("no pude abrir la bandeja: %w", err)
	}
	if total == 0 {
		return nil, nil
	}

	inicio := total - n + 1
	if inicio < 1 {
		inicio = 1
	}

	var correos []CorreoBandeja
	// BODY.PEEK[HEADER.FIELDS (FROM SUBJECT DATE)] no marca los correos como
	// leídos y trae solo las cabeceras.
	cmd := fmt.Sprintf("FETCH %d:%d (BODY.PEEK[HEADER.FIELDS (FROM SUBJECT DATE)])", inicio, total)
	err := c.comando(cmd, func(linea string) bool {
		if correo, ok := parsearFetch(linea, c.r); ok {
			correos = append(correos, correo)
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("no pude leer los correos: %w", err)
	}
	return correos, nil
}

// conectarIMAP establece la conexión TLS.
func conectarIMAP(host string, puerto int) (net.Conn, error) {
	if puerto == 0 {
		puerto = 993
	}
	direccion := fmt.Sprintf("%s:%d", host, puerto)
	conn, err := tls.Dial("tcp", direccion, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return nil, fmt.Errorf("no pude conectar a %s: %w", direccion, err)
	}
	return conn, nil
}

// comando envía un comando IMAP y lee respuestas hasta la línea etiquetada
// (tag OK / NO / BAD). visitador recibe cada línea de respuesta no etiquetada.
func (c *imapConexion) comando(cmd string, visitador func(linea string) bool) error {
	c.tag++
	tag := fmt.Sprintf("A%03d", c.tag)
	if _, err := fmt.Fprintf(c.w, "%s %s\r\n", tag, cmd); err != nil {
		return err
	}
	if err := c.w.Flush(); err != nil {
		return err
	}

	for {
		linea, err := c.r.ReadString('\n')
		if err != nil {
			return err
		}
		linea = strings.TrimRight(linea, "\r\n")
		if strings.HasPrefix(linea, tag+" OK") {
			return nil
		}
		if strings.HasPrefix(linea, tag+" NO") || strings.HasPrefix(linea, tag+" BAD") {
			return fmt.Errorf("IMAP %s", linea)
		}
		if strings.HasPrefix(linea, "*") {
			seguir := true
			if visitador != nil {
				seguir = visitador(linea)
			}
			if !seguir {
				return nil
			}
		}
	}
}

// cerrar cierra la conexión.
func (c *imapConexion) cerrar() {
	// Intento de logout amable, ignorando errores.
	_, _ = fmt.Fprintf(c.w, "A%03d LOGOUT\r\n", c.tag+1)
	_ = c.w.Flush()
	_ = c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, _ = io.Copy(io.Discard, c.r)
	c.conn.Close()
}

// extraerExists saca el total de "N EXISTS" de una respuesta SELECT.
func extraerExists(linea string) (int, bool) {
	idx := strings.Index(linea, " EXISTS")
	if idx < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(linea[1:idx]))
	if err != nil {
		return 0, false
	}
	return n, true
}

// parsearFetch lee una respuesta "* N FETCH (BODY[HEADER.FIELDS ...] {m}" con
// su literal de m bytes seguido de CRLF, y arma un CorreoBandeja.
func parsearFetch(linea string, r *bufio.Reader) (CorreoBandeja, bool) {
	if !strings.Contains(linea, "FETCH") {
		return CorreoBandeja{}, false
	}
	// número de secuencia al inicio: "* 3 FETCH ..."
	partes := strings.Fields(linea)
	if len(partes) < 3 {
		return CorreoBandeja{}, false
	}
	numero, err := strconv.Atoi(partes[1])
	if err != nil {
		return CorreoBandeja{}, false
	}

	// El literal viene como "...] {123}" al final de la línea.
	llave := strings.LastIndex(linea, "{")
	if llave < 0 {
		return CorreoBandeja{}, false
	}
	cierre := strings.Index(linea[llave:], "}")
	if cierre < 0 {
		return CorreoBandeja{}, false
	}
	m, err := strconv.Atoi(linea[llave+1 : llave+cierre])
	if err != nil {
		return CorreoBandeja{}, false
	}

	// Leer los m bytes del literal + CRLF.
	buf := make([]byte, m)
	if _, err := io.ReadFull(r, buf); err != nil {
		return CorreoBandeja{}, false
	}
	_, _ = r.ReadString('\n') // consume el CRLF tras el literal

	msg, err := mail.ReadMessage(strings.NewReader(string(buf)))
	if err != nil {
		return CorreoBandeja{}, false
	}
	return CorreoBandeja{
		Numero: numero,
		De:     msg.Header.Get("From"),
		Asunto: msg.Header.Get("Subject"),
		Fecha:  msg.Header.Get("Date"),
	}, true
}

// citar encierra un argumento IMAP entre comillas (escapando internas).
func citar(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
