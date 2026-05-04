param(
    [string]$ProjectName = "wepink-clone"
)

Write-Host "🚀 Criando estrutura do projeto: $ProjectName" -ForegroundColor Green

# Criar pasta raiz

New-Item -ItemType Directory -Path $ProjectName -Force | Out-Null
Set-Location $ProjectName

# =========================

# BACKEND (Go - Hexagonal)

# =========================

Write-Host "📦 Criando backend (hexagonal)..." -ForegroundColor Yellow

$backendPaths = @(
    "backend/internal/domain",
    "backend/internal/usecase",
    "backend/internal/ports",
    "backend/internal/adapters/http",
    "backend/internal/adapters/mysql",
    "backend/internal/adapters/redis",
    "backend/internal/adapters/rabbitmq",
    "backend/cmd/api",
    "backend/tests/unit",
    "backend/tests/integration",
    "backend/tests/mocks"
)

$backendPaths | ForEach-Object {
    New-Item -ItemType Directory -Path $_ -Force | Out-Null
}

New-Item -ItemType File -Path "backend/go.mod" -Force | Out-Null
New-Item -ItemType File -Path "backend/cmd/api/main.go" -Force | Out-Null

# =========================

# FRONTEND (Angular)

# =========================

Write-Host "🅰️ Criando frontend..." -ForegroundColor Yellow

$frontendPaths = @(
    "frontend/src/app",
    "frontend/src/assets",
    "frontend/src/environments",
    "frontend/tests/unit",
    "frontend/tests/integration"
)

$frontendPaths | ForEach-Object {
    New-Item -ItemType Directory -Path $_ -Force | Out-Null
}

New-Item -ItemType File -Path "frontend/angular.json" -Force | Out-Null
New-Item -ItemType File -Path "frontend/package.json" -Force | Out-Null

# =========================

# INFRA (Docker + serviços)

# =========================

Write-Host "🐳 Criando infra..." -ForegroundColor Yellow

$infraPaths = @(
    "infra/docker",
    "infra/mysql",
    "infra/redis",
    "infra/rabbitmq"
)

$infraPaths | ForEach-Object {
    New-Item -ItemType Directory -Path $_ -Force | Out-Null
}

New-Item -ItemType File -Path "infra/docker/docker-compose.yml" -Force | Out-Null

# =========================

# CONFIG

# =========================

Write-Host "⚙️ Criando configs..." -ForegroundColor Yellow

New-Item -ItemType Directory -Path "config" -Force | Out-Null
New-Item -ItemType File -Path "config/.env.example" -Force | Out-Null

# =========================

# SCRIPTS

# =========================

Write-Host "📜 Criando scripts auxiliares..." -ForegroundColor Yellow

New-Item -ItemType Directory -Path "scripts" -Force | Out-Null
New-Item -ItemType File -Path "scripts/setup.sh" -Force | Out-Null
New-Item -ItemType File -Path "scripts/test.sh" -Force | Out-Null

# =========================

# DOCS

# =========================

Write-Host "📚 Criando documentação..." -ForegroundColor Yellow

New-Item -ItemType Directory -Path "docs" -Force | Out-Null
New-Item -ItemType File -Path "docs/architecture.md" -Force | Out-Null
New-Item -ItemType File -Path "README.md" -Force | Out-Null

# =========================

# DOCKERFILES

# =========================

Write-Host "📦 Criando Dockerfiles..." -ForegroundColor Yellow

New-Item -ItemType File -Path "backend/Dockerfile" -Force | Out-Null
New-Item -ItemType File -Path "frontend/Dockerfile" -Force | Out-Null

# =========================

# FINAL

# =========================

Write-Host ""
Write-Host "✅ Estrutura criada com sucesso!" -ForegroundColor Green
Write-Host ""
Write-Host "📁 Organização:" -ForegroundColor Cyan
Write-Host " - backend (Go + Hexagonal)"
Write-Host " - frontend (Angular)"
Write-Host " - infra (Docker, MySQL, Redis, RabbitMQ)"
Write-Host " - tests (unit, integration, mocks)"
Write-Host ""
