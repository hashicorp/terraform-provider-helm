
provider "helm" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
  # localhost registry with password protection
  registries = [
    {
      url      = "oci://localhost:5000"
      username = "username"
      password = "password"
    }
  ]
}

resource "helm_release" "example" {
  name        = "testchart"
  namespace   = "helm_registry"
  repository  = "oci://localhost:5000/helm-charts"
  version     = "1.2.3"
  chart       = "test-chart"
}
