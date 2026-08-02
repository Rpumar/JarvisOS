package core

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// fetchRangoRe extrae "inicio:fin" de un comando FETCH.
var fetchRangoRe = regexp.MustCompile(`FETCH (\d+):(\d+)`)

// imapServidorFalso habla IMAP lo mínimo para validar el cliente: greeting,
// LOGIN, SELECT INBOX, FETCH con literales y LOGOUT.
type imapServidorFalso struct {
	ln     net.Listener
	correos int
}

func nuevoImapFalso(t *testing.T, correos int) (string, int, func()) {
	t.Helper()
	s := &imapServidorFalso{correos: correos}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s.ln = ln

	go s.aceptar(t)
	host := "127.0.0.1"
	puerto := ln.Addr().(*net.TCPAddr).Port
	return host, puerto, func() { ln.Close() }
}

func nuevoImapFalsoLoginRechazado(t *testing.T) (string, int, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				r := bufio.NewReader(conn)
				w := bufio.NewWriter(conn)
				fmt.Fprint(w, "* OK Server ready\r\n")
				w.Flush()
				for {
					linea, err := r.ReadString('\n')
					if err != nil {
						return
					}
					partes := strings.SplitN(strings.TrimRight(linea, "\r\n"), " ", 3)
					if len(partes) < 2 {
						continue
					}
					tag, comando := partes[0], partes[1]
					if comando == "LOGIN" {
						fmt.Fprintf(w, "%s NO [AUTHENTICATIONFAILED] Invalid credentials\r\n", tag)
						w.Flush()
					} else {
						fmt.Fprintf(w, "%s BAD unknown\r\n", tag)
						w.Flush()
					}
				}
			}()
		}
	}()
	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, func() { ln.Close() }
}

func (s *imapServidorFalso) aceptar(t *testing.T) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.atender(t, conn)
	}
}

func (s *imapServidorFalso) atender(t *testing.T, conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	fmt.Fprint(w, "* OK IMAP4rev1 Server ready\r\n")
	w.Flush()

	for {
		linea, err := r.ReadString('\n')
		if err != nil {
			return
		}
		linea = strings.TrimRight(linea, "\r\n")
		partes := strings.SplitN(linea, " ", 3)
		if len(partes) < 2 {
			continue
		}
		tag, comando := partes[0], partes[1]

		switch comando {
		case "LOGIN":
			fmt.Fprintf(w, "%s OK LOGIN completed\r\n", tag)
		case "SELECT":
			fmt.Fprintf(w, "* %d EXISTS\r\n", s.correos)
			fmt.Fprintf(w, "%s OK [READ-WRITE] SELECT completed\r\n", tag)
		case "FETCH":
			// FETCH inicio:fin (...) — responder solo ese rango.
			inicio, fin := 1, s.correos
			if m := fetchRangoRe.FindStringSubmatch(linea); m != nil {
				inicio, _ = strconv.Atoi(m[1])
				fin, _ = strconv.Atoi(m[2])
			}
			for i := fin; i >= inicio; i-- {
				literal := fmt.Sprintf("From: remitente%d@empresa.com\r\nSubject: Asunto %d\r\nDate: Mon, 1 Jan 2024 10:00:00 +0000\r\n\r\n", i, i)
				fmt.Fprintf(w, "* %d FETCH (BODY[HEADER.FIELDS (FROM SUBJECT DATE)] {%d}\r\n", i, len(literal))
				fmt.Fprint(w, literal)
				fmt.Fprint(w, ")\r\n")
			}
			fmt.Fprintf(w, "%s OK FETCH completed\r\n", tag)
		case "LOGOUT":
			fmt.Fprint(w, "* BYE Logging out\r\n")
			fmt.Fprintf(w, "%s OK LOGOUT completed\r\n", tag)
			w.Flush()
			return
		default:
			fmt.Fprintf(w, "%s BAD unknown command\r\n", tag)
		}
		w.Flush()
	}
}

