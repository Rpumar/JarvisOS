---
nombre: rest-api
activar: [api, endpoint, rest, servidor, json, crud, crear endpoint, nueva ruta]
---
Convenciones de la API del proyecto:
- Los endpoints viven bajo /api y devuelven JSON con Content-Type application/json; charset=utf-8.
- GET para leer, POST para crear, DELETE para borrar. Mantené esa semántica.
- Validá el JSON de entrada y respondé http.Error con el código correcto (400 bad request, 404 no encontrado, 405 método no permitido).
- Un recurso se expone como /api/recurso (colección) y /api/recurso/{id} (elemento).
- Si el recurso necesita persistencia, guardala en un archivo JSON de datos dentro del proyecto, no en memoria pura.
