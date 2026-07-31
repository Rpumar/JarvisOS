package core

import "strings"

// plantillasProyecto devuelve los archivos de una aplicación fullstack nueva:
// backend Go con la librería estándar y frontend HTML/CSS/JS puro. "NOMBRE" se
// reemplaza por el nombre real del proyecto. No se usan backticks en el código
// generado para poder embeberse como string crudo.
func plantillasProyecto(nombre string) map[string]string {
	return map[string]string{
		"go.mod": strings.ReplaceAll("module NOMBRE\n\ngo 1.25\n", "NOMBRE", nombre),
		"main.go": strings.ReplaceAll(plantillaMainGo, "NOMBRE", nombre),
		"frontend/index.html": strings.ReplaceAll(plantillaIndexHTML, "NOMBRE", nombre),
		"frontend/style.css":  strings.ReplaceAll(plantillaStyleCSS, "NOMBRE", nombre),
		"frontend/app.js":     strings.ReplaceAll(plantillaAppJS, "NOMBRE", nombre),
		"README.md":           strings.ReplaceAll(plantillaREADME, "NOMBRE", nombre),
	}
}

const plantillaMainGo = `package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Tarea struct {
	ID    int
	Texto string
	Hecho bool
}

type AppEstado struct {
	App    string
	Hora   string
	Fecha  string
	Uptime string
	Count  string
}

var (
	mu          sync.Mutex
	tareas      []Tarea
	siguienteID = 1
	inicio      = time.Now()
)

func main() {
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "9090"
	}
	http.Handle("/", http.FileServer(http.Dir("frontend")))
	http.HandleFunc("/api/estado", rutaEstado)
	http.HandleFunc("/api/tareas", rutaTareas)
	http.HandleFunc("/api/tareas/", rutaTareaID)
	log.Printf("=== NOMBRE escuchando en http://127.0.0.1:%s ===", puerto)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+puerto, nil))
}

func jsonRespuesta(w http.ResponseWriter, valor interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(valor)
}

func rutaEstado(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	conteo := strconv.Itoa(len(tareas))
	mu.Unlock()
	jsonRespuesta(w, AppEstado{
		App:    "NOMBRE",
		Hora:   time.Now().Format("15:04:05"),
		Fecha:  time.Now().Format("02/01/2006"),
		Uptime: time.Since(inicio).Truncate(time.Second).String(),
		Count:  conteo,
	})
}

func rutaTareas(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	if r.Method == http.MethodGet {
		jsonRespuesta(w, tareas)
		return
	}
	if r.Method == http.MethodPost {
		var t Tarea
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "json invalido", http.StatusBadRequest)
			return
		}
		t.ID = siguienteID
		siguienteID++
		tareas = append(tareas, t)
		jsonRespuesta(w, t)
		return
	}
	http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
}

func rutaTareaID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/tareas/"))
	if err != nil {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for i, t := range tareas {
		if t.ID == id {
			tareas = append(tareas[:i], tareas[i+1:]...)
			jsonRespuesta(w, map[string]bool{"borrado": true})
			return
		}
	}
	http.Error(w, "tarea no encontrada", http.StatusNotFound)
}
`

const plantillaIndexHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NOMBRE — Panel</title>
<link rel="stylesheet" href="style.css">
</head>
<body>
<header>
  <div class="marca">
    <span class="logo">◈</span>
    <div>
      <h1>NOMBRE</h1>
      <p>Generado por JarvisOS · Desarrollador Fullstack</p>
    </div>
  </div>
  <div class="stats">
    <div class="stat"><span class="valor" id="hora">--:--:--</span><span class="etiqueta">hora</span></div>
    <div class="stat"><span class="valor" id="fecha">--/--/----</span><span class="etiqueta">fecha</span></div>
    <div class="stat"><span class="valor" id="uptime">0s</span><span class="etiqueta">activo</span></div>
    <div class="stat"><span class="valor" id="count">0</span><span class="etiqueta">tareas</span></div>
  </div>
</header>
<main>
  <section class="tarjeta">
    <h2>Mis tareas</h2>
    <form id="form-tarea">
      <input id="input-tarea" type="text" placeholder="Nueva tarea..." autocomplete="off">
      <button type="submit">Agregar</button>
    </form>
    <ul id="lista"></ul>
    <p id="vacio" class="vacio">No hay tareas todavía.</p>
  </section>
