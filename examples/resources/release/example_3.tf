resource "helm_release" "example" {
  name  = "redis"
  chart = "https://charts.bitnami.com/bitnami/redis-10.7.16.tgz"
}
