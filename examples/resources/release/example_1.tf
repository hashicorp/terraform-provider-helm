resource "helm_release" "example" {
  name       = "my-redis-release"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "redis"
  version    = "6.0.1"

  values = [
    "${file("values.yaml")}"
  ]

  set = [
    {
      name  = "cluster.enabled"
      value = "true"
    },
    {
      name  = "delimited.value"
      value = "neither:map,nor,list"
      type  = "literal"
    },
    {
      name  = "service.annotations.prometheus\\.io/port"
      value = "9127"
      type  = "string"
    },
  ]
}
