resource "helm_release" "mariadb" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  set {
    name  = "service.port"
    value = "13306"
  }

  set {
    name  = "replication.enabled"
    value = "false"
  }
}

data "helm_diff" "mariadb_upgrade" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"

  set = [
    {
      name  = "service.port"
      value = "3306"  
    },
    {
      name  = "replication.enabled"
      value = "true"
    }
  ]

  depends_on = [helm_release.mariadb]
}

output "has_changes" {
  value       = data.helm_diff.mariadb_upgrade.has_changes
}

output "what_will_change" {
  value       = data.helm_diff.mariadb_upgrade.diff
}

output "changes_json" {
  value       = data.helm_diff.mariadb_upgrade.diff_json
}