</main>
<footer>JarvisOS · Ingeniero y Desarrollador Fullstack</footer>
<script src="app.js"></script>
</body>
</html>
`

const plantillaStyleCSS = `:root {
  --fondo: #0b0f17;
  --panel: #121826;
  --panel2: #1a2233;
  --borde: #253049;
  --texto: #e6edf7;
  --tenue: #8b9bb8;
  --acento: #4ade80;
  --acento2: #38bdf8;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: 'Segoe UI', system-ui, sans-serif;
  background: var(--fondo);
  color: var(--texto);
  min-height: 100vh;
}
header {
  display: flex; align-items: center; justify-content: space-between;
  flex-wrap: wrap; gap: 16px;
  padding: 20px 28px;
  border-bottom: 1px solid var(--borde);
  background: linear-gradient(180deg, #10172a, var(--fondo));
}
.marca { display: flex; align-items: center; gap: 14px; }
.logo { font-size: 30px; color: var(--acento2); }
.marca h1 { font-size: 22px; letter-spacing: .5px; }
.marca p { color: var(--tenue); font-size: 13px; }
.stats { display: flex; gap: 12px; flex-wrap: wrap; }
.stat {
  background: var(--panel); border: 1px solid var(--borde);
  border-radius: 12px; padding: 10px 16px; min-width: 100px;
  display: flex; flex-direction: column;
}
.stat .valor { font-size: 18px; font-weight: 600; font-variant-numeric: tabular-nums; }
.stat .etiqueta { font-size: 11px; color: var(--tenue); text-transform: uppercase; letter-spacing: .6px; }
main { max-width: 720px; margin: 0 auto; padding: 28px 20px; }
.tarjeta {
  background: var(--panel); border: 1px solid var(--borde);
  border-radius: 16px; padding: 22px;
}
.tarjeta h2 { font-size: 16px; margin-bottom: 16px; color: var(--acento2); }
#form-tarea { display: flex; gap: 10px; }
#input-tarea {
  flex: 1; background: var(--panel2); border: 1px solid var(--borde);
  color: var(--texto); border-radius: 10px; padding: 10px 14px; font-size: 14px;
}
#input-tarea:focus { outline: none; border-color: var(--acento2); }
button {
  background: var(--acento); color: #06280f; border: none; cursor: pointer;
  border-radius: 10px; padding: 10px 18px; font-weight: 700; font-size: 14px;
}
button:hover { filter: brightness(1.08); }
#lista { list-style: none; margin-top: 16px; display: flex; flex-direction: column; gap: 8px; }
#lista li {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  background: var(--panel2); border: 1px solid var(--borde);
  border-radius: 10px; padding: 10px 14px; font-size: 14px;
}
#lista li .hecho { text-decoration: line-through; color: var(--tenue); }
button.borrar { background: #dc2626; color: #fff; padding: 6px 12px; font-size: 12px; }
.vacio { color: var(--tenue); text-align: center; margin-top: 18px; font-size: 14px; }
footer { text-align: center; color: var(--tenue); font-size: 12px; padding: 20px; }
`

const plantillaAppJS = `var API = '/api';

function cargarEstado() {
  fetch(API + '/estado')
    .then(function (r) { return r.json(); })
    .then(function (e) {
      document.getElementById('hora').textContent = e.Hora;
      document.getElementById('fecha').textContent = e.Fecha;
      document.getElementById('uptime').textContent = e.Uptime;
      document.getElementById('count').textContent = e.Count;
    })
    .catch(function () {});
}

function renderTareas(lista) {
  var ul = document.getElementById('lista');
  ul.innerHTML = '';
  var vacio = document.getElementById('vacio');
  if (!lista || lista.length === 0) {
    vacio.style.display = 'block';
    return;
  }
  vacio.style.display = 'none';
  lista.forEach(function (t) {
    var li = document.createElement('li');
    var span = document.createElement('span');
    span.textContent = t.Texto;
    if (t.Hecho) { span.className = 'hecho'; }
    var boton = document.createElement('button');
    boton.textContent = 'Eliminar';
    boton.className = 'borrar';
    boton.addEventListener('click', function () { borrarTarea(t.ID); });
    li.appendChild(span);
    li.appendChild(boton);
    ul.appendChild(li);
  });
}

function cargarTareas() {
  fetch(API + '/tareas')
    .then(function (r) { return r.json(); })
    .then(renderTareas)
    .catch(function () {});
}

function borrarTarea(id) {
  fetch(API + '/tareas/' + id, { method: 'DELETE' })
    .then(function () { cargarTareas(); cargarEstado(); });
}

document.getElementById('form-tarea').addEventListener('submit', function (ev) {
  ev.preventDefault();
  var texto = document.getElementById('input-tarea').value.trim();
  if (!texto) { return; }
  fetch(API + '/tareas', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ Texto: texto, Hecho: false })
  })
    .then(function () {
      document.getElementById('input-tarea').value = '';
      cargarTareas();
      cargarEstado();
    });
});

cargarEstado();
cargarTareas();
setInterval(cargarEstado, 1000);
`

const plantillaREADME = `# NOMBRE

Aplicación fullstack generada por JarvisOS (Go + HTML/CSS/JS puro, sin dependencias externas).

## Estructura

- main.go - servidor HTTP con API JSON (backend).
- frontend/ - interfaz web: index.html, style.css y app.js.
- jarvis.log - salida de la última ejecución (se crea al ejecutarse).

## API

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /api/estado | hora, fecha, uptime y cantidad de tareas |
| GET | /api/tareas | lista de tareas |
| POST | /api/tareas | crea una tarea (JSON: {"Texto": "..."}) |
| DELETE | /api/tareas/:id | borra una tarea |

## Uso

- Compilar: go build -o NOMBRE.exe .
- Ejecutar: set PORT=9090 y luego NOMBRE.exe, y abrir http://127.0.0.1:9090
- Por voz: "ejecutar proyecto NOMBRE", "detener proyecto NOMBRE", "estado del proyecto NOMBRE".
`
