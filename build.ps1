# PowerShell script to build the bibbl-log-stream executable

Set-Location "c:\Users\GregoryHall\bibbl-log-stream"

Write-Host "Building Bibbl Log Stream..." -ForegroundColor Green

# Clean previous build
if (Test-Path "bibbl-stream.exe") {
    Remove-Item "bibbl-stream.exe"
}

# Get dependencies
Write-Host "Fetching dependencies..." -ForegroundColor Yellow
go mod download
go mod tidy

# Build web UI (prefer make; fallback to npm)
if (Get-Command make -ErrorAction SilentlyContinue) {
    Write-Host "Building web UI via make..." -ForegroundColor Yellow
    make web
} else {
    Write-Host "Make not found, building web UI via npm..." -ForegroundColor Yellow
    Push-Location "internal/web"
    try {
        if (Test-Path "package-lock.json") {
            Write-Host "Running npm ci..." -ForegroundColor Yellow
            npm ci
        } else {
            Write-Host "Running npm install..." -ForegroundColor Yellow
            npm install
        }
        Write-Host "Running npm run build..." -ForegroundColor Yellow
        npm run build
    } finally {
        Pop-Location
    }
}

# Build the executable
Write-Host "Compiling Windows executable..." -ForegroundColor Yellow
$buildTime = Get-Date -UFormat "%Y%m%d.%H%M%S"
$ldflags = "-s -w -X main.Version=1.0.0 -X main.BuildTime=$buildTime"

go build -ldflags="$ldflags" -o bibbl-stream.exe ./cmd/bibbl

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Output: bibbl-stream.exe" -ForegroundColor Green
    $fileSize = [math]::Round((Get-Item bibbl-stream.exe).Length / 1MB, 2)
    Write-Host "File size: $fileSize MB" -ForegroundColor Cyan
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
