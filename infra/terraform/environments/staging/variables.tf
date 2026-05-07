# infra/terraform/environments/staging/variables.tf

variable "region" {
  default = "nyc3" # New York 3 (Pode ser alterado para 'fra1' ou 'ams3' se preferir Europa)
}

variable "ssh_key_fingerprint" {
  description = "Fingerprint da sua chave SSH pública cadastrada na DigitalOcean"
}
