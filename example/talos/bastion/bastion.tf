terraform {
  required_providers {
    outscale = {
      source = "outscale/outscale"
      version = ">= 0.8.2"
    }
  }
}

provider "outscale" {
  region = "eu-west-2"
}

variable region {
  type = string
  default = "eu-west-2"
}

variable az {
  type = string
  default = "a"
}

variable net_id {
  type = string
}

variable control_plane_subnet_id {
  type = string
}

variable wg_listen_port {
  type = number
  default = 18513
}

data external local_wg_keypair {
  program = ["./gen-wg-keypair.sh"]
}

data external remote_wg_keypair {
  program = ["./gen-wg-keypair.sh"]
}

output wg_config {
	value = <<EOF
[Interface]
PrivateKey = ${data.external.local_wg_keypair.result.privkey}
Address = 10.0.5.2/24

[Peer]
PublicKey = ${data.external.remote_wg_keypair.result.pubkey}
AllowedIPs = 10.0.0.0/16
Endpoint = ${outscale_public_ip.bastion.public_ip}:${var.wg_listen_port}
PersistentKeepalive = 25
EOF
}

resource "outscale_security_group" "bastion" {
  net_id              = var.net_id
  security_group_name = "tmp-bastion-sg"
  tags {
    key   = "Name"
    value = "tmp-bastion-test"
  }
}

resource "outscale_security_group_rule" "icmp" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.bastion.security_group_id
  ip_protocol       = "icmp"
  from_port_range   = "-1"
  to_port_range     = "-1"
  ip_range          = "0.0.0.0/0"
}

resource "outscale_security_group_rule" "ssh" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.bastion.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = "22"
  to_port_range     = "22"
  ip_range          = "0.0.0.0/0"
}

resource "outscale_security_group_rule" "wireguard" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.bastion.security_group_id
  ip_protocol       = "udp"
  from_port_range   = var.wg_listen_port
  to_port_range     = var.wg_listen_port
  ip_range          = "0.0.0.0/0"
}

resource "outscale_vm" "bastion" {
  image_id                 = "ami-cd8d714e" // Ubuntu 22.04
  vm_type                  = "t1.micro"
  placement_subregion_name = "${var.region}${var.az}"
  security_group_ids       = [outscale_security_group.bastion.security_group_id]
  subnet_id                = var.control_plane_subnet_id
  user_data                = base64encode(templatefile("./user-data.sh.tftpl", {
    wg_privkey         = data.external.remote_wg_keypair.result.privkey
    wg_peer_pubkey     = data.external.local_wg_keypair.result.pubkey
    wg_listen_port     = var.wg_listen_port
  }))

  tags {
    key   = "Name"
    value = "tmp-bastion-test"
  }
}

resource "outscale_public_ip" "bastion" {
  tags {
    key   = "Name"
    value = "tmp-bastion-test"
  }
}

resource "outscale_public_ip_link" "dns" {
  vm_id     = outscale_vm.bastion.vm_id
  public_ip = outscale_public_ip.bastion.public_ip
}
