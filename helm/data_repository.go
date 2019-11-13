package helm

import (
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
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
	m.Lock()
	defer m.Unlock()

	name := d.Get("name").(string)

	//var entry *repo.Entry
	var file *repo.File

	if fileExists(m.Settings.RepositoryConfig) {
		var err error
		if file, err = repo.LoadFile(m.Settings.RepositoryConfig); err != nil {
			return err
		}

	} else {
		file = repo.NewFile()
	}

	entry := file.Get(name)

	// Not sure I agree with the logic here. Should a data source really update an underlying resource every time its called?
	if entry == nil {
		entry = &repo.Entry{
			Name:     name,
			URL:      d.Get("url").(string),
			CertFile: d.Get("cert_file").(string),
			KeyFile:  d.Get("key_file").(string),
			CAFile:   d.Get("ca_file").(string),
			Username: d.Get("username").(string),
			Password: d.Get("password").(string),
		}
	} else {
		entry.URL = d.Get("url").(string)
		entry.CertFile = d.Get("cert_file").(string)
		entry.KeyFile = d.Get("key_file").(string)
		entry.CAFile = d.Get("ca_file").(string)
		entry.Username = d.Get("username").(string)
		entry.Password = d.Get("password").(string)
	}

	file.Update(entry)

	if err := file.WriteFile(m.Settings.RepositoryConfig, 0644); err != nil {
		return err
	}

	re, err := repo.NewChartRepository(entry, getter.All(m.Settings))

	if err != nil {
		return err
	}

	if _, err := re.DownloadIndexFile(); err != nil {
		return err
	}

	return setIDAndMetadataFromRepository(d, entry)
}

func setIDAndMetadataFromRepository(d *schema.ResourceData, r *repo.Entry) error {
	d.SetId(r.Name)
	return d.Set("metadata", []map[string]interface{}{{
		"name": r.Name,
		"url":  r.URL,
	}})
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
