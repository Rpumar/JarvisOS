param(
    [ValidateSet("debug", "release")]
    [string]$Config = "release"
)

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$Output = "JarvisOS.exe"

$env:CGO_ENABLED = "1"

# Auto-detect MSYS2/MinGW GCC si no está en PATH
if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    $candidates = @(
        "C:\msys64\ucrt64\bin",
        "C:\msys64\mingw64\bin",
        "C:\msys64\mingw32\bin"
    )
    foreach ($dir in $candidates) {
        if (Test-Path "$dir\gcc.exe") {
            $env:PATH = "$dir;$env:PATH"
            break
        }
    }
}

$modFlag = "-mod=vendor"
if ($Config -eq "release") {
    Write-Host "[BUILD] Release - $Output"
    & go build $modFlag -tags cgo -ldflags="-s -w" -o "$ProjectRoot\$Output" "$ProjectRoot"
} else {
    Write-Host "[BUILD] Debug - $Output"
    & go build $modFlag -tags cgo -o "$ProjectRoot\$Output" "$ProjectRoot"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] $Output compilado en configuracion $Config"
} else {
    Write-Host "[ERROR] Fallo la compilacion"
    exit 1
}
