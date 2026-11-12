#!/usr/bin/env bash
#
# Kopia zapasowa katalogu data/ (baza + pliki).
#
# Backup to po prostu kopia katalogu — nic więcej. Baza SQLite jest jednym
# plikiem, pliki leżą obok, więc odtworzenie polega na skopiowaniu tego z powrotem.
#
# WAŻNE: przy trybie WAL baza to trzy pliki (.db, .db-wal, .db-shm).
# Skrypt kopiuje wszystkie, ale dla gwarancji spójności zatrzymaj wcześniej
# aplikację:
#
#     docker compose stop
#
# Użycie:
#   ./scripts/backup.sh                       # data/ -> backups/
#   DEST=/mnt/dysk/kopie KEEP=10 ./scripts/backup.sh

# -e: przerwij przy błędzie, -u: błąd przy nieustawionej zmiennej,
# -o pipefail: błąd w potoku nie ginie.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

DATA_DIR="${DATA_DIR:-$ROOT_DIR/data}"
DEST="${DEST:-$ROOT_DIR/backups}"
KEEP="${KEEP:-7}"

if [ ! -d "$DATA_DIR" ]; then
    echo "Nie ma katalogu z danymi: $DATA_DIR" >&2
    exit 1
fi

if pgrep -x designfolio >/dev/null 2>&1; then
    echo "UWAGA: serwer DesignFolio wygląda na uruchomiony." >&2
    echo "       Kopia będzie spójna tylko wtedy, gdy nic w tej chwili nie zapisuje." >&2
    echo "       Dla pewności zatrzymaj go najpierw: docker compose stop" >&2
fi

STAMP="$(date +%Y-%m-%d_%H%M%S)"
TARGET="$DEST/data_$STAMP"

mkdir -p "$TARGET"

echo "Kopiuję $DATA_DIR -> $TARGET"
cp -a "$DATA_DIR/." "$TARGET/"

COUNT="$(find "$TARGET" -type f | wc -l | tr -d ' ')"
SIZE="$(du -sh "$TARGET" | cut -f1)"
echo "Gotowe: $COUNT plików, $SIZE"

# Sprzątanie starych kopii. Nazwy zawierają datę, więc sortowanie
# alfabetyczne jest jednocześnie sortowaniem chronologicznym.
if [ "$KEEP" -gt 0 ]; then
    # shellcheck disable=SC2012  # nazwy katalogów są nasze, bez znaków specjalnych
    ls -1d "$DEST"/data_* 2>/dev/null | sort -r | tail -n "+$((KEEP + 1))" | while read -r old; do
        echo "Usuwam starą kopię: $(basename "$old")"
        rm -rf "$old"
    done
fi
