package helm

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRepository() *schema.Resource {
	return schema.DataSourceResourceShim("helm_repository", dataRepository())
}
