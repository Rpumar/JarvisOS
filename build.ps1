param(
    [ValidateSet("debug", "release")]
    [string]$Config = "release"
)

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$Output = "JarvisOS.exe"

if ($Config -eq "release") {
    Write-Host "[BUILD] Release - $Output"
    & go build -ldflags="-s -w" -o "$ProjectRoot\$Output" "$ProjectRoot"
} else {
    Write-Host "[BUILD] Debug - $Output"
    & go build -o "$ProjectRoot\$Output" "$ProjectRoot"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] $Output compilado en configuracion $Config"
} else {
    Write-Host "[ERROR] Fallo la compilacion"
    exit 1
}
