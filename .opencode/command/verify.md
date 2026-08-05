---
description: Ejecuta las tres verificaciones obligatorias del proyecto (build, vet, test) desde la raíz.
---
Ejecutá en orden, desde la raíz del repo, las verificaciones obligatorias del proyecto:

```powershell
go build ./...
go vet ./...
go test ./...
```

Usá `-mod=vendor` si `go` intentara resolver de la red.

Reportá si alguna falla y corregí los errores antes de dar la tarea por terminada.
