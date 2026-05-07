#!/bin/bash
set -euo pipefail

COD_EMP=$1
DB_NAME="papp_${COD_EMP}"
LOG_FILE="migracao_${DB_NAME}.log"

# --- OLD ---
OLD_HOST="10.20.0.65"
OLD_USER="gerencio_adm"
OLD_PASS="SUA_SENHA"

# --- NEW (DO) ---
NEW_HOST="facilitaponto-db-do-user-34380799-0.l.db.ondigitalocean.com"
NEW_USER="doadmin"
NEW_PASS="SUA_SENHA"
NEW_PORT="25060"

MAX_RETRY=3

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log "========================================="
log "Iniciando migração: ${DB_NAME}"
log "========================================="

# =========================
# PASSO 0 - SKIP POR PRD_HOST
# =========================
log "[0/5] Verificando prd_host..."

PRD_HOST=$(mysql -h $OLD_HOST -u $OLD_USER -p$OLD_PASS -N -e "
SELECT prd_host
FROM pontoapp_dbempresa.empresa
WHERE chave = '${COD_EMP}'
LIMIT 1;
")

if [[ "$PRD_HOST" == *"$NEW_HOST"* ]]; then
    log "⏭️ Já migrado (${PRD_HOST}). Pulando."
    exit 0
fi

# =========================
# PASSO 0.1 - JÁ TEM DADOS?
# =========================
TABLES_EXIST=$(mysql -h $NEW_HOST -P $NEW_PORT -u $NEW_USER -p$NEW_PASS -N -e "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = '${DB_NAME}';
")

if [ "${TABLES_EXIST:-0}" -gt 0 ]; then
    log "⚠️ Banco já tem ${TABLES_EXIST} tabelas. Atualizando prd_host..."

    # OLD
    mysql -h $OLD_HOST -u $OLD_USER -p$OLD_PASS -e "
    UPDATE pontoapp_dbempresa.empresa
    SET prd_host = '${NEW_HOST}:${NEW_PORT}',
        prd_user = '${NEW_USER}',
        prd_pwd = 'SUA_SENHA'
    WHERE chave = '${COD_EMP}';
    "

    # NEW
    mysql -h $NEW_HOST -P $NEW_PORT -u $NEW_USER -p$NEW_PASS -e "
    UPDATE pontoapp_dbempresa.empresa
    SET prd_host = '${NEW_HOST}:${NEW_PORT}',
        prd_user = '${NEW_USER}',
        prd_pwd = 'SUA_SENHA'
    WHERE chave = '${COD_EMP}';
    "

    exit 0
fi

# =========================
# FUNÇÃO DUMP COM RETRY
# =========================
dump_database() {
    for i in $(seq 1 $MAX_RETRY); do
        log "[1/5] Dump tentativa $i..."

        mysqldump -h $OLD_HOST -u $OLD_USER -p$OLD_PASS \
            --single-transaction \
            --quick \
            --set-gtid-purged=OFF \
            --add-drop-table \
            --default-character-set=utf8mb4 \
            --connect-timeout=10 \
            $DB_NAME 2>>"$LOG_FILE" | gzip > "${DB_NAME}.sql.gz"

        if [ $? -eq 0 ]; then
            return 0
        fi

        log "⚠️ Dump falhou, retry em 5s..."
        sleep 5
    done

    return 1
}

# =========================
# PASSO 1 - DUMP
# =========================
dump_database || {
    log "❌ ERRO: dump falhou após $MAX_RETRY tentativas"
    exit 1
}

# valida arquivo
if [ ! -s "${DB_NAME}.sql.gz" ]; then
    log "❌ Dump vazio!"
    exit 1
fi

# valida conteúdo (mais inteligente)
LINES=$(gunzip -c "${DB_NAME}.sql.gz" | wc -l)

log "Linhas no dump: $LINES"

if [ "$LINES" -lt 50 ]; then
    log "❌ Dump suspeito (muito pequeno)"
    exit 1
fi

log "✅ Dump válido"

# =========================
# PASSO 2 - CRIAR DB
# =========================
log "[2/5] Criando banco..."

mysql -h $NEW_HOST -P $NEW_PORT -u $NEW_USER -p$NEW_PASS \
    -e "CREATE DATABASE IF NOT EXISTS ${DB_NAME};"

log "✅ Banco criado"

# =========================
# PASSO 3 - IMPORT
# =========================
log "[3/5] Importando..."

gunzip < "${DB_NAME}.sql.gz" | mysql \
    -h $NEW_HOST -P $NEW_PORT \
    -u $NEW_USER -p$NEW_PASS \
    $DB_NAME 2>>"$LOG_FILE"

if [ $? -ne 0 ]; then
    log "❌ ERRO na importação"
    exit 1
fi

log "✅ Import concluído"

# =========================
# PASSO 4 - VALIDAR IMPORT
# =========================
log "[4/5] Validando..."

TABLES_COUNT=$(mysql -h $NEW_HOST -P $NEW_PORT -u $NEW_USER -p$NEW_PASS -N -e "
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = '${DB_NAME}';
")

log "Tabelas encontradas: $TABLES_COUNT"

if [ "${TABLES_COUNT:-0}" -eq 0 ]; then
    log "❌ ERRO: banco vazio após import"
    exit 1
fi

log "✅ Validação OK"

# =========================
# PASSO 5 - UPDATE OLD + NEW
# =========================
log "[5/5] Atualizando prd_host..."

# OLD
mysql -h $OLD_HOST -u $OLD_USER -p$OLD_PASS -e "
UPDATE pontoapp_dbempresa.empresa
SET prd_host = '${NEW_HOST}:${NEW_PORT}',
    prd_user = '${NEW_USER}',
    prd_pwd = 'SUA_SENHA'
WHERE chave = '${COD_EMP}';
"

# NEW
mysql -h $NEW_HOST -P $NEW_PORT -u $NEW_USER -p$NEW_PASS -e "
UPDATE pontoapp_dbempresa.empresa
SET prd_host = '${NEW_HOST}:${NEW_PORT}',
    prd_user = '${NEW_USER}',
    prd_pwd = 'SUA_SENHA'
WHERE chave = '${COD_EMP}';
"

log "✅ prd_host atualizado"

# =========================
# LIMPEZA
# =========================
rm -f "${DB_NAME}.sql.gz"

log "========================================="
log "✅ SUCESSO: ${DB_NAME}"
log "========================================="