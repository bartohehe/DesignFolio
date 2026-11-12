<#
.SYNOPSIS
    Kopia zapasowa katalogu data/ (baza + pliki).

.DESCRIPTION
    Backup to po prostu kopia katalogu — nic więcej. Baza SQLite jest jednym
    plikiem, pliki leżą obok, więc odtworzenie polega na skopiowaniu tego
    z powrotem.

    WAŻNE: przy włączonym trybie WAL baza składa się z trzech plików
    (.db, .db-wal, .db-shm). Skrypt kopiuje wszystkie i to wystarcza,
    ALE żeby mieć gwarancję spójności, zatrzymaj aplikację przed backupem:

        docker compose stop

    Kopia zrobiona w trakcie zapisu może złapać bazę w połowie transakcji.
    Skrypt ostrzega, gdy wykryje działający serwer.

.EXAMPLE
    .\scripts\backup.ps1
    .\scripts\backup.ps1 -Destination D:\kopie -Keep 10
#>
[CmdletBinding()]
param(
    # Katalog źródłowy z danymi aplikacji.
    [string]$DataDir = (Join-Path $PSScriptRoot '..\data'),

    # Gdzie składować kopie.
    [string]$Destination = (Join-Path $PSScriptRoot '..\backups'),

    # Ile ostatnich kopii zachować (0 = wszystkie).
    [int]$Keep = 7
)

$ErrorActionPreference = 'Stop'

$DataDir = [IO.Path]::GetFullPath($DataDir)
$Destination = [IO.Path]::GetFullPath($Destination)

if (-not (Test-Path -LiteralPath $DataDir)) {
    throw "Nie ma katalogu z danymi: $DataDir"
}

# Ostrzeżenie, gdy serwer prawdopodobnie pracuje.
$running = Get-Process -Name 'designfolio' -ErrorAction SilentlyContinue
if ($running) {
    Write-Warning 'Serwer DesignFolio wygląda na uruchomiony.'
    Write-Warning 'Kopia będzie spójna tylko wtedy, gdy w tej chwili nic nie zapisuje.'
    Write-Warning 'Dla pewności zatrzymaj go najpierw (docker compose stop).'
}

$stamp  = Get-Date -Format 'yyyy-MM-dd_HHmmss'
$target = Join-Path $Destination "data_$stamp"

New-Item -ItemType Directory -Force -Path $target | Out-Null

Write-Host "Kopiuję $DataDir -> $target"

# Kopiujemy ZAWARTOŚĆ katalogu, nie sam katalog. Bez gwiazdki Copy-Item
# zagnieżdża źródło w celu (backup/data_.../data/...), a kopia ma mieć
# dokładnie taką strukturę jak oryginał — inaczej odtwarzanie jest zgadywanką.
Copy-Item -Path (Join-Path $DataDir '*') -Destination $target -Recurse -Force

$files = Get-ChildItem -LiteralPath $target -Recurse -File
$bytes = ($files | Measure-Object -Property Length -Sum).Sum
$mb    = [math]::Round($bytes / 1MB, 1)

Write-Host "Gotowe: $($files.Count) plików, $mb MB"

# Sprzątanie starych kopii.
if ($Keep -gt 0) {
    $old = Get-ChildItem -LiteralPath $Destination -Directory |
        Where-Object { $_.Name -like 'data_*' } |
        Sort-Object Name -Descending |
        Select-Object -Skip $Keep

    foreach ($dir in $old) {
        Write-Host "Usuwam starą kopię: $($dir.Name)"
        Remove-Item -LiteralPath $dir.FullName -Recurse -Force
    }
}
