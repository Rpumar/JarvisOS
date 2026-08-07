param(
    [string]$DemoDir = (Join-Path $env:TEMP "jarvisos-demo-datos"),
    [string]$EmailHost = "",
    [string]$EmailUser = "",
    [string]$EmailPass = "",
    [switch]$SkipBuild,
    [switch]$Run
)

$ErrorActionPreference = "Stop"

$emailEnabled = $false
if ($EmailHost -and $EmailUser -and $EmailPass) {
    $emailEnabled = $true
}

Write-Host "=============================================" -ForegroundColor Cyan
Write-Host " JarvisOS - Demo guiada (3 escenarios)"
Write-Host " Sandbox de datos: $DemoDir"
if ($emailEnabled) {
    Write-Host " Email SMTP: activo para $EmailUser"
} else {
    Write-Host " Email SMTP: desactivado (use -EmailHost/-EmailUser/-EmailPass)"
}
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""

# --- 1. Compilar ---
if (-not $SkipBuild) {
    Write-Host "[1/4] Compilando JarvisOS (release)..." -ForegroundColor Cyan
    & "$PSScriptRoot\build.ps1" -Config release
    if ($LASTEXITCODE -ne 0) { exit 1 }
} else {
    Write-Host "[1/4] Compilacion omitida (-SkipBuild)." -ForegroundColor Cyan
}

# --- 2. Preparar sandbox ---
Write-Host "[2/4] Preparando sandbox de demo..." -ForegroundColor Cyan
if (Test-Path $DemoDir) {
    Remove-Item -Recurse -Force $DemoDir
}
New-Item -ItemType Directory -Force -Path (Join-Path $DemoDir "informes") | Out-Null

# PIN y contrasena de la demo (1234 / demo2026) - solo para la sandbox.
function HashTexto([string]$texto) {
    $sha = [System.Security.Cryptography.SHA256]::Create()
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($texto)
    $hash = $sha.ComputeHash($bytes)
    return ([System.BitConverter]::ToString($hash)).Replace('-', '').ToLower()
}
$pinHash = HashTexto "1234"
$claveHash = HashTexto "demo2026"

# EscribirJSON guarda el archivo en UTF-8 SIN BOM (Go no acepta BOM en JSON).
function EscribirJSON([string]$ruta, $obj) {
    $json = $obj | ConvertTo-Json -Depth 6
    [System.IO.File]::WriteAllText($ruta, $json, [System.Text.UTF8Encoding]::new($false))
}
function EscribirJSONL([string]$ruta, [object[]]$lineas) {
    $sb = New-Object System.Text.StringBuilder
    foreach ($l in $lineas) {
        [void]$sb.AppendLine(($l | ConvertTo-Json -Compress))
    }
    [System.IO.File]::WriteAllText($ruta, $sb.ToString(), [System.Text.UTF8Encoding]::new($false))
}

# config.json de la demo (ruta memoria apunta dentro de la sandbox).
$config = @{
    app_name = "JARVISOS"
    version = "0.14.0"
    require_approval = $true
    timeout_segundos = 30
    ruta_memoria = (Join-Path $DemoDir "memoria.json")
    max_historial_ia = 20
    modelo_ia = "mistral:latest"
    ia_url = ""
    ia_api_key = ""
    pin_hash = $pinHash
    workspace_root = (Join-Path $env:USERPROFILE "Desktop")
    open_weather_key = ""
    news_api_key = ""
    comando_timeout_segundos = 30
    login_password_hash = $claveHash
    email_enabled = $emailEnabled
    email_smtp_host = $EmailHost
    email_smtp_port = 587
    email_usuario = $EmailUser
    email_password = $EmailPass
    email_desde = "Jarvis"
    email_imap_host = ""
    email_imap_port = 993
    email_imap_max = 10
}
EscribirJSON (Join-Path $DemoDir "config.json") $config

