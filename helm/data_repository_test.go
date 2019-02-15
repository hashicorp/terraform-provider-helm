package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataRepository_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURLAlt),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURLAlt),
			),
		}},
	})
}

func testAccHelmRepositoryConfigBasic(name, url string) string {
	return fmt.Sprintf(`
		data "helm_repository" "test" {
 			name = %q
			url  = %q
		}
	`, name, url)
}
