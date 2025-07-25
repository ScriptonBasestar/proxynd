# ProxyND Infrastructure as Code - Terraform Configuration
# AWS EKS 클러스터 및 관련 리소스 정의

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }
  
  backend "s3" {
    bucket = "proxynd-terraform-state"
    key    = "infrastructure/terraform.tfstate"
    region = "us-west-2"
    
    dynamodb_table = "proxynd-terraform-locks"
    encrypt        = true
  }
}

# 데이터 소스
data "aws_availability_zones" "available" {
  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

data "aws_caller_identity" "current" {}

# 로컬 변수
locals {
  cluster_name = "${var.project_name}-${var.environment}"
  
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
    Owner       = var.owner
  }
}

# VPC 모듈
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  
  name = "${local.cluster_name}-vpc"
  cidr = var.vpc_cidr
  
  azs             = data.aws_availability_zones.available.names
  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets
  
  enable_nat_gateway   = true
  enable_vpn_gateway   = false
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  # EKS 요구사항
  public_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                      = "1"
  }
  
  private_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"             = "1"
  }
  
  tags = local.common_tags
}

# EKS 클러스터
module "eks" {
  source = "terraform-aws-modules/eks/aws"
  
  cluster_name    = local.cluster_name
  cluster_version = var.kubernetes_version
  
  vpc_id                         = module.vpc.vpc_id
  subnet_ids                     = module.vpc.private_subnets
  cluster_endpoint_public_access = var.cluster_endpoint_public_access
  
  # EKS 관리형 노드 그룹
  eks_managed_node_groups = {
    main = {
      name           = "${local.cluster_name}-main"
      instance_types = var.node_instance_types
      
      min_size     = var.node_group_min_size
      max_size     = var.node_group_max_size
      desired_size = var.node_group_desired_size
      
      disk_size = var.node_disk_size
      
      labels = {
        Environment = var.environment
        NodeGroup   = "main"
      }
      
      taints = []
      
      tags = local.common_tags
    }
    
    # 프로덕션 전용 노드 그룹
    production = {
      name           = "${local.cluster_name}-production"
      instance_types = var.prod_node_instance_types
      
      min_size     = var.prod_node_group_min_size
      max_size     = var.prod_node_group_max_size
      desired_size = var.prod_node_group_desired_size
      
      disk_size = var.prod_node_disk_size
      
      labels = {
        Environment = "production"
        NodeGroup   = "production"
        WorkloadType = "application"
      }
      
      taints = [
        {
          key    = "workload-type"
          value  = "application"
          effect = "NO_SCHEDULE"
        }
      ]
      
      tags = merge(local.common_tags, {
        NodeGroup = "production"
      })
    }
  }
  
  # OIDC Identity Provider
  cluster_identity_providers = {
    sts = {
      client_id = "sts.amazonaws.com"
    }
  }
  
  tags = local.common_tags
}

# EKS 애드온
resource "aws_eks_addon" "addons" {
  for_each = {
    coredns = {
      version = var.coredns_version
    }
    kube-proxy = {
      version = var.kube_proxy_version
    }
    vpc-cni = {
      version = var.vpc_cni_version
    }
    aws-ebs-csi-driver = {
      version = var.ebs_csi_version
    }
  }
  
  cluster_name      = module.eks.cluster_name
  addon_name        = each.key
  addon_version     = each.value.version
  resolve_conflicts = "OVERWRITE"
  
  depends_on = [module.eks]
}

# IAM 역할 및 정책
resource "aws_iam_role" "proxynd_service_role" {
  name = "${local.cluster_name}-proxynd-service-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Principal = {
          Federated = module.eks.oidc_provider_arn
        }
        Condition = {
          StringEquals = {
            "${replace(module.eks.oidc_provider_arn, "/^(.*provider/)/", "")}:sub": "system:serviceaccount:production:proxynd"
            "${replace(module.eks.oidc_provider_arn, "/^(.*provider/)/", "")}:aud": "sts.amazonaws.com"
          }
        }
      }
    ]
  })
  
  tags = local.common_tags
}

