#!/bin/bash

# Direitório base da infra
BASE_DIR="infra"

echo "🚀 Criando estrutura de pastas para Infraestrutura..."

# Terraform: Modules e Environments
mkdir -p $BASE_DIR/terraform/modules/networking
mkdir -p $BASE_DIR/terraform/modules/compute
mkdir -p $BASE_DIR/terraform/modules/storage
mkdir -p $BASE_DIR/terraform/modules/oke
mkdir -p $BASE_DIR/terraform/environments/staging
mkdir -p $BASE_DIR/terraform/environments/prod

# Ansible: Roles e Playbooks
mkdir -p $BASE_DIR/ansible/inventory/staging
mkdir -p $BASE_DIR/ansible/inventory/prod
mkdir -p $BASE_DIR/ansible/roles/common/tasks
mkdir -p $BASE_DIR/ansible/roles/docker/tasks
mkdir -p $BASE_DIR/ansible/roles/wepink_app/tasks
mkdir -p $BASE_DIR/ansible/playbooks
mkdir -p $BASE_DIR/ansible/group_vars

# Criando arquivos placeholders para evitar pastas vazias no Git
touch $BASE_DIR/terraform/environments/staging/main.tf
touch $BASE_DIR/terraform/environments/prod/main.tf
touch $BASE_DIR/ansible/site.yml

echo "✅ Estrutura criada em ./$BASE_DIR"
echo "💡 Dica: Agora você pode usar o Terraform para criar sua VM de Dev na Oracle Cloud."
