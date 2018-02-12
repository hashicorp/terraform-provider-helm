package helm

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

// ErrRepositoryNotFound is the error when a Helm repository is not found
var ErrRepositoryNotFound = errors.New("repository not found")

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Read:   resourceRepositoryRead,
		Delete: resourceRepositoryDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Chart repository name.",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Chart repository URL.",
			},
			"key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Identify HTTPS client using this SSL key file.",
			},
			"cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Identify HTTPS client using this SSL certificate file.",
			},
			"ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Verify certificates of HTTPS-enabled servers using this CA bundle",
			},
			"metadata": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Status of the deployed release.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the repository read from the home.",
						},
						"url": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "URL of the repository read from the home.",
						},
					},
				},
			},
		},
	}
}

func resourceRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)
	err := addRepository(m,
		name,
		d.Get("url").(string),
		m.Settings.Home,
		d.Get("cert_file").(string),
		d.Get("key_file").(string),
		d.Get("ca_file").(string),
		false,
	)

	if err != nil {
		return err
	}

	debug("%q has been added to your repositories\n", name)
	return resourceRepositoryRead(d, meta)
}

func resourceRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	r, err := getRepository(d, m)
	if err != nil {
		return err
	}

	return setIDAndMetadataFromRepository(d, r)
}

func resourceRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)
	if err := removeRepoLine(name, m.Settings.Home); err != nil {
		return err
	}

	debug("%q has been removed from your repositories\n", name)
	d.SetId("")
	return nil
}

func setIDAndMetadataFromRepository(d *schema.ResourceData, r *repo.Entry) error {
	d.SetId(r.Name)
	return d.Set("metadata", []map[string]interface{}{{
		"name": r.Name,
		"url":  r.URL,
	}})
}

func getRepository(d *schema.ResourceData, m *Meta) (*repo.Entry, error) {
	name := d.Get("name").(string)

	f, err := repo.LoadRepositoriesFile(m.Settings.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}

	for _, r := range f.Repositories {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, ErrRepositoryNotFound

}

// from helm
func addRepository(m *Meta,
	name, url string, home helmpath.Home, certFile, keyFile, caFile string, noUpdate bool,
) error {

	f, err := repo.LoadRepositoriesFile(home.RepositoryFile())
	if err != nil {
		return err
	}

	if noUpdate && f.Has(name) {
		return fmt.Errorf("repository name (%s) already exists, please specify a different name", name)
	}

	cif := home.CacheIndex(name)
	c := repo.Entry{
		Name:     name,
		Cache:    cif,
		URL:      url,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	r, err := repo.NewChartRepository(&c, getter.All(*m.Settings))
	if err != nil {
		return err
	}

	if err := r.DownloadIndexFile(home.Cache()); err != nil {
		return fmt.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", url, err.Error())
	}

	f.Update(&c)

	return f.WriteFile(home.RepositoryFile(), 0644)
}

func removeRepoLine(name string, home helmpath.Home) error {
	repoFile := home.RepositoryFile()
	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !r.Remove(name) {
		return fmt.Errorf("no repo named %q found", name)
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if err := removeRepoCache(name, home); err != nil {
		return err
	}

	return nil
}

func removeRepoCache(name string, home helmpath.Home) error {
	if _, err := os.Stat(home.CacheIndex(name)); err == nil {
		err = os.Remove(home.CacheIndex(name))
		if err != nil {
			return err
		}
	}
	return nil
}
