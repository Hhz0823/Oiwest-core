$ErrorActionPreference = "Continue"
$BUILD_DIR = "build"
$VERSION = "2.1.0"
$CGO = "0"
$LDFLAGS = "-s -w"
$GOPROXY = "https://goproxy.cn,direct"

Remove-Item -Recurse -Force $BUILD_DIR -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $BUILD_DIR | Out-Null

$platforms = @(
    @{OS="windows";  ARCH="amd64";    CLI_NAME="oiwest-core.exe";   DAEMON_NAME="oiwest-daemon.exe";   DIR="windows-amd64"   },
    @{OS="windows";  ARCH="arm64";    CLI_NAME="oiwest-core.exe";   DAEMON_NAME="oiwest-daemon.exe";   DIR="windows-arm64"   },
    @{OS="windows";  ARCH="386";      CLI_NAME="oiwest-core.exe";   DAEMON_NAME="oiwest-daemon.exe";   DIR="windows-386"     },
    @{OS="linux";    ARCH="amd64";    CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-amd64"     },
    @{OS="linux";    ARCH="arm64";    CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-arm64"     },
    @{OS="linux";    ARCH="arm";      CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-armv7";    },
    @{OS="linux";    ARCH="386";      CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-386"       },
    @{OS="linux";    ARCH="mips";     CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-mips"      },
    @{OS="linux";    ARCH="mipsle";   CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-mipsle"    },
    @{OS="linux";    ARCH="mips64";   CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-mips64"    },
    @{OS="linux";    ARCH="mips64le"; CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="linux-mips64le"  },
    @{OS="darwin";   ARCH="amd64";    CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="darwin-amd64"    },
    @{OS="darwin";   ARCH="arm64";    CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="darwin-arm64"    },
    @{OS="android";  ARCH="arm64";    CLI_NAME="oiwest-core";        DAEMON_NAME="oiwest-daemon";        DIR="android-arm64"   }
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Oiwest Core v$VERSION - Build & Package" -ForegroundColor Cyan
Write-Host "  Target platforms: $($platforms.Count)" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

$total = $platforms.Count
$current = 0
$failed = @()

foreach ($p in $platforms) {
    $current++
    $dir = Join-Path $BUILD_DIR $p.DIR
    New-Item -ItemType Directory -Force $dir | Out-Null
    
    $label = "$($p.OS)/$($p.ARCH)".PadRight(20)
    $percent = [math]::Round($current / $total * 100)
    Write-Host "[$current/$total ${percent}%] Building $label ... " -NoNewline -ForegroundColor Yellow

    $env:GOOS = $p.OS
    $env:GOARCH = $p.ARCH
    $env:CGO_ENABLED = $CGO
    $env:GOPROXY = $GOPROXY
    
    if ($p.ARCH -eq "arm") {
        $env:GOARM = "7"
    }
    if ($p.ARCH -eq "mips" -or $p.ARCH -eq "mipsle") {
        $env:GOMIPS = "softfloat"
    }

    $cliPath = Join-Path $dir $p.CLI_NAME
    $daemonPath = Join-Path $dir $p.DAEMON_NAME

    $cliOk = $true
    $daemonOk = $true

    try {
        $cliResult = go build -ldflags "$LDFLAGS" -o $cliPath ./app/cmd/cli 2>&1
        if ($LASTEXITCODE -ne 0) { $cliOk = $false; $failed += "$($p.DIR)-cli" }
    } catch {
        $cliOk = $false
        $failed += "$($p.DIR)-cli"
    }

    try {
        $daemonResult = go build -ldflags "$LDFLAGS" -o $daemonPath ./app/cmd/daemon 2>&1
        if ($LASTEXITCODE -ne 0) { $daemonOk = $false; $failed += "$($p.DIR)-daemon" }
    } catch {
        $daemonOk = $false
        $failed += "$($p.DIR)-daemon"
    }

    if ($cliOk -and $daemonOk) {
        $cliSize = "{0:N2} MB" -f ((Get-Item $cliPath).Length / 1MB)
        $daemonSize = "{0:N2} MB" -f ((Get-Item $daemonPath).Length / 1MB)
        Write-Host "OK (cli:$cliSize daemon:$daemonSize)" -ForegroundColor Green
    } elseif ($cliOk) {
        Write-Host "PARTIAL (daemon failed)" -ForegroundColor DarkYellow
    } elseif ($daemonOk) {
        Write-Host "PARTIAL (cli failed)" -ForegroundColor DarkYellow
    } else {
        Write-Host "FAILED" -ForegroundColor Red
    }
}

# Reset env
Remove-Item env:GOOS -ErrorAction SilentlyContinue
Remove-Item env:GOARCH -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Packaging ZIP archives..." -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan

$zipDir = Join-Path $BUILD_DIR "zip"
New-Item -ItemType Directory -Force $zipDir | Out-Null

Get-ChildItem $BUILD_DIR -Directory | Where-Object { $_.Name -ne "zip" } | ForEach-Object {
    $targetDir = $_.FullName
    $targetName = $_.Name
    $versionedName = "oiwest-core-v${VERSION}-${targetName}.zip"
    $zipPath = Join-Path $zipDir $versionedName

    $files = Get-ChildItem $targetDir -File
    if ($files.Count -gt 0) {
        Write-Host "  $targetName -> $versionedName" -ForegroundColor White
        try {
            Compress-Archive -Path "$targetDir\*" -DestinationPath $zipPath -Force
            $zipSize = "{0:N2} MB" -f ((Get-Item $zipPath).Length / 1MB)
            Write-Host "    Size: $zipSize" -ForegroundColor DarkGray
        } catch {
            Write-Host "    ZIP failed: $_" -ForegroundColor Red
        }
    }
}

Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Build Summary" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Version     : $VERSION" -ForegroundColor White
Write-Host "  Total       : $total platforms" -ForegroundColor White
Write-Host "  Failed      : $($failed.Count) " -NoNewline -ForegroundColor White
if ($failed.Count -gt 0) {
    Write-Host "($failed -join ', ')" -ForegroundColor Red
} else {
    Write-Host "(0)" -ForegroundColor Green
}
Write-Host ""

$zipFiles = Get-ChildItem $zipDir -File
Write-Host "  ZIP archives: $($zipFiles.Count)" -ForegroundColor White
foreach ($z in $zipFiles) {
    $size = "{0:N2} MB" -f ($z.Length / 1MB)
    Write-Host "    $($z.Name) ($size)" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "  Output: $(Resolve-Path $zipDir)" -ForegroundColor Green
Write-Host "  Build complete!" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Cyan

