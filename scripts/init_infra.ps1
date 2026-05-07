$BaseDir = "infra"

Write-Host "🚀 Criando estrutura de pastas para Infraestrutura..." -ForegroundColor Cyan

# Terraform: Modules e Environments
$TfPaths = @(
    "$BaseDir/terraform/modules/networking",
    "$BaseDir/terraform/modules/compute",
    "$BaseDir/terraform/modules/storage",
    "$BaseDir/terraform/modules/oke",
    "$BaseDir/terraform/environments/staging",
    "$BaseDir/terraform/environments/prod"
)

# Ansible: Roles e Playbooks
$AnsiblePaths = @(
    "$BaseDir/ansible/inventory/staging",
    "$BaseDir/ansible/inventory/prod",
    "$BaseDir/ansible/roles/common/tasks",
    "$BaseDir/ansible/roles/docker/tasks",
    "$BaseDir/ansible/roles/wepink_app/tasks",
    "$BaseDir/ansible/playbooks",
    "$BaseDir/ansible/group_vars"
)

# Criar Pastas
foreach ($Path in ($TfPaths + $AnsiblePaths)) {
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

# Criar arquivos placeholders
New-Item -ItemType File -Path "$BaseDir/terraform/environments/staging/main.tf" -Force | Out-Null
New-Item -ItemType File -Path "$BaseDir/terraform/environments/prod/main.tf" -Force | Out-Null
New-Item -ItemType File -Path "$BaseDir/ansible/site.yml" -Force | Out-Null

Write-Host "✅ Estrutura criada em ./$BaseDir" -ForegroundColor Green
Write-Host "💡 Dica: O seu PC vai agradecer por rodar o Docker na nuvem!" -ForegroundColor Yellow
