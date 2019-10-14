package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataRepository_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmDataRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmDataRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmDataRepositoryConfigBasic(testRepositoryName, testRepositoryURLAlt),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("data.helm_repository.test", "metadata.0.url", testRepositoryURLAlt),
			),
		}},
	})
}

func testAccHelmDataRepositoryConfigBasic(name, url string) string {
	return fmt.Sprintf(`
		data "helm_repository" "test" {
 			name = %q
			url  = %q
		}
	`, name, url)
}
