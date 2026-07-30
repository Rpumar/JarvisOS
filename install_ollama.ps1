Write-Host "======================================="
Write-Host " Instalador de Ollama para JarvisOS"
Write-Host "======================================="
Write-Host ""

# Detectar arquitectura
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -eq "ARM64") {
    $url = "https://github.com/ollama/ollama/releases/latest/download/OllamaSetup-arm64.exe"
} else {
    $url = "https://github.com/ollama/ollama/releases/latest/download/OllamaSetup.exe"
}

$destino = "$env:TEMP\OllamaSetup.exe"

Write-Host "[1/3] Descargando Ollama desde $url ..."
try {
    $wc = New-Object System.Net.WebClient
    $wc.DownloadFile($url, $destino)
    Write-Host " Descarga completada."
} catch {
    Write-Host "[ERROR] No se pudo descargar Ollama: $_"
    Write-Host "Descarguelo manualmente desde: https://ollama.com/download"
    exit 1
}

Write-Host "[2/3] Instalando Ollama (esto requiere permisos de administrador)..."
try {
    Start-Process -FilePath $destino -ArgumentList "/S" -Wait -NoNewWindow
    Write-Host " Instalacion completada."
} catch {
    Write-Host "[ERROR] No se pudo instalar Ollama: $_"
    exit 1
}

Write-Host "[3/3] Descargando modelo Llama 3.2 3B (esto puede tomar varios minutos)..."
try {
    & "$env:LOCALAPPDATA\Programs\Ollama\ollama.exe" pull llama3.2:3b
    Write-Host " Modelo descargado correctamente."
} catch {
    Write-Host "[ADVERTENCIA] No se pudo descargar el modelo automaticamente."
    Write-Host "Ejecute manualmente: ollama pull llama3.2:3b"
}

Write-Host ""
Write-Host "======================================="
Write-Host " Instalacion completada."
Write-Host "======================================="
Write-Host ""
Write-Host "Pasos siguientes:"
Write-Host "1. Asegurese de que Ollama este corriendo en segundo plano"
Write-Host "2. Ejecute JarvisOS.exe"
Write-Host "3. Disfrute de JarvisOS con IA activa"
