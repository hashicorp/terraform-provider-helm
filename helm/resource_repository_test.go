package helm

import (
	"fmt"
	"os"
	"testing"

	"k8s.io/helm/pkg/repo"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

// These tests are kept to test backwards compatibility for the helm_repository resource

func TestAccResourceRepository_basic(t *testing.T) {
	name := fmt.Sprintf("%s-%s", testRepositoryName, acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Note: this helm resource does not automatically create namespaces so no cleanup needed here

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
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

func testAccPreCheckHelmRepositoryDestroy(t *testing.T, name string) {
	settings := testAccProvider.Meta().(*Meta).Settings

	repoFile := settings.Home.RepositoryFile()
	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Remove(name) {
		t.Log(fmt.Sprintf("no repo named %q found, nothing to do", name))
		return
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		t.Fatalf("Failed to write repositories file: %s", err)
	}

	if _, err := os.Stat(settings.Home.CacheIndex(name)); err == nil {
		err = os.Remove(settings.Home.CacheIndex(name))
		if err != nil {
			t.Fatalf("Failed to remove repository cache: %s", err)
		}
	}

	t.Log(fmt.Sprintf("%q has been removed from your repositories\n", name))
}
