# PowerShell script to start the bibbl-log-stream executable

Set-Location "c:\Users\GregoryHall\bibbl-log-stream"

Write-Host "Starting Bibbl Log Stream..." -ForegroundColor Green

# Check if executable exists
if (-not (Test-Path "bibbl-stream.exe")) {
    Write-Host "Error: bibbl-stream.exe not found. Please build first." -ForegroundColor Red
    exit 1
}

# Start options - choose one:

# Option 1: Run with default config
Write-Host "Starting with default configuration..." -ForegroundColor Yellow
.\bibbl-stream.exe

# Option 2: Run with specific config file (uncomment to use)
# .\bibbl-stream.exe --config config.yaml

# Option 3: Run with web UI on specific port (uncomment to use)
# .\bibbl-stream.exe --web-port 8080

# Option 4: Run in debug mode (uncomment to use)
# .\bibbl-stream.exe --debug

# Option 5: Install as Windows service (requires admin) (uncomment to use)
# .\bibbl-stream.exe install --service
