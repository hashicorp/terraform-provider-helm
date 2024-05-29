data "helm_template" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  show_only = [
    "templates/master-statefulset.yaml",
    "templates/master-svc.yaml",
  ]
  
  set {
    name  = "service.port"
    value = "13306"
  }

  set_sensitive {
    name = "rootUser.password"
    value = "s3cr3t!"
  }
}

resource "local_file" "mariadb_manifests" {
  for_each = data.helm_template.mariadb_instance.manifests

  filename = "./${each.key}"
  content  = each.value
}

output "mariadb_instance_manifest" {
  value = data.helm_template.mariadb_instance.manifest
}

output "mariadb_instance_manifests" {
  value = data.helm_template.mariadb_instance.manifests
}

output "mariadb_instance_notes" {
  value = data.helm_template.mariadb_instance.notes
}
