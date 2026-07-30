param(
    [switch]$NoBuild
)

$ErrorActionPreference = "Stop"

Write-Host "======================================="
Write-Host " Instalador de JarvisOS"
Write-Host "======================================="
Write-Host ""

$script:paso = 1
$total = 5

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

# --- Paso 2: MSYS2 + MinGW ---
Paso "Verificando MSYS2/MinGW (necesario para CGO)..."
$gcc = Get-Command gcc -ErrorAction SilentlyContinue
if (-not $gcc) {
    $candidates = @(
        "C:\msys64\ucrt64\bin\gcc.exe",
        "C:\msys64\mingw64\bin\gcc.exe",
        "C:\msys64\mingw32\bin\gcc.exe"
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) {
            $gcc = Get-Command $c
            break
        }
    }
}
if ($gcc) {
    Write-Host "  GCC encontrado: $($gcc.Source)" -ForegroundColor Green
} else {
    Write-Host "  [FALTA] GCC no encontrado. Instalando MSYS2..." -ForegroundColor Yellow
    $msys2Url = "https://github.com/msys2/msys2-installer/releases/download/2025-01-26/msys2-x86_64-20250126.exe"
    $msys2Installer = "$env:TEMP\msys2.exe"
    Write-Host "  Descargando MSYS2..."
    try {
        $wc = New-Object System.Net.WebClient
        $wc.DownloadFile($msys2Url, $msys2Installer)
        Write-Host "  Instalando MSYS2 (esto lleva unos minutos)..."
        Start-Process -FilePath $msys2Installer -ArgumentList "install --root C:\msys64 --confirm-command --overwrite" -Wait -NoNewWindow
        Write-Host "  Instalando MinGW (gcc)..."
        Start-Process -FilePath "C:\msys64\usr\bin\pacman.exe" -ArgumentList "-S --noconfirm mingw-w64-ucrt-x86_64-gcc" -Wait -NoNewWindow
        $env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
        Write-Host "  GCC instalado." -ForegroundColor Green
    } catch {
        Write-Host "  [ERROR] No se pudo instalar MSYS2 automaticamente: $_" -ForegroundColor Red
        Write-Host "  Instalalo manualmente desde: https://www.msys2.org/" -ForegroundColor Yellow
        Write-Host "  Luego corre: pacman -S mingw-w64-ucrt-x86_64-gcc" -ForegroundColor Yellow
        exit 1
    }
}

# --- Paso 3: Ollama + modelo ---
Paso "Verificando Ollama..."
$ollama = Get-Command ollama -ErrorAction SilentlyContinue
if (-not $ollama) {
    Write-Host "  [FALTA] Ollama no encontrado. Instalando..." -ForegroundColor Yellow
    & "$PSScriptRoot\install_ollama.ps1"
    # Refresh PATH
    $ollama = Get-Command ollama -ErrorAction SilentlyContinue
}

if ($ollama) {
    Write-Host "  Ollama encontrado." -ForegroundColor Green
    Write-Host "  Verificando modelo llama3.2:3b..."
    $modelos = & ollama list 2>&1
    if ($modelos -match "llama3.2") {
        Write-Host "  Modelo llama3.2:3b presente." -ForegroundColor Green
    } else {
        Write-Host "  Descargando modelo llama3.2:3b (esto lleva varios minutos)..."
        & ollama pull llama3.2:3b
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Modelo descargado." -ForegroundColor Green
        } else {
            Write-Host "  [ERROR] No se pudo descargar el modelo. Corre manualmente: ollama pull llama3.2:3b" -ForegroundColor Red
        }
    }
} else {
    Write-Host "  [ADVERTENCIA] Ollama no disponible. El asistente funcionara sin IA." -ForegroundColor Yellow
}

# --- Paso 4: Modelo de voz Vosk ---
Paso "Verificando modelo de voz (Vosk)..."
$modeloDir = "$PSScriptRoot\modelo-voz-es"
if (Test-Path "$modeloDir\am-vosk-model-small-es-0.42") {
    Write-Host "  Modelo Vosk encontrado." -ForegroundColor Green
} else {
    Write-Host "  Descargando modelo de voz espanol (70 MB)..."
    $url = "https://alphacephei.com/vosk/models/vosk-model-small-es-0.42.zip"
    $zip = "$env:TEMP\vosk-model.zip"
    try {
        $wc = New-Object System.Net.WebClient
        $wc.DownloadFile($url, $zip)
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zip, $modeloDir)
        Write-Host "  Modelo de voz descargado y extraido." -ForegroundColor Green
    } catch {
        Write-Host "  [ERROR] No se pudo descargar el modelo: $_" -ForegroundColor Red
        Write-Host "  Descargalo manualmente de: https://alphacephei.com/vosk/models/vosk-model-small-es-0.42.zip" -ForegroundColor Yellow
        Write-Host "  Extraelo en: $modeloDir" -ForegroundColor Yellow
    }
}

# --- Paso 5: Compilar ---
if (-not $NoBuild) {
    Paso "Compilando JarvisOS..."
    $env:CGO_ENABLED = "1"
    $env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
    try {
        & go build -tags cgo -ldflags="-s -w" -o "$PSScriptRoot\JarvisOS.exe" "$PSScriptRoot"
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

Write-Host ""
Write-Host "======================================="
Write-Host " Instalacion completada." -ForegroundColor Green
Write-Host " Ejecuta: .\JarvisOS.exe" -ForegroundColor Cyan
Write-Host "======================================="
