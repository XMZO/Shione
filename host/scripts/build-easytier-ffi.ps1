$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$easyTierRoot = Join-Path $repoRoot "..\\onani\\EasyTier"
$releaseDir = Join-Path $easyTierRoot "target\\release"

if (-not $env:LIBCLANG_PATH) {
    $defaultLibclang = "E:\\gayhub\\LLVM\\bin"
    if (Test-Path (Join-Path $defaultLibclang "libclang.dll")) {
        $env:LIBCLANG_PATH = $defaultLibclang
    }
}

if (-not $env:PROTOC) {
    $defaultProtoc = "C:\\Users\\john\\.cache\\shione-tools\\protoc\\33.3\\bin\\protoc.exe"
    if (Test-Path $defaultProtoc) {
        $env:PROTOC = $defaultProtoc
    }
}

if (-not $env:LIBCLANG_PATH) {
    throw "LIBCLANG_PATH is not set and no default libclang.dll was found."
}

if (-not $env:PROTOC) {
    throw "PROTOC is not set and no cached protoc.exe was found."
}

Write-Host "Using LIBCLANG_PATH=$env:LIBCLANG_PATH"
Write-Host "Using PROTOC=$env:PROTOC"

Push-Location $easyTierRoot
try {
    cargo build -p easytier-ffi --release
}
finally {
    Pop-Location
}

switch ($env:PROCESSOR_ARCHITECTURE.ToUpperInvariant()) {
    "AMD64" { $runtimeArch = "x86_64" }
    "ARM64" { $runtimeArch = "arm64" }
    "X86" { $runtimeArch = "i686" }
    default { $runtimeArch = $null }
}

if ($runtimeArch) {
    $runtimeDir = Join-Path $easyTierRoot "easytier\\third_party\\$runtimeArch"
    foreach ($name in @("Packet.dll", "wintun.dll", "WinDivert64.sys")) {
        $source = Join-Path $runtimeDir $name
        if (Test-Path $source) {
            Copy-Item $source $releaseDir -Force
            Write-Host "Staged $name to $releaseDir"
        }
    }
}
