# Найти и заменить ВО ВСЕХ файлах
Get-ChildItem -Path . -Recurse -Include *.go, go.mod, go.sum | ForEach-Object {
    $content = Get-Content $_.FullName -Raw
    $original = $content
    
    # Заменяем ВСЕ варианты неправильных путей
    $content = $content -replace 'dnachulino5', 'dmachulin05'
    $content = $content -replace 'github\.com/[^/]+/survey-auth-service/authorization-server/', ''
    
    if ($content -ne $original) {
        Set-Content $_.FullName $content -Encoding UTF8
        Write-Host "Исправлен: $($_.FullName)"
    }
}

# Удалить go.sum чтобы пересоздать
Remove-Item go.sum -ErrorAction SilentlyContinue
