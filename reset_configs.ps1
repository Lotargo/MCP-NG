# reset_configs.ps1
# Этот скрипт возвращает все конфиги в состояние, совместимое с 'go run' и 'python'.

Write-Host "Resetting all tool config.json files to their default state..." -ForegroundColor Cyan

# --- 1. Сброс Go-инструментов ---
$goToolsPath = ".\MCP-NG\tools\go"
$goToolDirs = Get-ChildItem -Path $goToolsPath -Directory

foreach ($toolDir in $goToolDirs) {
    $toolName = $toolDir.Name
    $configPath = Join-Path $toolDir.FullName "config.json"
    
    if (Test-Path $configPath) {
        Write-Host "Resetting Go config for: $toolName"
        $config = Get-Content $configPath -Raw | ConvertFrom-Json
        # Возвращаем команду 'go run'
        $config.command = @("go", "run", ".")
        $config | ConvertTo-Json -Depth 5 | Set-Content $configPath
    }
}

# --- 2. Сброс Python-инструментов ---
$pythonToolsPath = ".\MCP-NG\tools\python"
$pythonToolDirs = Get-ChildItem -Path $pythonToolsPath -Directory

foreach ($toolDir in $pythonToolDirs) {
    $toolName = $toolDir.Name
    $configPath = Join-Path $toolDir.FullName "config.json"
    
    if (Test-Path $configPath) {
        Write-Host "Resetting Python config for: $toolName"
        $config = Get-Content $configPath -Raw | ConvertFrom-Json
        # Возвращаем простую команду 'python server.py'
        $config.command = @("python", "server.py")
        $config | ConvertTo-Json -Depth 5 | Set-Content $configPath
    }
}

Write-Host "All tool configs have been reset." -ForegroundColor Green