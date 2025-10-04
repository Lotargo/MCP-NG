Write-Host "Updating all Go tool config.json files..." -ForegroundColor Cyan

$goToolsPath = ".\MCP-NG\tools\go"
$goToolDirs = Get-ChildItem -Path $goToolsPath -Directory

foreach ($toolDir in $goToolDirs) {
    $toolName = $toolDir.Name
    $configPath = Join-Path $toolDir.FullName "config.json"
    
    if (Test-Path $configPath) {
        Write-Host "Updating config for: $toolName"
        try {
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config.command = @($toolName)
            $config | ConvertTo-Json -Depth 5 | Set-Content $configPath
        } catch {
            Write-Warning "Could not process $($configPath): $_"
        }
    }
}

Write-Host "All Go tool configs have been updated to use binary names." -ForegroundColor Green