package helm

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

func dataRepository() *schema.Resource {
	return &schema.Resource{
		Read: dataRepositoryRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Chart repository name",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Chart repository URL",
			},
			"key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Identify HTTPS client using this SSL key file",
			},
			"cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Identify HTTPS client using this SSL certificate file",
			},
			"ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Verify certificates of HTTPS-enabled servers using this CA bundle",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username for HTTP basic authentication",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password for HTTP basic authentication",
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Status of the deployed release",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the repository read from the home",
						},
						"url": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "URL of the repository read from the home",
						},
					},
				},
			},
		},
	}
}

func dataRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)
	err := addRepository(m,
		name,
		d.Get("url").(string),
		m.Settings.Home,
		d.Get("cert_file").(string),
		d.Get("key_file").(string),
		d.Get("ca_file").(string),
		d.Get("username").(string),
		d.Get("password").(string),
	)

	if err != nil {
		return err
	}

	debug("%q has been added to your repositories\n", name)

	r, err := getRepository(d, m)
	if err != nil {
		return err
	}

	return setIDAndMetadataFromRepository(d, r)
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

	return nil, errors.New("repository not found")
}

func addRepository(m *Meta, name, url string, home helmpath.Home, certFile, keyFile, caFile string, username string, password string) error {
	m.Lock()
	defer m.Unlock()

	repoFile, err := repo.LoadRepositoriesFile(home.RepositoryFile())
	if err != nil {
		return err
	}

	cif := home.CacheIndex(name)
	entry := repo.Entry{
		Name:     name,
		Cache:    cif,
		URL:      url,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
		Username: username,
		Password: password,
	}

	repo, err := repo.NewChartRepository(&entry, getter.All(*m.Settings))
	if err != nil {
		return err
	}

	if err := repo.DownloadIndexFile(home.Cache()); err != nil {
		return fmt.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", url, err.Error())
	}

	repoFile.Update(&entry)

	return repoFile.WriteFile(home.RepositoryFile(), 0644)
}

func setIDAndMetadataFromRepository(d *schema.ResourceData, r *repo.Entry) error {
	d.SetId(r.Name)
	return d.Set("metadata", []map[string]interface{}{{
		"name": r.Name,
		"url":  r.URL,
	}})
}
