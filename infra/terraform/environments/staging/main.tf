# infra/terraform/environments/staging/main.tf

resource "digitalocean_vpc" "awakenrise_vpc" {
  name   = "awakenrise-vpc-staging"
  region = var.region
}

# Banco de Dados MySQL Gerenciado
resource "digitalocean_database_cluster" "mysql_staging" {
  name       = "awakenrise-mysql-staging"
  engine     = "mysql"
  version    = "8"
  size       = "db-s-1vcpu-1gb"
  region     = var.region
  node_count = 1
}

resource "digitalocean_database_firewall" "mysql_fw" {
  cluster_id = digitalocean_database_cluster.mysql_staging.id

  rule {
    type  = "droplet"
    value = digitalocean_droplet.wepink_app.id
  }
}


resource "digitalocean_droplet" "wepink_app" {
  image  = "ubuntu-22-04-x64"
  name   = "wepink-app-staging"
  region = var.region
  size   = "s-1vcpu-1gb"
  vpc_uuid = digitalocean_vpc.awakenrise_vpc.id
  ssh_keys = [var.ssh_key_fingerprint]

  tags = ["staging", "wepink"]
}

output "droplet_ip" {
  value = digitalocean_droplet.wepink_app.ipv4_address
}

output "mysql_host" {
  value = digitalocean_database_cluster.mysql_staging.private_host
}

output "mysql_port" {
  value = digitalocean_database_cluster.mysql_staging.port
}

output "mysql_user" {
  value = digitalocean_database_cluster.mysql_staging.user
}

output "mysql_password" {
  value     = digitalocean_database_cluster.mysql_staging.password
  sensitive = true
}

resource "digitalocean_firewall" "wepink_firewall" {
  name = "wepink-firewall-staging"

  droplet_ids = [digitalocean_droplet.wepink_app.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "8080"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "3000"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "9090"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "15672"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}
