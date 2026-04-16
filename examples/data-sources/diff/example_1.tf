# Deploy initial release with specific configuration
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

# Check what would change if we modify the configuration
data "helm_diff" "mariadb_upgrade" {
  name       = "mariadb-instance"
  namespace  = "default"
  repository = "https://charts.helm.sh/stable"

  chart   = "mariadb"
  version = "7.1.0"  # Same version

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
  description = "Whether the proposed config differs from deployed"
  value       = data.helm_diff.mariadb_upgrade.has_changes
}

output "what_will_change" {
  description = "Detailed diff of configuration changes"
  value       = data.helm_diff.mariadb_upgrade.diff
}

output "changes_json" {
  description = "JSON structure showing modified resources"
  value       = data.helm_diff.mariadb_upgrade.diff_json
}
