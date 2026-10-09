<#
.SYNOPSIS
    Compila Lua para Windows e empaqueta o instalador (lua_installer).
    Equivalente Windows do obxectivo "build"/"files" do Makefile (que é só
    bash/apt+wails v2, pensado para Linux): compila lua.exe (Wails v3, via
    Taskfile - ver lua/Taskfile.yml) e mételo en files/ (o que
    install.go despois extrae a destDir vía go:embed), e por último
    "wails build" (CLI v2, este instalador segue en Wails v2) para xerar
    build\bin\lua-installer.exe. Ver Makefile para a versión Linux/make e
    install.go/nomeExecutable para por que o binario ten que chamarse
    "lua.exe" aquí dentro.

.PARAMETER SkipFiles
    Salta a recompilación de lua.exe e a copia a files\ (usa o que xa
    haxa aí). Útil se só cambiou o frontend do propio instalador.

.EXAMPLE
    .\build-windows.ps1
#>
[CmdletBinding()]
param(
    [switch]$SkipFiles
)

$ErrorActionPreference = "Stop"

$installerDir = $PSScriptRoot
$luaDir = (Resolve-Path (Join-Path $installerDir "..\lua")).Path

function Require-Command {
    param([string]$Name, [string]$Hint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Non se atopou '$Name' no PATH. $Hint"
    }
}

Write-Host "==> Comprobando dependencias de compilación..." -ForegroundColor Cyan
Require-Command go    "Instala Go: https://go.dev/dl/"
Require-Command wails "Instala: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
Require-Command wails3 "Instala: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
Require-Command task  "Instala: winget install Task.Task (ou choco install go-task)"
Require-Command npm   "Instala Node.js: https://nodejs.org/"

# ── Logo: mesma orixe ca assets/logo.png e build/appicon.png do Makefile ───
Write-Host "==> Copiando logo dende lua..." -ForegroundColor Cyan
$logoSrc = Join-Path $luaDir "logo.png"
New-Item -ItemType Directory -Force -Path (Join-Path $installerDir "assets") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $installerDir "build") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $installerDir "frontend\src\assets\images") | Out-Null
Copy-Item $logoSrc (Join-Path $installerDir "assets\logo.png") -Force
Copy-Item $logoSrc (Join-Path $installerDir "build\appicon.png") -Force
Copy-Item $logoSrc (Join-Path $installerDir "frontend\src\assets\images\logo.png") -Force

if (-not $SkipFiles) {
    # ── files/: o que extractFiles (install.go) escribe en destDir. Mesmos
    # tres ficheiros ca "files:" do Makefile (lua, logo.png, modulo.json),
    # pero "lua.exe" en vez de "lua". ──────────────────────────────────────
    Write-Host "==> Compilando lua.exe (Wails v3, task build)..." -ForegroundColor Cyan
    Push-Location $luaDir
    try {
        task build
        if ($LASTEXITCODE -ne 0) { throw "task build (lua) fallou" }
    } finally {
        Pop-Location
    }

    Write-Host "==> Empaquetando files\..." -ForegroundColor Cyan
    $filesDir = Join-Path $installerDir "files"
    if (Test-Path $filesDir) { Remove-Item -Recurse -Force $filesDir }
    New-Item -ItemType Directory -Force -Path $filesDir | Out-Null
    Copy-Item (Join-Path $luaDir "build\bin\lua.exe") (Join-Path $filesDir "lua.exe") -Force
    Copy-Item (Join-Path $luaDir "logo.png") (Join-Path $filesDir "logo.png") -Force
    Copy-Item (Join-Path $luaDir "modulo.json") (Join-Path $filesDir "modulo.json") -Force
    Write-Host "✅ lua.exe compilado e ficheiros copiados a lua_installer\files\" -ForegroundColor Green
} else {
    Write-Host "==> -SkipFiles: uso files\ tal como está" -ForegroundColor Yellow
}

# ── O propio instalador (segue en Wails v2) ─────────────────────────────────
Push-Location $installerDir
try {
    Write-Host "==> go mod tidy..." -ForegroundColor Cyan
    go mod tidy
    if ($LASTEXITCODE -ne 0) { throw "go mod tidy fallou" }

    Write-Host "==> wails build..." -ForegroundColor Cyan
    wails build
    if ($LASTEXITCODE -ne 0) { throw "wails build fallou" }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "✅ Compilado: lua_installer\build\bin\lua-installer.exe" -ForegroundColor Green
Write-Host "   Execútao coma administrador (UAC pediraino automaticamente, ver elevate_normal.go)."
