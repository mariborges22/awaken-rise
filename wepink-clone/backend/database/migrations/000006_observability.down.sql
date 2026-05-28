-- 000006_observability.down.sql
DROP VIEW  IF EXISTS v_order_throughput;
DROP VIEW  IF EXISTS v_error_summary;
DROP VIEW  IF EXISTS v_api_performance;
DROP EVENT IF EXISTS archive_old_cancelled_orders;
DROP EVENT IF EXISTS cleanup_old_error_logs;
DROP EVENT IF EXISTS cleanup_old_metrics;
DROP EVENT IF EXISTS cleanup_processed_events;
DROP TABLE IF EXISTS orders_archive;
DROP TABLE IF EXISTS error_logs;
DROP TABLE IF EXISTS api_metrics;
