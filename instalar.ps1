param(
    [switch]$NoBuild
)

$ErrorActionPreference = "Stop"

Write-Host "======================================="
Write-Host " Instalador de JarvisOS"
Write-Host "======================================="
Write-Host ""

$script:paso = 1
$total = 4

function Paso($titulo) {
    Write-Host "[$script:paso/$total] $titulo" -ForegroundColor Cyan
    $script:paso++
}

# --- Paso 1: Go ---
Paso "Verificando Go..."
try {
    $ver = go version
    Write-Host "  $ver" -ForegroundColor Green
} catch {
    Write-Host "  [FALTA] Go no encontrado. Descargalo de https://go.dev/dl/ e instalalo." -ForegroundColor Red
    Write-Host "  Elegi el .msi para Windows, siguiente, siguiente, finalizar."
    exit 1
}

# --- Paso 2: Ollama + modelo ---
Paso "Verificando Ollama..."
$ollama = Get-Command ollama -ErrorAction SilentlyContinue
if (-not $ollama) {
    Write-Host "  [FALTA] Ollama no encontrado. Instalando..." -ForegroundColor Yellow
    & "$PSScriptRoot\install_ollama.ps1"
    $ollama = Get-Command ollama -ErrorAction SilentlyContinue
}

if ($ollama) {
    Write-Host "  Ollama encontrado." -ForegroundColor Green
    Write-Host "  Verificando modelo qwen2.5-coder:7b..."
    $modelos = & ollama list 2>&1
    if ($modelos -match "qwen2.5-coder") {
        Write-Host "  Modelo qwen2.5-coder:7b presente." -ForegroundColor Green
    } else {
        Write-Host "  Descargando modelo qwen2.5-coder:7b (esto lleva varios minutos)..."
        & ollama pull qwen2.5-coder:7b
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Modelo descargado." -ForegroundColor Green
        } else {
            Write-Host "  [ERROR] No se pudo descargar el modelo. Corre manualmente: ollama pull qwen2.5-coder:7b" -ForegroundColor Red
        }
    }
} else {
    Write-Host "  [ADVERTENCIA] Ollama no disponible. El asistente funcionara sin IA." -ForegroundColor Yellow
}

# --- Paso 3: Compilar ---
if (-not $NoBuild) {
    Paso "Compilando JarvisOS..."
    try {
        & go build -ldflags="-s -w" -o "$PSScriptRoot\JarvisOS.exe" "$PSScriptRoot"
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Compilacion exitosa: JarvisOS.exe" -ForegroundColor Green
        } else {
            Write-Host "  [ERROR] Fallo la compilacion." -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "  [ERROR] Fallo la compilacion: $_" -ForegroundColor Red
        exit 1
    }
}

# --- Paso 4: Acceso web ---
Paso "Preparando acceso web..."
$datosDir = Join-Path $env:USERPROFILE "JarvisOS-datos"
New-Item -ItemType Directory -Force -Path $datosDir | Out-Null
Write-Host "  Panel web disponible con: .\JarvisOS.exe --web" -ForegroundColor Green
Write-Host "  URL: http://127.0.0.1:8080" -ForegroundColor Green
Write-Host "  (La contrasena del panel se define dentro de JarvisOS con el comando correspondiente.)" -ForegroundColor Green

Write-Host ""
Write-Host "======================================="
Write-Host " Instalacion completada."
Write-Host " Ejecuta: .\JarvisOS.exe" -ForegroundColor Cyan
Write-Host "======================================="
