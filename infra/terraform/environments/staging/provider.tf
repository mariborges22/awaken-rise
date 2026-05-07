# infra/terraform/environments/staging/provider.tf

terraform {
  cloud {
    organization = "awakenrise"
    workspaces {
      name = "wepink-staging"
    }
  }

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

variable "do_token" {
  description = "DigitalOcean Personal Access Token"
}

provider "digitalocean" {
  token = var.do_token
}
