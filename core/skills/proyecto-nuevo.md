---
nombre: proyecto-nuevo
activar: [crear proyecto, nueva app, nuevo proyecto, nueva web, scaffold, crear una app]
---
Guía para crear o extender proyectos de JarvisOS:
- Un proyecto vive en su propia carpeta con go.mod, main.go y frontend/ (index.html, style.css, app.js).
- main.go levanta un servidor HTTP que sirve frontend/ y expone la API en /api.
- Verificá siempre: go build -o <nombre>.exe ., go vet . y go test ./...
- Generá UN archivo a la vez, completo y compilable, y seguí el estilo del proyecto existente.
- La variable de entorno PORT define el puerto (por defecto 9090).
