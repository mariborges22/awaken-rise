$path = "migracao.sh"
$out = "migracao_lf.sh"
$zip = "migracao.zip"

# Garante que estamos no diretório correto
$currentDir = Get-Location
Write-Host "Trabalhando no diretório: $currentDir"

if (Test-Path $path) {
    Write-Host "Lendo $path..."
    # Lê o conteúdo original
    $text = [System.IO.File]::ReadAllText((Join-Path $currentDir $path))
    
    # Substitui CRLF por LF
    $text = $text -replace "`r`n", "`n"

    Write-Host "Salvando em formato LF (Unix) em $out..."
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText((Join-Path $currentDir $out), $text, $utf8NoBom)

    Write-Host "Criando arquivo $zip..."
    if (Test-Path $zip) { Remove-Item $zip -Force }
    Compress-Archive -Path $out -DestinationPath $zip

    Write-Host "---"
    Write-Host "Sucesso! O arquivo '$zip' foi gerado com o script no formato LF."
} else {
    Write-Error "Erro: O arquivo '$path' não foi encontrado neste diretório."
}
