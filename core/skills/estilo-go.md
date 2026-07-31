---
nombre: estilo-go
activar: [go, golang, backend, codigo go, funcion, main.go, handler]
---
Convenciones de Go para este proyecto:
- Usá SOLO la librería estándar: net/http, encoding/json, sync, time, strconv. Nada de dependencias externas.
- Nombres en camelCase para variables locales y PascalCase para tipos y funciones exportadas. Los errores se devuelven como último valor y se usan defer para recursos.
- El backend sirve el frontend con http.FileServer(http.Dir("frontend")) y expone la API bajo /api con respuestas JSON (Content-Type application/json; charset=utf-8).
- Protegé el estado compartido con sync.Mutex para evitar data races.
- Cada handler debe ser corto; mové la lógica a funciones auxiliares cuando crezca.
