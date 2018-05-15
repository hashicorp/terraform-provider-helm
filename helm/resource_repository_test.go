package helm

import (
	"fmt"
	"testing"

	"k8s.io/helm/pkg/repo"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceRepository_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", testRepositoryName),
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.url", testRepositoryURL),
			),
		}, {
			Config: testAccHelmRepositoryConfigBasic(testRepositoryName, testRepositoryURLAlt),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_repository.test", "metadata.0.name", testRepositoryName),
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

func testAccCheckHelmRepositoryDestroy(s *terraform.State) error {
	settings := testAccProvider.Meta().(*Meta).Settings

	f, err := repo.LoadRepositoriesFile(settings.Home.RepositoryFile())
	if err != nil {
		return err
	}

	for _, r := range f.Repositories {
		if r.Name == testRepositoryName {
			return fmt.Errorf("found %q repository", testResourceName)
		}
	}

	return nil
}
