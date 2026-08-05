---
name: security-review
description: Checklist de seguridad obligatorio antes de escribir o modificar código que ejecute comandos del sistema, PowerShell, shell, acceso al registro o a archivos del usuario en JarvisOS. Use ONLY when the task involves exec, os/exec, PowerShell, shell, registro, borrar, eliminar, apagar, firewall, usuarios, secretos, password, token, API key, config.json.
---

# Revisión de seguridad — JarvisOS

Este proyecto ejecuta comandos del sistema (`core/hands.go`, `core/armas*.go`). Antes de terminar cualquier cambio que toque ejecución de comandos, pasá esta checklist.

## Prohibiciones ABSOLUTAS

NUNCA generes código que:
- Borre archivos o carpetas del usuario, formatee discos, modifique el registro de Windows, deshabilite firewall/seguridad, cree/modifique usuarios, o apague/reinicie el equipo.
- Descargue o ejecute archivos de internet, ni lea claves o contraseñas.
- Instale, desinstale o modifique programas del sistema.

## Procesos protegidos

- `cerrarApp` (en `core/hands.go`) jamás debe matar los procesos de `procesosProtegidos` (winlogon, csrss, lsass, explorer, etc.). Mantené esa lista intacta y probada.
- Los scripts generados por IA también deben bloquear esos patrones (ver `patronesBloqueados` en `agents/coder_agent.go`).

## Secretos y datos del usuario

- Los datos viven en `%USERPROFILE%\JarvisOS-datos\` (FUERA del repo).
- `config.json` contiene secretos en texto plano (password de email, API keys, tokens). NUNCA lo leas completo, lo muestres, lo traslades al repo ni lo subas a git.
- Si tocás una clave de config, editá solo ese campo y nunca lo imprimas.

## Confirmación explícita

- Toda acción destructiva pasa por confirmación explícita del usuario o queda con explicación clara del riesgo.
- Para código nuevo que ejecute comandos: escapá correctamente los argumentos, validá la entrada, y preferí la lista blanca de comandos conocidos.

## Cómo reportar

- Si la petición es peligrosa o ambigua, no generes el código: explicá por qué y proponé una alternativa segura.