# Perfil de empresa de la demo.
$empresa = @{
    nombre = "Consultora ABC"
    rubro = "Consultoria de gestion"
    descripcion = "Consultora que prepara informes de gestion, presupuestos y presentaciones para sus clientes."
    tamano = "5 empleados"
    productos = @("Informes de gestion", "Presupuestos", "Presentaciones mensuales")
    clientes = @("Pymes locales", "Profesionales independientes")
    facturacion = "USD 15.000/mes aprox."
    costos = "Salarios, herramientas de oficina"
    objetivos = @("Reducir 5 horas semanales de tareas operativas", "Informar cada dia al dueno")
    equipo = "Dueno + 2 analistas + 2 asistentes"
    competidores = @()
    redes = @()
    contacto_dueno = "Sr. Dueno"
    contacto_mail = "info@consultoraabc.com"
    contacto_telefono = "+54 11 5555 0199"
}
EscribirJSON (Join-Path $DemoDir "empresa.json") $empresa

# Perfil de usuarios: dueno + operador.
$perfil = @{
    usuarios = @(
        @{ nombre = "Sr. Dueno"; area = "Direccion"; rol = "dueno" },
        @{ nombre = "Operador"; area = "Operaciones"; rol = "empleado" }
    )
    activo = "dueno"
}
EscribirJSON (Join-Path $DemoDir "perfil.json") $perfil

# Ordenes de demo: 2 cumplidas en el periodo (para las metricas del piloto)
# y 1 pendiente (para el escenario 1).
$hoy = Get-Date
$ordenes = @(
    @{
        id = 1
        objetivo = "preparar la presentacion mensual"
        pedido_por = "dueno"
        estado = "terminada"
        fecha_creacion = $hoy.AddDays(-5).ToString("yyyy-MM-dd") + " 09:00"
        historial = @(
            @{ momento = "09:05"; accion = "iniciar orden"; resultado = "preparar la presentacion mensual" },
            @{ momento = "09:07"; accion = "abrir powerpoint"; resultado = "PowerPoint abierto" },
            @{ momento = "09:09"; accion = "verificar"; resultado = "Presentacion generada y guardada en el workspace." }
        )
        reporte = "Presentacion mensual generada y guardada."
    },
    @{
        id = 2
        objetivo = "redactar y guardar el informe semanal"
        pedido_por = "dueno"
        estado = "terminada"
        fecha_creacion = $hoy.AddDays(-3).ToString("yyyy-MM-dd") + " 10:00"
        historial = @(
            @{ momento = "10:05"; accion = "iniciar orden"; resultado = "redactar y guardar el informe semanal" },
            @{ momento = "10:08"; accion = "crear archivo"; resultado = "informe-semanal.txt creado en el workspace" }
        )
        reporte = "Informe semanal redactado y guardado."
    },
    @{
        id = 3
        objetivo = "preparar la presentacion mensual"
        pedido_por = "dueno"
        estado = "pendiente"
        fecha_creacion = $hoy.ToString("yyyy-MM-dd") + " 08:30"
        historial = @(@{ momento = "08:31"; accion = "iniciar orden"; resultado = "preparar la presentacion mensual" })
    }
)
EscribirJSON (Join-Path $DemoDir "ordenes.json") $ordenes

# Tareas de demo.
$tareas = @(
    @{ id = 1; nombre = "enviar el presupuesto al cliente"; detalle = ""; fecha = ""; hecha = $true; creada = $hoy.AddDays(-2).ToString("yyyy-MM-dd") + " 11:00"; completada = $hoy.AddDays(-2).ToString("yyyy-MM-dd") + " 11:30" },
    @{ id = 2; nombre = "repasar la agenda de manana"; detalle = ""; fecha = ""; hecha = $false; creada = $hoy.ToString("yyyy-MM-dd") + " 09:00"; completada = "" }
)
EscribirJSON (Join-Path $DemoDir "tareas.json") $tareas

# Agenda de demo: reunion manana.
$agenda = @(
    @{ id = 1; titulo = "Reunion con el cliente"; inicio = $hoy.AddDays(1).ToString("yyyy-MM-dd") + " 15:00"; ubicacion = ""; cancelado = $false }
)
EscribirJSON (Join-Path $DemoDir "agenda.json") $agenda

# Procedimientos de demo: el dueno ya le enseno a Jarvis como se hace la
# presentacion mensual (escenario 1 se cumple de verdad).
$procs = @{
    procedimientos = @(
        @{
            nombre = "preparar la presentacion mensual"
            pasos = @("crear archivo presentacion-mensual", "buscar archivo presupuesto")
        }
    )
}
EscribirJSON (Join-Path $DemoDir "procedimientos.json") $procs

