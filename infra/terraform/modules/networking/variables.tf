# infra/terraform/modules/networking/variables.tf

variable "compartment_id" {
  description = "OCID do Compartment"
}

variable "project_name" {
  description = "Nome do projeto para prefixo de recursos"
}

variable "vcn_cidr" {
  default = "10.0.0.0/16"
}

variable "public_subnet_cidr" {
  default = "10.0.1.0/24"
}
