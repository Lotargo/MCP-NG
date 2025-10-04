# install_deps.ps1 (Version 6.0 - Back to Basics, Robust)
# This script prepares the complete local development environment.
# Must be run with Administrator privileges.

if (-Not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as Administrator. Please re-run from an elevated PowerShell terminal."
    exit 1
}

Write-Host "Starting complete project setup..." -ForegroundColor Green

# --- Define project paths ---
$projectRoot = Get-Location
$goToolsPath = Join-Path $projectRoot "MCP-NG\tools\go"
$pythonExecutable = Join-Path $projectRoot ".venv\Scripts\python.exe"
$binPath = Join-Path $projectRoot "bin"

if (-not (Test-Path $binPath)) {
    New-Item -ItemType Directory -Path $binPath
    Write-Host "Created 'bin' directory at $binPath"
}

# --- 1. Main Server Setup ---
Write-Host "`n[SERVER] Setting up main server..." -ForegroundColor Cyan
$serverSourcePath = Join-Path $projectRoot "MCP-NG\server\cmd\server"
$serverExePath = Join-Path $binPath "server.exe"

Write-Host "  -> Compiling main server to '$serverExePath'..."
Push-Location $serverSourcePath
go build -o $serverExePath . # Compile from its own directory
Pop-Location

$ruleName = "Allow MCP-NG Main Server"
Write-Host "  -> Adding firewall rule for main server..."
netsh advfirewall firewall delete rule name=$ruleName dir=in > $null
netsh advfirewall firewall add rule name=$ruleName dir=in action=allow program=$serverExePath enable=yes

# --- 2. Go Tools Setup ---
Write-Host "`n[GO TOOLS] Compiling tools and setting firewall rules..." -ForegroundColor Cyan
$goToolDirs = Get-ChildItem -Path $goToolsPath -Directory

foreach ($toolDir in $goToolDirs) {
    $toolName = $toolDir.Name
    $goModPath = Join-Path $toolDir.FullName "go.mod"
    
    if (Test-Path $goModPath) {
        Write-Host "  -> Processing Go tool: $toolName" -ForegroundColor Yellow
        
        Push-Location $toolDir.FullName # Go into the tool's directory
        
        Write-Host "     - Running 'go mod tidy'..."
        go mod tidy
        
        $exePath = Join-Path $binPath "$toolName.exe"
        Write-Host "     - Compiling to '$exePath'..."
        go build -o $exePath . # Compile from its own directory
        
        Pop-Location # Go back to the original directory
        
        $ruleName = "Allow MCP-NG Tool: $toolName"
        Write-Host "     - Adding firewall rule for '$exePath'..."
        netsh advfirewall firewall delete rule name=$ruleName dir=in > $null
        netsh advfirewall firewall add rule name=$ruleName dir=in action=allow program=$exePath enable=yes
    }
}

# --- 3. Python Environment Setup ---
Write-Host "`n[PYTHON] Setting up Python environment..." -ForegroundColor Cyan
if (Test-Path $pythonExecutable) {
    $ruleName = "Allow MCP-NG Python venv"
    Write-Host "  -> Adding firewall rule for Python interpreter..." -ForegroundColor Yellow
    netsh advfirewall firewall delete rule name=$ruleName dir=in > $null
    netsh advfirewall firewall add rule name=$ruleName dir=in action=allow program=$pythonExecutable enable=yes
    
    Write-Host "  -> Installing Python dependencies..."
    $requirementsFile = Join-Path $projectRoot "requirements_for_windows.txt"
    # --- ИЗМЕНЕНИЕ: Запускаем pip через явный путь к python.exe ---
    & $pythonExecutable -m pip install -r $requirementsFile
} else {
    Write-Warning "Python interpreter not found at '$pythonExecutable'. Skipping Python setup."
}

Write-Host "`nComplete setup finished! You can now run the compiled server." -ForegroundColor Green
Write-Host "In a new terminal, run '.\bin\server.exe' to start the application." -ForegroundColor Yellow