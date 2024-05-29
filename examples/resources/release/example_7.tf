
# run this first: `helm repo add bitnami https://charts.bitnami.com/bitnami`

resource "helm_release" "example" {
  name  = "redis"
  chart = "bitnami/redis"
}
