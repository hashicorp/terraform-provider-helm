package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// These tests are kept to test backwards compatibility for the helm_repository resource

func TestAccResourceRepository_basic(t *testing.T) {
	name := fmt.Sprintf("%s-%s", testRepositoryName, acctest.RandString(10))
	// Adding tfproviderlint ignore for missing CheckDestroy as this resource
	// is being removed next release per the message on the data_source_repository
	// Space is required for linter ignore to work

	//lintignore:AT001
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t, "") },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmRepositoryConfigBasic(name, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(name, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(name, testRepositoryURLAlt),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.url", testRepositoryURLAlt),
			),
		}},
	})
}

func testAccHelmRepositoryConfigBasic(name, url string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "test" {
 			name = %q
			url  = %q
		}
	`, name, url)
}
