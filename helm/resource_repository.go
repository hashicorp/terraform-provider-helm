package helm

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRepository() *schema.Resource {
	return schema.DataSourceResourceShim("helm_repository", dataRepository())
}
