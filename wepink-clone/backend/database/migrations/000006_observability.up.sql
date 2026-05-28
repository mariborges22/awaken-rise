-- 000006_observability.up.sql
-- Camada de observabilidade, métricas e rotinas de limpeza automática para produção

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. TABELA DE MÉTRICAS DE API
-- Populada pelo MetricsMiddleware do Go de forma assíncrona.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS api_metrics (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    method       VARCHAR(10)  NOT NULL,
    path         VARCHAR(255) NOT NULL,
    status_code  SMALLINT     NOT NULL,
    latency_ms   INT          NOT NULL,
    tenant_id    VARCHAR(100) NULL,
    created_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_metrics_path        (path),
    INDEX idx_metrics_status      (status_code),
    INDEX idx_metrics_created_at  (created_at),
    INDEX idx_metrics_tenant      (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. TABELA DE LOG DE ERROS INTERNOS (500)
-- Separada das métricas gerais para facilitar alertas e triagem em produção.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS error_logs (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY,
    correlation_id   VARCHAR(100) NOT NULL,
    tenant_id        VARCHAR(100) NULL,
    method           VARCHAR(10)  NOT NULL,
    path             VARCHAR(255) NOT NULL,
    error_message    TEXT         NOT NULL,
    created_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_errors_correlation  (correlation_id),
    INDEX idx_errors_tenant       (tenant_id),
    INDEX idx_errors_created_at   (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. TABELA DE ARQUIVAMENTO DE PEDIDOS
-- Pedidos CANCELLED com mais de 60 dias são movidos para cá pelo EVENT abaixo.
-- Sem índices de busca pesados — apenas para auditoria e compliance.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS orders_archive (
    id          VARCHAR(100) PRIMARY KEY,
    tenant_id   VARCHAR(100) NOT NULL,
    status      VARCHAR(50)  NOT NULL,
    total       DECIMAL(10, 2) NOT NULL,
    archived_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    created_at  TIMESTAMP    NULL,
    INDEX idx_archive_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. MYSQL EVENTS — ROTINAS DE LIMPEZA AUTOMÁTICA
-- Requer event_scheduler=ON (configurado no docker-compose: --event-scheduler=ON)
-- ─────────────────────────────────────────────────────────────────────────────

-- Limpeza de eventos processados (anti-duplicidade): retém apenas os 30 dias mais recentes.
-- Evita crescimento indefinido da tabela processed_events, que não tem limpeza hoje.
CREATE EVENT IF NOT EXISTS cleanup_processed_events
    ON SCHEDULE EVERY 1 DAY
    STARTS (TIMESTAMP(CURRENT_DATE) + INTERVAL '03:00' HOUR_MINUTE)
    ON COMPLETION PRESERVE
    ENABLE
    DO
        DELETE FROM processed_events
        WHERE processed_at < NOW() - INTERVAL 30 DAY;

-- Limpeza de métricas antigas: retém apenas os 90 dias mais recentes.
CREATE EVENT IF NOT EXISTS cleanup_old_metrics
    ON SCHEDULE EVERY 1 WEEK
    STARTS (TIMESTAMP(CURRENT_DATE) + INTERVAL '04:00' HOUR_MINUTE)
    ON COMPLETION PRESERVE
    ENABLE
    DO
        DELETE FROM api_metrics
        WHERE created_at < NOW() - INTERVAL 90 DAY;

-- Limpeza de logs de erro: retém apenas os 180 dias mais recentes (6 meses para auditoria).
CREATE EVENT IF NOT EXISTS cleanup_old_error_logs
    ON SCHEDULE EVERY 1 WEEK
    STARTS (TIMESTAMP(CURRENT_DATE) + INTERVAL '04:30' HOUR_MINUTE)
    ON COMPLETION PRESERVE
    ENABLE
    DO
        DELETE FROM error_logs
        WHERE created_at < NOW() - INTERVAL 180 DAY;

-- Arquivamento de pedidos cancelados antigos.
-- Move pedidos CANCELLED com mais de 60 dias da tabela hot para orders_archive.
CREATE EVENT IF NOT EXISTS archive_old_cancelled_orders
    ON SCHEDULE EVERY 1 DAY
    STARTS (TIMESTAMP(CURRENT_DATE) + INTERVAL '02:00' HOUR_MINUTE)
    ON COMPLETION PRESERVE
    ENABLE
    DO BEGIN
        INSERT IGNORE INTO orders_archive (id, tenant_id, status, total, created_at)
            SELECT id, tenant_id, status, total, created_at
            FROM orders
            WHERE status = 'CANCELLED'
              AND updated_at < NOW() - INTERVAL 60 DAY;

        DELETE FROM orders
        WHERE status = 'CANCELLED'
          AND updated_at < NOW() - INTERVAL 60 DAY;
    END;

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. VIEWS DE OBSERVABILIDADE
-- Consultas prontas para análise de performance sem escrever SQL ad-hoc.
-- ─────────────────────────────────────────────────────────────────────────────

-- Performance por rota e dia: avg, P95 e máximo de latência
CREATE OR REPLACE VIEW v_api_performance AS
    SELECT
        DATE(created_at)                                 AS day,
        method,
        path,
        status_code,
        COUNT(*)                                         AS total_requests,
        ROUND(AVG(latency_ms), 2)                        AS avg_latency_ms,
        MAX(latency_ms)                                  AS max_latency_ms,
        -- P95 aproximado via subconsulta ordenada
        (
            SELECT latency_ms FROM api_metrics m2
            WHERE m2.path = m.path AND m2.method = m.method
              AND DATE(m2.created_at) = DATE(m.created_at)
            ORDER BY latency_ms
            LIMIT 1 OFFSET FLOOR(0.95 * COUNT(*))
        )                                                AS p95_latency_ms
    FROM api_metrics m
    GROUP BY DATE(created_at), method, path, status_code;

-- Erros internos agrupados nas últimas 24h para triagem rápida
CREATE OR REPLACE VIEW v_error_summary AS
    SELECT
        path,
        method,
        LEFT(error_message, 100)    AS error_snippet,
        COUNT(*)                    AS occurrences,
        MAX(created_at)             AS last_seen,
        GROUP_CONCAT(DISTINCT tenant_id ORDER BY tenant_id SEPARATOR ', ')
                                    AS affected_tenants
    FROM error_logs
    WHERE created_at >= NOW() - INTERVAL 24 HOUR
    GROUP BY path, method, LEFT(error_message, 100)
    ORDER BY occurrences DESC;

-- Throughput de pedidos por hora (útil para monitorar picos em Black Friday)
CREATE OR REPLACE VIEW v_order_throughput AS
    SELECT
        DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00')   AS hour_bucket,
        tenant_id,
        status,
        COUNT(*)                                        AS order_count,
        ROUND(SUM(total), 2)                            AS total_revenue
    FROM orders
    GROUP BY DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00'), tenant_id, status
    ORDER BY hour_bucket DESC;
