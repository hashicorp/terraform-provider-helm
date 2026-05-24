data "helm_diff" "mariadb_instance" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  set = [
    {
      name  = "service.port"
      value = "13306"
    }
  ]

  set_sensitive = [
    {
      name  = "rootUser.password"
      value = "s3cr3t!"
    }
  ]
}


output "has_changes" {
  value = data.helm_diff.mariadb_instance.has_changes
}

output "diff_output" {
  value = data.helm_diff.mariadb_instance.diff
}

output "current_manifest" {
  value = data.helm_diff.mariadb_instance.current_manifest
}

output "proposed_manifest" {
  value = data.helm_diff.mariadb_instance.proposed_manifest
}