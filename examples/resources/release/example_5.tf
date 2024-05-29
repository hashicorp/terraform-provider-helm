
# Install GCS plugin
`helm plugin install https://github.com/hayorov/helm-gcs.git`

# Run follow commands to setup GCS repository

# Init a new repository:
#   helm gcs init gs://bucket/path

# Add your repository to Helm:
#   helm repo add repo-name gs://bucket/path

# Push a chart to your repository:
#   helm gcs push chart.tar.gz repo-name

# Update Helm cache:
#   helm repo update

# Get your chart:

resource "helm_release" "GCS" {
  name        = "GCS"
  repository  = "gs://tf-test-helm-repo/charts"
  chart       = "chart"
}
