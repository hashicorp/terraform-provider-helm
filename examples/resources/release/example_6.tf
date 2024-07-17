
# Install AWS S3 plugin
`helm plugin install https://github.com/hypnoglow/helm-s3.git`

# Run follow commands to setup S3 repository

# Init a new repository:
#   helm s3 init s3://my-helm-charts/stable/myapp

# Add your repository to Helm:
#   helm repo add stable-myapp s3://my-helm-charts/stable/myapp/

# Push a chart to your repository:
#   helm s3 push chart.tar.gz repo-name

# Update Helm cache:
#   helm repo update

# Get your chart:

resource "helm_release" "S3" {
  name        = "S3"
  repository  = "s3://tf-test-helm-repo/charts"
  chart       = "chart"
}