# S3 버킷 (캐시 스토리지)
resource "aws_s3_bucket" "proxynd_cache" {
  bucket = "${local.cluster_name}-cache-storage"
  
  tags = local.common_tags
}

resource "aws_s3_bucket_versioning" "proxynd_cache" {
  bucket = aws_s3_bucket.proxynd_cache.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "proxynd_cache" {
  bucket = aws_s3_bucket.proxynd_cache.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "proxynd_cache" {
  bucket = aws_s3_bucket.proxynd_cache.id
  
  rule {
    id     = "cache_cleanup"
    status = "Enabled"
    
    expiration {
      days = var.cache_retention_days
    }
    
    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# RDS 인스턴스 (메타데이터 스토리지)
resource "aws_db_subnet_group" "proxynd" {
  name       = "${local.cluster_name}-db-subnet-group"
  subnet_ids = module.vpc.private_subnets
  
  tags = local.common_tags
}

resource "aws_security_group" "rds" {
  name        = "${local.cluster_name}-rds-sg"
  description = "Security group for ProxyND RDS instance"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [module.eks.cluster_security_group_id]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = local.common_tags
}

resource "aws_db_instance" "proxynd" {
  identifier = "${local.cluster_name}-db"
  
  engine         = "postgres"
  engine_version = var.postgres_version
  instance_class = var.db_instance_class
  
  allocated_storage     = var.db_allocated_storage
  max_allocated_storage = var.db_max_allocated_storage
  storage_encrypted     = true
  
  db_name  = "proxynd"
  username = var.db_username
  password = var.db_password
  
  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.proxynd.name
  
  backup_retention_period = var.db_backup_retention_period
  backup_window          = var.db_backup_window
  maintenance_window     = var.db_maintenance_window
  
  skip_final_snapshot = var.environment != "production"
  final_snapshot_identifier = var.environment == "production" ? "${local.cluster_name}-final-snapshot" : null
  
  tags = local.common_tags
}

# ElastiCache Redis 클러스터
resource "aws_elasticache_subnet_group" "proxynd" {
  name       = "${local.cluster_name}-cache-subnet"
  subnet_ids = module.vpc.private_subnets
  
  tags = local.common_tags
}

resource "aws_security_group" "redis" {
  name        = "${local.cluster_name}-redis-sg"
  description = "Security group for ProxyND Redis cluster"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [module.eks.cluster_security_group_id]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = local.common_tags
}

resource "aws_elasticache_replication_group" "proxynd" {
  replication_group_id       = "${local.cluster_name}-redis"
  description                = "ProxyND Redis cluster"
  
  port                = 6379
  parameter_group_name = "default.redis7"
  engine_version      = var.redis_version
  node_type           = var.redis_node_type
  
  num_cache_clusters = var.redis_num_cache_nodes
  
  subnet_group_name  = aws_elasticache_subnet_group.proxynd.name
  security_group_ids = [aws_security_group.redis.id]
  
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  
  tags = local.common_tags
}

# Application Load Balancer
resource "aws_security_group" "alb" {
  name        = "${local.cluster_name}-alb-sg"
  description = "Security group for ProxyND ALB"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = local.common_tags
}

# CloudWatch 로그 그룹
resource "aws_cloudwatch_log_group" "proxynd" {
  name              = "/aws/eks/${local.cluster_name}/proxynd"
  retention_in_days = var.log_retention_days
  
  tags = local.common_tags
}

# SSM 파라미터 (설정 관리)
resource "aws_ssm_parameter" "db_connection" {
  name  = "/${var.project_name}/${var.environment}/database/connection_string"
  type  = "SecureString"
  value = "postgresql://${var.db_username}:${var.db_password}@${aws_db_instance.proxynd.endpoint}:5432/proxynd"
  
  tags = local.common_tags
}

resource "aws_ssm_parameter" "redis_connection" {
  name  = "/${var.project_name}/${var.environment}/redis/connection_string"
  type  = "SecureString"
  value = aws_elasticache_replication_group.proxynd.configuration_endpoint_address
  
  tags = local.common_tags
}