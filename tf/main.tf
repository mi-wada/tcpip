terraform {
  required_version = "1.14.3"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.27.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.1.0"
    }
    github = {
      source  = "integrations/github"
      version = "6.9.0"
    }
  }
}

terraform {
  backend "s3" {
    bucket       = "tf-state-03d7dc79-74e1-4100-b66e-55d830971e7b"
    key          = "terraform.5de212af-3fa8-4015-b9b3-a50449cd99da.tfstate"
    region       = "ap-northeast-1"
    use_lockfile = true
  }
}

locals {
  project = "tcpip"
}

provider "aws" {
  region = "ap-northeast-1"
  default_tags {
    tags = {
      Project = local.project
    }
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.10.0.0/16"
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.10.1.0/24"
  map_public_ip_on_launch = true
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

resource "aws_instance" "main" {
  ami                         = "ami-0aec5ae807cea9ce0" # Ubuntu Server 24.04 LTS (HVM), SSD Volume Type
  instance_type               = "t3.small"
  subnet_id                   = aws_subnet.public.id
  vpc_security_group_ids      = [aws_security_group.main.id]
  key_name                    = aws_key_pair.main.key_name
  user_data_replace_on_change = true
  user_data                   = <<-EOF
    #!/usr/bin/env bash
    set -euxo pipefail

    export DEBIAN_FRONTEND=noninteractive

    apt-get update
    apt-get install -y --no-install-recommends \
      git \
      build-essential \
      gdb \
      strace \
      tcpdump \
      iproute2 \
      iputils-ping \
      ca-certificates \
      curl \
      openssh-client

    USERNAME="ubuntu"
    HOME_DIR="/home/$${USERNAME}"
    SSH_DIR="$${HOME_DIR}/.ssh"
    install -d -m 700 -o "$${USERNAME}" -g "$${USERNAME}" "$${SSH_DIR}"
    cat > "$${SSH_DIR}/id_ed25519_github_deploy" <<'KEY'
    ${tls_private_key.github_deploy.private_key_openssh}
    KEY

    chown "$${USERNAME}:$${USERNAME}" "$${SSH_DIR}/id_ed25519_github_deploy"
    chmod 600 "$${SSH_DIR}/id_ed25519_github_deploy"

    sudo -u "$${USERNAME}" ssh-keyscan -t rsa,ed25519 github.com >> "$${SSH_DIR}/known_hosts"
    chown "$${USERNAME}:$${USERNAME}" "$${SSH_DIR}/known_hosts"
    chmod 644 "$${SSH_DIR}/known_hosts"

    cat > "$${SSH_DIR}/config" <<'CFG'
    Host github.com
      HostName github.com
      User git
      IdentityFile ~/.ssh/id_ed25519_github_deploy
      IdentitiesOnly yes
      StrictHostKeyChecking yes
    CFG

    chown "$${USERNAME}:$${USERNAME}" "$${SSH_DIR}/config"
    chmod 600 "$${SSH_DIR}/config"

    sudo -u "$USERNAME" -H bash <<'EOS'
      set -euxo pipefail

      cd "$HOME"

      git clone git@github.com:mi-wada/tcpip.git

      git config --global user.email '49638956+mi-wada@users.noreply.github.com'
      git config --global user.name 'Mitsuaki Wada'
    EOS
  EOF
}

resource "aws_security_group" "main" {
  name   = local.project
  vpc_id = aws_vpc.main.id
}

resource "aws_security_group_rule" "ingress_ssh" {
  security_group_id = aws_security_group.main.id
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["152.165.52.145/32"]
}

resource "aws_security_group_rule" "egress_all" {
  security_group_id = aws_security_group.main.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_key_pair" "main" {
  key_name   = "${local.project}-login"
  public_key = file("~/.ssh/id_ed25519.pub")
}

output "ssh_command" {
  value = "ssh -i ~/.ssh/id_ed25519 ubuntu@${aws_instance.main.public_ip}"
}

output "public_ip" {
  value = aws_instance.main.public_ip
}

provider "github" {
  owner = "mi-wada"
}

resource "github_repository_deploy_key" "main" {
  repository = "tcpip"
  title      = "${local.project}-deploy-key"
  key        = tls_private_key.github_deploy.public_key_openssh
  read_only  = false # write enabled
}

resource "tls_private_key" "github_deploy" {
  algorithm = "ED25519"
}
