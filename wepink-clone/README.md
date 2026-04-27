# 🚀 Wepink Clone - Fullstack Project

Este projeto é um clone funcional e arquitetado do sistema de e-commerce da **Wepink**, focado em robustez, consistência eventual e alta performance.

## 🎯 Inspiração: Wepink
O projeto simula o fluxo crítico de um e-commerce de alto tráfego (como o da Wepink em épocas de lançamento/promoção):
- **Criação de Pedidos**: Rápida e resiliente.
- **Pagamentos**: Fluxo assíncrono e seguro, garantindo que nenhum pedido seja pago duas vezes (Idempotência).
- **Escalabilidade**: Preparado para suportar picos de acessos via mensageria.

---

## 🏗️ O que foi feito (Step-by-Step)

### 1. Backend (Go - Arquitetura Hexagonal)
- **Domínio**: Entidades de `Order` e `Payment` com regras de negócio isoladas.
- **Persistência**: MySQL com proteção total contra SQL Injection e uso de transações ACID.
- **Idempotência**: Camada dupla de proteção (Redis para cache rápido + MySQL para garantia final) para evitar duplicidade de pagamentos e pedidos.
- **Mensageria**: Integração com RabbitMQ para processamento assíncrono de pagamentos e notificações.
- **Observabilidade**: Logs estruturados em JSON e propagação de `Correlation ID` para rastreio ponta a ponta.

### 2. Frontend (Angular)
- **Consumo de API**: Service centralizado e modelos tipados.
- **Interceptors**: Adição automática de `X-Correlation-ID` em todas as requisições.
- **UI**: Interface para criação de pedidos, consulta de status e processamento de pagamentos simulados.

### 3. Infraestrutura & Segurança
- **Proxy Reverso**: Nginx configurado para centralizar o acesso e eliminar problemas de CORS.
- **Docker Hardening**: 
  - Containers rodando como **Non-Root** (Privilégio Mínimo).
  - Uso de imagens Alpine (leves e seguras).
  - Scanner de vulnerabilidades **Trivy** integrado no processo de build.

---

## 🛠️ Como Acessar

Com o Docker instalado, basta rodar:
```powershell
docker-compose -f infra/docker/docker-compose.yml up -d
```

### URLs Locais:
- **Aplicação (Frontend)**: [http://localhost](http://localhost)
- **API (Health Check)**: [http://localhost/api/health/ready](http://localhost/api/health/ready)
- **RabbitMQ Admin**: [http://localhost:15672](http://localhost:15672) (`guest` / `guest`)

---

## 🧪 Testes de Carga (k6)
Para validar a performance sob estresse:
```powershell
k6 run tests/load/load_test.js
```

---

## 📂 Organização das Pastas
- `/backend`: API em Go (Hexagonal).
- `/frontend`: Aplicação Angular.
- `/infra`: Configurações de Docker, MySQL, Redis e Nginx.
- `/tests`: Scripts de teste de carga e integração.
