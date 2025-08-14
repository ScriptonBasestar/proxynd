# ProxyND Infrastructure Variables
# Terraform 변수 정의

variable "project_name" {
  description = "프로젝트 이름"
  type        = string
  default     = "proxynd"
}

variable "environment" {
  description = "환경 (dev, staging, production)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "owner" {
  description = "리소스 소유자"
  type        = string
  default     = "devops-team"
}

variable "region" {
  description = "AWS 리전"
  type        = string
  default     = "us-west-2"
}

# 네트워크 설정
variable "vpc_cidr" {
  description = "VPC CIDR 블록"
  type        = string
  default     = "10.0.0.0/16"
}

variable "private_subnets" {
  description = "프라이빗 서브넷 CIDR 목록"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "public_subnets" {
  description = "퍼블릭 서브넷 CIDR 목록"
  type        = list(string)
  default     = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
}

# EKS 클러스터 설정
variable "kubernetes_version" {
  description = "Kubernetes 버전"
  type        = string
  default     = "1.28"
}

variable "cluster_endpoint_public_access" {
  description = "클러스터 엔드포인트 퍼블릭 액세스 허용"
  type        = bool
  default     = true
}

# 노드 그룹 설정
variable "node_instance_types" {
  description = "일반 노드 인스턴스 타입"
  type        = list(string)
  default     = ["t3.medium", "t3.large"]
}

variable "node_group_min_size" {
  description = "일반 노드 그룹 최소 크기"
  type        = number
  default     = 1
}

variable "node_group_max_size" {
  description = "일반 노드 그룹 최대 크기"
  type        = number
  default     = 10
}

variable "node_group_desired_size" {
  description = "일반 노드 그룹 원하는 크기"
  type        = number
  default     = 3
}

variable "node_disk_size" {
  description = "일반 노드 디스크 크기 (GB)"
  type        = number
  default     = 50
}

# 프로덕션 노드 그룹 설정
variable "prod_node_instance_types" {
  description = "프로덕션 노드 인스턴스 타입"
  type        = list(string)
  default     = ["m5.large", "m5.xlarge"]
}

variable "prod_node_group_min_size" {
  description = "프로덕션 노드 그룹 최소 크기"
  type        = number
  default     = 2
}

variable "prod_node_group_max_size" {
  description = "프로덕션 노드 그룹 최대 크기"
  type        = number
  default     = 20
}

variable "prod_node_group_desired_size" {
  description = "프로덕션 노드 그룹 원하는 크기"
  type        = number
  default     = 5
}

variable "prod_node_disk_size" {
  description = "프로덕션 노드 디스크 크기 (GB)"
  type        = number
  default     = 100
}

# EKS 애드온 버전
variable "coredns_version" {
  description = "CoreDNS 애드온 버전"
  type        = string
  default     = "v1.10.1-eksbuild.5"
}

variable "kube_proxy_version" {
  description = "kube-proxy 애드온 버전"
  type        = string
  default     = "v1.28.2-eksbuild.2"
}

variable "vpc_cni_version" {
  description = "VPC CNI 애드온 버전"
  type        = string
  default     = "v1.15.1-eksbuild.1"
}

variable "ebs_csi_version" {
  description = "EBS CSI 드라이버 버전"
  type        = string
  default     = "v1.24.0-eksbuild.1"
}

# 데이터베이스 설정
variable "postgres_version" {
  description = "PostgreSQL 버전"
  type        = string
  default     = "15.4"
}

variable "db_instance_class" {
  description = "RDS 인스턴스 클래스"
  type        = string
  default     = "db.t3.medium"
}

variable "db_allocated_storage" {
  description = "RDS 할당된 스토리지 (GB)"
  type        = number
  default     = 100
}

variable "db_max_allocated_storage" {
  description = "RDS 최대 할당 스토리지 (GB)"
  type        = number
  default     = 1000
}

variable "db_username" {
  description = "데이터베이스 사용자명"
  type        = string
  default     = "proxynd"
  sensitive   = true
}

variable "db_password" {
  description = "데이터베이스 비밀번호"
  type        = string
  sensitive   = true
}

variable "db_backup_retention_period" {
  description = "데이터베이스 백업 보존 기간 (일)"
  type        = number
  default     = 7
}

variable "db_backup_window" {
  description = "데이터베이스 백업 윈도우"
  type        = string
  default     = "03:00-04:00"
}

variable "db_maintenance_window" {
  description = "데이터베이스 유지보수 윈도우"
  type        = string
  default     = "sun:04:00-sun:05:00"
}

# Redis 설정
variable "redis_version" {
  description = "Redis 버전"
  type        = string
  default     = "7.0"
}

variable "redis_node_type" {
  description = "Redis 노드 타입"
  type        = string
  default     = "cache.t3.medium"
}

variable "redis_num_cache_nodes" {
  description = "Redis 캐시 노드 수"
  type        = number
  default     = 2
}

# 스토리지 설정
variable "cache_retention_days" {
  description = "S3 캐시 보존 기간 (일)"
  type        = number
  default     = 30
}

# 로깅 설정
variable "log_retention_days" {
  description = "CloudWatch 로그 보존 기간 (일)"
  type        = number
  default     = 14
}

# 모니터링 설정
variable "enable_monitoring" {
  description = "상세 모니터링 활성화"
  type        = bool
  default     = true
}

variable "enable_logging" {
  description = "상세 로깅 활성화"
  type        = bool
  default     = true
}

# 보안 설정
variable "enable_encryption" {
  description = "암호화 활성화"
  type        = bool
  default     = true
}

variable "enable_secrets_manager" {
  description = "AWS Secrets Manager 사용"
  type        = bool
  default     = true
}

# 태그 설정
variable "additional_tags" {
  description = "추가 태그"
  type        = map(string)
  default     = {}
}

# 기능 플래그
variable "enable_autoscaling" {
  description = "자동 스케일링 활성화"
  type        = bool
  default     = true
}

variable "enable_spot_instances" {
  description = "스팟 인스턴스 사용"
  type        = bool
  default     = false
}

variable "enable_fargate" {
  description = "Fargate 프로파일 활성화"
  type        = bool
  default     = false
}

# 네트워크 보안
variable "allowed_cidr_blocks" {
  description = "허용된 CIDR 블록 목록"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "enable_waf" {
  description = "AWS WAF 활성화"
  type        = bool
  default     = true
}

# 백업 및 재해 복구
variable "enable_cross_region_backup" {
  description = "크로스 리전 백업 활성화"
  type        = bool
  default     = false
}

variable "backup_region" {
  description = "백업 리전"
  type        = string
  default     = "us-east-1"
}

# 비용 최적화
variable "enable_savings_plans" {
  description = "Savings Plans 활성화"
  type        = bool
  default     = false
}

variable "enable_reserved_instances" {
  description = "예약 인스턴스 활성화"
  type        = bool
  default     = false
}