# Auditoria de demo: acciones del periodo (aprobadas, denegada, expirada)
# para que el informe de piloto muestre numeros reales.
$auditoria = @()
$acciones = @(
    @{ cmd = "enviar un email al cliente con el presupuesto"; res = "Email enviado a info@consultoraabc.com" },
    @{ cmd = "publicar en x el aviso de la promo"; res = "Tuit publicado (aprobado por el dueno)" },
    @{ cmd = "borrar la carpeta de respaldos"; res = "denegada_por_el_dueno" },
    @{ cmd = "publicar en linkedin el informe"; res = "expirado_por_timeout_aprobacion" }
)
for ($i = 0; $i -lt $acciones.Count; $i++) {
    $a = $acciones[$i]
    $auditoria += [ordered]@{
        momento = $hoy.AddDays(-($i + 1)).ToString("yyyy-MM-dd") + " 14:00:00"
        usuario = "dueno"
        rol = "dueno"
        orden = ($i % 2) + 1
        comando = $a.cmd
        resultado = $a.res
    }
}
EscribirJSONL (Join-Path $DemoDir "auditoria.jsonl") $auditoria

Write-Host "  Sandbox lista. Datos demo sembrados." -ForegroundColor Green
Write-Host ""

# --- 3. Guion de la demo ---
Write-Host "[3/4] Guion de la demo (3 escenarios):" -ForegroundColor Cyan
Write-Host ""
Write-Host "  ESCENARIO 1 - Informe semanal (documentacion y trazabilidad)" -ForegroundColor Yellow
Write-Host "    > agenda una orden preparar la presentacion mensual"
Write-Host "    > ejecuta la orden #4"
Write-Host "    > reporta la orden #4"
Write-Host ""
Write-Host "  ESCENARIO 2 - Email con aprobacion (control total)" -ForegroundColor Yellow
if ($emailEnabled) {
    Write-Host "    > envia un email a info@consultoraabc.com con asunto Presupuesto y el texto Adjunto el presupuesto"
    Write-Host "    > aprobar la orden #N (ingrese el PIN 1234)"
} else {
    Write-Host "    [Email desactivado: pase -EmailHost, -EmailUser y -EmailPass para mostrarlo.]" -ForegroundColor Red
    Write-Host "    Mientras tanto: 'borrar la carpeta de respaldos' muestra la confirmacion del dueno."
    Write-Host "    Confirme con 'si': Jarvis respondera que aun no sabe ejecutarla y como ensenarle."
}
Write-Host ""
Write-Host "  ESCENARIO 3 - Panel del dueno + informe de piloto (ROI)" -ForegroundColor Yellow
Write-Host "    > informe del piloto"
Write-Host "    Panel web: http://127.0.0.1:8080  (usuario dueno, contrasena demo2026)"
Write-Host ""

# --- 4. Lanzar ---
Write-Host "[4/4] Lanzando JarvisOS en la sandbox..." -ForegroundColor Cyan
$env:JARVISOS_DATOS = $DemoDir

$exe = Join-Path $PSScriptRoot "JarvisOS.exe"
if (-not (Test-Path $exe)) {
    Write-Host "  [ERROR] No esta JarvisOS.exe. Compile primero (build.ps1 o sin -SkipBuild)." -ForegroundColor Red
    exit 1
}

if ($Run) {
    Write-Host "  Consola de JarvisOS (dele las ordenes de la demo)." -ForegroundColor Green
    & $exe
    if ($LASTEXITCODE -ne 0) { Write-Host "  Salida con codigo $LASTEXITCODE" -ForegroundColor Yellow }
} else {
    Write-Host "  La sandbox quedo lista. La variable JARVISOS_DATOS quedo fijada para esta terminal,"
    Write-Host "  asi que la proxima corrida usa la sandbox sin tocar sus datos reales."
    Write-Host "  Ejecute a mano (en esta misma terminal):"
    Write-Host "    & `"$exe`""
    Write-Host ""
    Write-Host "  O lance la demo guiada completa: .\demo.ps1 -Run"
}
