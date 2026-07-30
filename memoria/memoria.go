package memoria

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Recordatorio struct {
	ID       int64     `json:"id"`
	Texto    string    `json:"texto"`
	Momento  time.Time `json:"momento"`
	Cumplido bool      `json:"cumplido"`
	Periodo  string    `json:"periodo,omitempty"`
}

type Almacen struct {
	mu sync.Mutex
	db *sql.DB
}

func NuevoAlmacen(ruta string) (*Almacen, error) {
	db, err := sql.Open("sqlite", ruta)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la base de datos: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la base de datos: %w", err)
	}
	if err := migrar(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("no se pudo migrar la base de datos: %w", err)
	}
	return &Almacen{db: db}, nil
}

func migrar(db *sql.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS hechos (
		clave TEXT PRIMARY KEY,
		valor TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS notas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		texto TEXT NOT NULL,
		fecha TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS recordatorios (
		id INTEGER PRIMARY KEY,
		texto TEXT NOT NULL,
		momento TEXT NOT NULL,
		cumplido INTEGER NOT NULL DEFAULT 0,
		periodo TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS listas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT NOT NULL UNIQUE
	);
	CREATE TABLE IF NOT EXISTS items_lista (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lista_id INTEGER NOT NULL REFERENCES listas(id),
		texto TEXT NOT NULL,
		hecho INTEGER NOT NULL DEFAULT 0
	);
	`
	_, err := db.Exec(sql)
	return err
}

func (a *Almacen) GuardarHecho(clave, valor string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(
		"INSERT INTO hechos (clave, valor) VALUES (?, ?) ON CONFLICT(clave) DO UPDATE SET valor=excluded.valor",
		clave, valor,
	)
	return err
}

func (a *Almacen) ObtenerHecho(clave string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var valor string
	err := a.db.QueryRow("SELECT valor FROM hechos WHERE clave = ?", clave).Scan(&valor)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return valor, true
}

func (a *Almacen) AgregarNota(texto string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(
		"INSERT INTO notas (texto, fecha) VALUES (?, ?)",
		texto, time.Now().Format(time.RFC3339),
	)
	return err
}

func (a *Almacen) ObtenerNotas() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	rows, err := a.db.Query("SELECT texto, fecha FROM notas ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var resultado []string
	for rows.Next() {
		var texto, fecha string
		if err := rows.Scan(&texto, &fecha); err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, fecha)
		if err != nil {
			resultado = append(resultado, texto)
			continue
		}
		resultado = append(resultado, fmt.Sprintf("%s: %s", t.Format("2006-01-02"), texto))
	}
	return resultado
}

func (a *Almacen) BuscarNotas(texto string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	busqueda := "%" + strings.ToLower(texto) + "%"
	rows, err := a.db.Query(
		"SELECT texto, fecha FROM notas WHERE LOWER(texto) LIKE ? ORDER BY id",
		busqueda,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var resultado []string
	for rows.Next() {
		var texto, fecha string
		if err := rows.Scan(&texto, &fecha); err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, fecha)
		if err != nil {
			resultado = append(resultado, texto)
			continue
		}
		resultado = append(resultado, fmt.Sprintf("%s: %s", t.Format("2006-01-02"), texto))
	}
	return resultado
}

func (a *Almacen) AgregarRecordatorio(texto string, momento time.Time) error {
	return a.AgregarRecordatorioConPeriodo(texto, momento, "")
}

func (a *Almacen) AgregarRecordatorioConPeriodo(texto string, momento time.Time, periodo string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := time.Now().UnixNano()
	_, err := a.db.Exec(
		"INSERT INTO recordatorios (id, texto, momento, cumplido, periodo) VALUES (?, ?, ?, 0, ?)",
		id, texto, momento.Format(time.RFC3339), periodo,
	)
	return err
}

func (a *Almacen) RecordatoriosPendientes(ahora time.Time) []Recordatorio {
	a.mu.Lock()
	defer a.mu.Unlock()
	ahoraStr := ahora.Format(time.RFC3339)
	rows, err := a.db.Query(
		"SELECT id, texto, momento, periodo FROM recordatorios WHERE cumplido = 0 AND momento <= ?",
		ahoraStr,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var pendientes []Recordatorio
	for rows.Next() {
		var id int64
		var texto, momentoStr, periodo string
		if err := rows.Scan(&id, &texto, &momentoStr, &periodo); err != nil {
			continue
		}
		momento, err := time.Parse(time.RFC3339, momentoStr)
		if err != nil {
			continue
		}
		r := Recordatorio{ID: id, Texto: texto, Momento: momento, Cumplido: false, Periodo: periodo}
		if periodo != "" {
			pendientes = append(pendientes, r)
			nuevoMomento := siguienteOcurrencia(momento, periodo, ahora)
			_, _ = a.db.Exec(
				"UPDATE recordatorios SET momento = ? WHERE id = ?",
				nuevoMomento.Format(time.RFC3339), id,
			)
			continue
		}
		pendientes = append(pendientes, r)
	}
	return pendientes
}

func siguienteOcurrencia(ultima time.Time, periodo string, ahora time.Time) time.Time {
	switch periodo {
	case "diario":
		prox := ultima.Add(24 * time.Hour)
		for prox.Before(ahora) {
			prox = prox.Add(24 * time.Hour)
		}
		return prox
	case "semanal":
		prox := ultima.Add(7 * 24 * time.Hour)
		for prox.Before(ahora) {
			prox = prox.Add(7 * 24 * time.Hour)
		}
		return prox
	case "mensual":
		prox := ultima.AddDate(0, 1, 0)
		for prox.Before(ahora) {
			prox = prox.AddDate(0, 1, 0)
		}
		return prox
	}
	return ultima.Add(24 * time.Hour)
}

func (a *Almacen) MarcarCumplido(id int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec("UPDATE recordatorios SET cumplido = 1 WHERE id = ?", id)
	return err
}

func (a *Almacen) ObtenerRecordatoriosPendientesTexto() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	rows, err := a.db.Query(
		"SELECT id, texto, momento, periodo FROM recordatorios WHERE cumplido = 0 ORDER BY id",
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var resultado []string
	for rows.Next() {
		var id int64
		var texto, momentoStr, periodo string
		if err := rows.Scan(&id, &texto, &momentoStr, &periodo); err != nil {
			continue
		}
		momento, err := time.Parse(time.RFC3339, momentoStr)
		if err != nil {
			continue
		}
		s := fmt.Sprintf("%s (%s)", texto, momento.Format("2006-01-02 15:04"))
		if periodo != "" {
			s += fmt.Sprintf(" [%s]", periodo)
		}
		resultado = append(resultado, s)
	}
	return resultado
}

func (a *Almacen) CancelarRecordatorios(textoBusqueda string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if textoBusqueda == "" {
		res, err := a.db.Exec("UPDATE recordatorios SET cumplido = 1 WHERE cumplido = 0")
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}
	busqueda := "%" + strings.ToLower(textoBusqueda) + "%"
	res, err := a.db.Exec(
		"UPDATE recordatorios SET cumplido = 1 WHERE cumplido = 0 AND LOWER(texto) LIKE ?",
		busqueda,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (a *Almacen) CrearLista(nombre string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec("INSERT INTO listas (nombre) VALUES (?)", nombre)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("ya existe una lista llamada '%s'", nombre)
		}
		return err
	}
	return nil
}

func (a *Almacen) AgregarItemLista(nombreLista, item string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var listaID int64
	err := a.db.QueryRow(
		"SELECT id FROM listas WHERE LOWER(nombre) = LOWER(?)",
		nombreLista,
	).Scan(&listaID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("no encontré la lista '%s'", nombreLista)
	}
	if err != nil {
		return err
	}
	_, err = a.db.Exec("INSERT INTO items_lista (lista_id, texto) VALUES (?, ?)", listaID, item)
	return err
}

func (a *Almacen) MarcarItemLista(nombreLista, item string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var listaID int64
	err := a.db.QueryRow(
		"SELECT id FROM listas WHERE LOWER(nombre) = LOWER(?)",
		nombreLista,
	).Scan(&listaID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no encontré la lista '%s'", nombreLista)
	}
	if err != nil {
		return "", err
	}
	busq := "%" + strings.ToLower(item) + "%"
	var texto string
	err = a.db.QueryRow(
		"SELECT texto FROM items_lista WHERE lista_id = ? AND LOWER(texto) LIKE ? AND hecho = 0 LIMIT 1",
		listaID, busq,
	).Scan(&texto)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no encontré '%s' en la lista '%s'", item, nombreLista)
	}
	if err != nil {
		return "", err
	}
	_, err = a.db.Exec("UPDATE items_lista SET hecho = 1 WHERE lista_id = ? AND texto = ?", listaID, texto)
	if err != nil {
		return "", err
	}
	return texto, nil
}

func (a *Almacen) ObtenerListas() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	rows, err := a.db.Query("SELECT id, nombre FROM listas ORDER BY id")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var resultado []string
	for rows.Next() {
		var id int64
		var nombre string
		if err := rows.Scan(&id, &nombre); err != nil {
			continue
		}
		s := nombre + ":"
		items, err := a.db.Query("SELECT texto, hecho FROM items_lista WHERE lista_id = ? ORDER BY id", id)
		if err != nil {
			resultado = append(resultado, s+" (error al leer items)")
			continue
		}
		primerItem := true
		for items.Next() {
			var itemTexto string
			var hecho int
			if err := items.Scan(&itemTexto, &hecho); err != nil {
				continue
			}
			if primerItem {
				primerItem = false
			}
			estado := "☐"
			if hecho == 1 {
				estado = "☑"
			}
			s += fmt.Sprintf("\n  %s %s", estado, itemTexto)
		}
		items.Close()
		if primerItem {
			s += " (vacía)"
		}
		resultado = append(resultado, s)
	}
	return resultado
}

func (a *Almacen) ObtenerLista(nombre string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var id int64
	err := a.db.QueryRow(
		"SELECT id FROM listas WHERE LOWER(nombre) = LOWER(?)",
		nombre,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	s := nombre + ":"
	items, err := a.db.Query("SELECT texto, hecho FROM items_lista WHERE lista_id = ? ORDER BY id", id)
	if err != nil {
		return s + " (error)", true
	}
	defer items.Close()
	primerItem := true
	for items.Next() {
		var itemTexto string
		var hecho int
		if err := items.Scan(&itemTexto, &hecho); err != nil {
			continue
		}
		if primerItem {
			primerItem = false
		}
		estado := "☐"
		if hecho == 1 {
			estado = "☑"
		}
		s += fmt.Sprintf("\n  %s %s", estado, itemTexto)
	}
	if primerItem {
		s += " (vacía)"
	}
	return s, true
}

func (a *Almacen) EliminarLista(nombre string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var id int64
	err := a.db.QueryRow(
		"SELECT id FROM listas WHERE LOWER(nombre) = LOWER(?)",
		nombre,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("no encontré la lista '%s'", nombre)
	}
	if err != nil {
		return err
	}
	_, _ = a.db.Exec("DELETE FROM items_lista WHERE lista_id = ?", id)
	_, err = a.db.Exec("DELETE FROM listas WHERE id = ?", id)
	return err
}

func (a *Almacen) Cerrar() error {
	return a.db.Close()
}
