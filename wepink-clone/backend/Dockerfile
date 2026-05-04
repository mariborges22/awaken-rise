# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Instalar dependências necessárias para o build
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código e compilar
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/api/main.go

# Stage 2: Runtime
FROM alpine:3.19

# Instalar certificados (necessário para chamadas HTTPS de APIs externas)
RUN apk --no-cache add ca-certificates

# Criar usuário não-root por segurança
RUN adduser -D -g '' appuser
USER appuser

WORKDIR /app

# Copiar binário do estágio de build
COPY --from=builder /app/main .

EXPOSE 8080

# Metadata
LABEL maintainer="Wepink Clone Team"
LABEL version="1.0.0"

CMD ["./main"]