func TestLeerBandejaIMAP(t *testing.T) {
	host, puerto, cerrar := nuevoImapFalso(t, 3)
	defer cerrar()

	conn, err := net.Dial("tcp", net.JoinHostPort(host, fmt.Sprint(puerto)))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	correos, err := leerBandejaIMAPCon(conn, "yo@gmail.com", "clave-app", 5)
	if err != nil {
		t.Fatalf("leerBandejaIMAP: %v", err)
	}
	if len(correos) != 3 {
		t.Fatalf("se esperaban 3 correos, hay %d", len(correos))
	}
	if correos[0].Numero != 3 || correos[0].Asunto != "Asunto 3" {
		t.Errorf("el más reciente debería ser el #3, fue %+v", correos[0])
	}
	if !strings.Contains(correos[0].De, "remitente3") {
		t.Errorf("debería traer el remitente, fue %q", correos[0].De)
	}
}

func TestLeerBandejaIMAPLimite(t *testing.T) {
	host, puerto, cerrar := nuevoImapFalso(t, 10)
	defer cerrar()

	conn, err := net.Dial("tcp", net.JoinHostPort(host, fmt.Sprint(puerto)))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	correos, err := leerBandejaIMAPCon(conn, "yo@gmail.com", "clave-app", 3)
	if err != nil {
		t.Fatalf("leerBandejaIMAP: %v", err)
	}
	if len(correos) != 3 {
		t.Fatalf("con límite 3 se esperaban 3, hay %d", len(correos))
	}
	if correos[0].Numero != 10 {
		t.Errorf("el primero debería ser el #10, fue %d", correos[0].Numero)
	}
}

func TestLeerBandejaIMAPLoginRechazado(t *testing.T) {
	host, puerto, cerrar := nuevoImapFalsoLoginRechazado(t)
	defer cerrar()

	conn, err := net.Dial("tcp", net.JoinHostPort(host, fmt.Sprint(puerto)))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, err = leerBandejaIMAPCon(conn, "yo@gmail.com", "clave-mala", 5)
	if err == nil {
		t.Fatal("debería fallar si el login es rechazado")
	}
	if !strings.Contains(err.Error(), "login rechazado") {
		t.Errorf("error poco claro: %v", err)
	}
}

func TestManejarEmailLeeBandeja(t *testing.T) {
	host, puerto, cerrar := nuevoImapFalso(t, 2)
	defer cerrar()

	antiguo := imapDial
	imapDial = func(h string, p int) (net.Conn, error) {
		return net.Dial("tcp", net.JoinHostPort(h, fmt.Sprint(p)))
	}
	defer func() { imapDial = antiguo }()

	h := &Hands{
		EmailEnabled:  true,
		EmailImapHost: host,
		EmailImapPort: puerto,
		EmailUsuario:  "yo@gmail.com",
		EmailPassword: "clave-app",
		EmailImapMax:  5,
	}
	msg := h.leerEmail("leé los últimos 2 correos")
	if !strings.Contains(msg, "2 correos") {
		t.Errorf("debería listar 2 correos, fue: %q", msg)
	}
	if !strings.Contains(msg, "Asunto") {
		t.Errorf("debería listar los asuntos, fue: %q", msg)
	}
}

func TestExtraerExists(t *testing.T) {
	casos := map[string]struct {
		n  int
		ok bool
	}{
		"* 7 EXISTS":        {7, true},
		"* 0 EXISTS":        {0, true},
		"* 3 RECENT":        {0, false},
		"* OK INBOX":        {0, false},
	}
	for linea, esperado := range casos {
		n, ok := extraerExists(linea)
		if ok != esperado.ok || n != esperado.n {
			t.Errorf("extraerExists(%q) = %d,%v, esperaba %d,%v", linea, n, ok, esperado.n, esperado.ok)
		}
	}
}

