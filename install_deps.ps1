# install_deps.ps1 (Версия 2.0 - с компиляцией и правилами брандмауэра)
# Запускать от имени Администратора!

# --- Начало скрипта ---

# Проверяем, запущен ли скрипт от имени администратора
if (-Not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as Administrator to add firewall rules. Please re-run from an elevated PowerShell terminal."
    exit 1
}

Write-Host "Starting setup for all tools (dependencies and firewall rules)..." -ForegroundColor Green

# Определяем пути
$projectRoot = Get-Location
$goToolsPath = Join-Path $projectRoot "MCP-NG\tools\go"
$pythonToolsPath = Join-Path $projectRoot "MCP-NG\tools\python"
$pythonExecutable = Join-Path $projectRoot ".venv\Scripts\python.exe"

# --- 1. Обработка Go-микросервисов ---
Write-Host "`n[GO] Searching for Go tools in: $goToolsPath" -ForegroundColor Cyan
$goToolDirs = Get-ChildItem -Path $goToolsPath -Directory

foreach ($toolDir in $goToolDirs) {
    $toolName = $toolDir.Name
    $goModPath = Join-Path $toolDir.FullName "go.mod"
    
    if (Test-Path $goModPath) {
        Write-Host "  -> Found Go tool: $toolName" -ForegroundColor Yellow
        
        # Установка зависимостей
        Write-Host "     - Running 'go mod tidy'..."
        Push-Location $toolDir.FullName
        go mod tidy
        
        # Компиляция в .exe файл
        $exePath = Join-Path $toolDir.FullName "$toolName.exe"
        Write-Host "     - Compiling to '$exePath'..."
        go build -o $exePath .
        
        # Добавление правила в брандмауэр
        $ruleName = "Allow MCP-NG Tool: $toolName"
        Write-Host "     - Adding firewall rule: '$ruleName'..."
        # Сначала удаляем старое правило, если оно есть, чтобы избежать дубликатов
        netsh advfirewall firewall delete rule name=$ruleName dir=in > $null
        # Добавляем новое правило
        netsh advfirewall firewall add rule name=$ruleName dir=in action=allow program=$exePath enable=yes

        Pop-Location
    }
}

# --- 2. Обработка Python-микросервисов ---
Write-Host "`n[PYTHON] Searching for Python tools in: $pythonToolsPath" -ForegroundColor Cyan
if (Test-Path $pythonExecutable) {
    # Добавление правила в брандмауэр для Python из venv
    $ruleName = "Allow MCP-NG Python venv"
    Write-Host "  -> Adding firewall rule for Python interpreter: '$($pythonExecutable)'..." -ForegroundColor Yellow
    netsh advfirewall firewall delete rule name=$ruleName dir=in > $null
    netsh advfirewall firewall add rule name=$ruleName dir=in action=allow program=$pythonExecutable enable=yes
    
    # Установка зависимостей
    Write-Host "  -> Installing Python dependencies from root requirements.txt..."
    pip install -r (Join-Path $projectRoot "requirements.txt")
} else {
    Write-Warning "Python interpreter not found at '$pythonExecutable'. Skipping Python setup."
}

Write-Host "`nSetup complete! All tools are compiled and firewall rules are set." -ForegroundColor Green

# --- Конец скрипта ---