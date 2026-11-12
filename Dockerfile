# Budowanie w trzech etapach: frontend → binarka Go → chudy obraz produkcyjny.
# Efekt końcowy to jeden plik wykonywalny ze wbudowanym frontendem
# plus katalog /app/data podpięty jako wolumen.

# --- 1. Frontend -------------------------------------------------------------
FROM node:24-alpine AS web

WORKDIR /app/web

# Najpierw same manifesty: dopóki się nie zmienią, Docker użyje cache'a
# i pominie ponowne pobieranie zależności przy każdej zmianie kodu.
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-fund --no-audit

COPY web/ ./
RUN npm run build

# --- 2. Serwer ---------------------------------------------------------------
FROM golang:1.26-alpine AS build

ARG VERSION=docker

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY web/embed.go ./web/
COPY --from=web /app/web/dist ./web/dist

# CGO_ENABLED=0 daje statyczną binarkę bez zależności od libc — dlatego
# w projekcie jest modernc.org/sqlite (SQLite przepisany na Go), a nie mattn/go-sqlite3.
# -s -w wycina tablice symboli i debug info, co ścina rozmiar o kilka MB.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/designfolio ./cmd/server

# --- 3. Obraz końcowy --------------------------------------------------------
FROM alpine:3.22

# Nie uruchamiamy serwera jako root. UID jest stały, żeby prawa do plików
# w podpiętym wolumenie były przewidywalne między przebudowami.
RUN adduser -D -u 10001 app

WORKDIR /app
COPY --from=build /out/designfolio /app/designfolio
RUN mkdir -p /app/data && chown -R app:app /app/data

USER app

# Wewnątrz kontenera musimy słuchać na 0.0.0.0 — 127.0.0.1 byłoby widoczne
# tylko z wnętrza kontenera. Ograniczenie dostępu robi mapowanie portu
# w docker-compose.yml (127.0.0.1:8080:8080).
ENV APP_ADDR=0.0.0.0:8080 \
    DATA_DIR=/app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/designfolio"]
