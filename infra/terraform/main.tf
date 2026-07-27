# Terraform entrypoint.
#
# Deliberately portable: everything except object storage runs as
# Kubernetes-native workloads (Strimzi for Kafka, CloudNativePG for
# Postgres, Bitnami's chart for Redis), so this applies unchanged against
# EKS, GKE, AKS, or a self-managed cluster — whatever `kubeconfig` points
# at. Provisioning the cluster itself is out of scope here; wire in
# whichever cloud's cluster module (or none, for on-prem) ahead of this.
#
# The one deliberate exception is object storage: S3's presigned-URL API
# is the closest thing this space has to a real standard (AWS S3, MinIO,
# Cloudflare R2, Backblaze B2 all implement it), so we provision a bucket
# with it directly rather than adding an abstraction layer with no other
# consumer. Point `s3_endpoint` at a non-AWS provider if you're not using
# real AWS S3 — see variables.tf.

terraform {
  required_providers {
    aws        = { source = "hashicorp/aws", version = "~> 5.0" }
    kubernetes = { source = "hashicorp/kubernetes", version = "~> 2.27" }
    helm       = { source = "hashicorp/helm", version = "~> 2.13" }
    kubectl    = { source = "gavinbunney/kubectl", version = "~> 1.14" } # for CRDs (Strimzi Kafka, CloudNativePG Cluster)
  }
}

provider "aws" {
  region = var.aws_region
}

# --- Kafka (Strimzi operator + KafkaNodePool/Kafka CR) -----------------
# Self-hosted on the cluster, not a managed queue — swap for a managed
# Kafka-API-compatible service (Confluent Cloud, Aiven, Redpanda Cloud)
# by pointing KAFKA_BROKERS at it instead, with no application code change.

resource "helm_release" "strimzi" {
  name             = "strimzi-kafka-operator"
  repository       = "https://strimzi.io/charts/"
  chart            = "strimzi-kafka-operator"
  namespace        = "kafka"
  create_namespace = true
}

resource "kubectl_manifest" "kafka_cluster" {
  depends_on = [helm_release.strimzi]
  yaml_body  = templatefile("${path.module}/manifests/kafka-cluster.yaml.tftpl", {
    name      = "${var.project}-${var.environment}"
    namespace = "kafka"
    replicas  = var.environment == "prod" ? 3 : 1
  })
}

# --- Postgres (CloudNativePG operator + Cluster CR) ---------------------
# Replaces RDS: same Postgres engine, runs anywhere the cluster does.
# For managed Postgres instead, just point DATABASE_URL at it and skip
# this module — the application code has no idea which one it's talking to.

resource "helm_release" "cnpg" {
  name             = "cloudnative-pg"
  repository       = "https://cloudnative-pg.github.io/charts"
  chart            = "cloudnative-pg"
  namespace        = "cnpg-system"
  create_namespace = true
}

resource "kubectl_manifest" "catalog_db" {
  depends_on = [helm_release.cnpg]
  yaml_body  = templatefile("${path.module}/manifests/postgres-cluster.yaml.tftpl", {
    name        = "${var.project}-catalog-${var.environment}"
    namespace   = "default"
    instances   = var.environment == "prod" ? 3 : 1
    storage_gb  = 100
  })
}

# --- Redis (query result / enrichment cache) -----------------------------
# Bitnami's chart runs a single-node Redis suitable for the query result
# and enrichment caches described in the article — swap for a managed
# Redis-compatible service the same way as Postgres above if preferred.

resource "helm_release" "redis" {
  name       = "query-cache"
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "redis"
  namespace  = "default"

  set {
    name  = "architecture"
    value = "standalone"
  }
}

# --- Object storage (the one AWS-flavored exception) ---------------------

resource "aws_s3_bucket" "product_images" {
  bucket = "${var.project}-product-images-${var.environment}"
}

resource "aws_s3_bucket_cors_configuration" "product_images" {
  bucket = aws_s3_bucket.product_images.id
  cors_rule {
    allowed_methods = ["PUT"]
    allowed_origins = var.allowed_upload_origins
    allowed_headers = ["*"]
  }
}
