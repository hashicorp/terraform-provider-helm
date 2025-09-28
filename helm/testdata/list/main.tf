provider "helm" {}

variable "test_repo" {}

resource "helm_release" "test" {
  count      = 3
  name       = "test-${count.index}"
  repository = var.test_repo
  chart      = "test-chart"
}
