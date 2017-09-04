package helm

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/strvals"
)

var ErrReleaseNotFound = errors.New("release not found")

func resourceChart() *schema.Resource {
	return &schema.Resource{
		Create: resourceChartCreate,
		Read:   resourceChartRead,
		Delete: resourceChartDelete,
		Update: resourceChartUpdate,
		Exists: resourceChartExists,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Release name.",
			},
			"repository": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Repository where to locate the requested chart. If is an URL the chart is installed without install the repository.",
			},
			"chart": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Chart name to be installed.",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed.",
			},
			"values": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Values in raw yaml file to pass to helm.",
			},
			"set": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom values to be merge with the values.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				Description: "Namespace to install the release into.",
			},
			"repository_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Repository URL where to locate the requested chart without install the repository.",
			},
			"verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Verify the package before installing it.",
			},
			"keyring": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     os.ExpandEnv("$HOME/.gnupg/pubring.gpg"),
				Description: "Location of public keys used for verification.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
				Description: "Time in seconds to wait for any individual kubernetes operation.",
			},
			"disable_webhooks": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Prevent hooks from running.",
			},
			"force_update": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Force resource update through delete/recreate if needed.",
			},
			"recreate_pods": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "On update performs pods restart for the resource if applicable.",
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
							Description: "Name is the name of the release.",
						},
						"revision": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Version is an int32 which represents the version of the release.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace is the kubernetes namespace of the release.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Status of the release.",
						},
						"chart": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the chart.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A SemVer 2 conformant version string of the chart.",
						},
					},
				},
			},
		},
	}
}

func resourceChartCreate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	r, err := getRelease(m.GetHelmClient(), d)
	if err == nil {
		if r.Info.Status.GetCode() != release.Status_FAILED {
			return setIdAndMetadataFromRelease(d, r)
		}

		if err := resourceChartDelete(d, meta); err != nil {
			return err
		}
	}

	if err != ErrReleaseNotFound {
		return err
	}

	chart, _, err := getChart(d, m)
	if err != nil {
		return err
	}

	values, err := getValues(d)
	if err != nil {
		return err
	}

	opts := []helm.InstallOption{
		helm.ReleaseName(d.Get("name").(string)),
		helm.ValueOverrides(values),
		helm.InstallDisableHooks(d.Get("disable_webhooks").(bool)),
		helm.InstallTimeout(int64(d.Get("timeout").(int))),
		helm.InstallWait(true),
	}

	ns := d.Get("namespace").(string)
	res, err := m.GetHelmClient().InstallReleaseFromChart(chart, ns, opts...)
	if err != nil {
		return err
	}

	return setIdAndMetadataFromRelease(d, res.Release)
}

func resourceChartRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	r, err := getRelease(m.GetHelmClient(), d)
	if err != nil {
		return err
	}

	return setIdAndMetadataFromRelease(d, r)
}

func setIdAndMetadataFromRelease(d *schema.ResourceData, r *release.Release) error {
	d.SetId(r.Name)

	return d.Set("metadata", []map[string]interface{}{{
		"name":      r.Name,
		"revision":  r.Version,
		"namespace": r.Namespace,
		"status":    r.Info.Status.Code.String(),
		"chart":     r.Chart.Metadata.Name,
		"version":   r.Chart.Metadata.Version,
	}})
}

func resourceChartUpdate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	values, err := getValues(d)
	if err != nil {
		return err
	}

	_, path, err := getChart(d, m)
	if err != nil {
		return err
	}

	opts := []helm.UpdateOption{
		helm.UpdateValueOverrides(values),
		helm.UpgradeRecreate(d.Get("recreate_pods").(bool)),
		helm.UpgradeForce(d.Get("force_update").(bool)),
		helm.UpgradeDisableHooks(d.Get("disable_webhooks").(bool)),
		helm.UpgradeTimeout(int64(d.Get("timeout").(int))),
		helm.UpgradeWait(true),
	}

	name := d.Get("name").(string)
	res, err := m.GetHelmClient().UpdateRelease(name, path, opts...)
	if err != nil {
		return err
	}

	return setIdAndMetadataFromRelease(d, res.Release)
}
func resourceChartDelete(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)
	opts := []helm.DeleteOption{
		helm.DeleteDisableHooks(d.Get("disable_webhooks").(bool)),
		helm.DeletePurge(true),
		helm.DeleteTimeout(int64(d.Get("timeout").(int))),
	}

	if _, err := m.GetHelmClient().DeleteRelease(name, opts...); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceChartExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	m := meta.(*Meta)

	_, err := getRelease(m.GetHelmClient(), d)
	if err == nil {
		return true, nil
	}

	if err == ErrReleaseNotFound {
		return false, nil
	}

	return false, err
}

func getChart(d *schema.ResourceData, m *Meta) (c *chart.Chart, path string, err error) {
	l, err := newChartLocator(m,
		d.Get("repository").(string),
		d.Get("chart").(string),
		d.Get("version").(string),
		d.Get("verify").(bool),
		d.Get("keyring").(string),
	)
	if err != nil {
		return
	}

	path, err = l.Locate()
	if err != nil {
		return
	}

	c, err = chartutil.Load(path)
	if err != nil {
		return
	}

	if req, err := chartutil.LoadRequirements(c); err == nil {
		if err := checkDependencies(c, req); err != nil {
			return nil, "", err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, "", fmt.Errorf("cannot load requirements: %v", err)
	}

	return
}

func getValues(d *schema.ResourceData) ([]byte, error) {
	base := map[string]interface{}{}

	values := d.Get("values").(string)
	if values != "" {
		if err := yaml.Unmarshal([]byte(values), &base); err != nil {
			return nil, err
		}
	}

	for _, raw := range d.Get("set").(*schema.Set).List() {
		set := raw.(map[string]interface{})

		name := set["name"].(string)
		value := set["value"].(string)

		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			return nil, fmt.Errorf("failed parsing key %q with value %s, %s", name, value, err)
		}
	}

	yaml, err := yaml.Marshal(base)
	if err == nil {
		log.Printf("---[ values.yaml ]-----------------------------------\n%s\n", yaml)
	}

	return yaml, err
}

var all = []release.Status_Code{
	release.Status_UNKNOWN,
	release.Status_DEPLOYED,
	release.Status_DELETED,
	release.Status_DELETING,
	release.Status_FAILED,
}

func getRelease(client helm.Interface, d *schema.ResourceData) (*release.Release, error) {
	name := d.Get("name").(string)

	res, err := client.ListReleases(
		helm.ReleaseListFilter(name),
		helm.ReleaseListStatuses(all),
	)

	if err != nil {
		return nil, err
	}

	for _, r := range res.Releases {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, ErrReleaseNotFound
}

type chartLocator struct {
	meta *Meta

	name          string
	version       string
	repositoryURL string
	verify        bool
	keyring       string
}

func newChartLocator(meta *Meta, repository, name, version string, verify bool, keyring string) (*chartLocator, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)

	repositoryURL, name, err := resolveChartName(repository, name)
	if err != nil {
		return nil, err
	}

	return &chartLocator{
		meta:          meta,
		name:          name,
		version:       version,
		repositoryURL: repositoryURL,
		verify:        verify,
		keyring:       keyring,
	}, nil

}

func resolveChartName(repository, name string) (string, string, error) {
	_, err := url.ParseRequestURI(repository)
	if err == nil {
		return repository, name, nil
	}

	if strings.Index(name, "/") == -1 && repository != "" {
		name = fmt.Sprintf("%s/%s", repository, name)
	}

	return "", name, nil
}

func (l *chartLocator) Locate() (string, error) {
	pipeline := []func() (string, error){
		l.locateChartPathInLocal,
		l.locateChartPathInLocalRepository,
		l.locateChartPathInRepository,
	}

	for _, f := range pipeline {
		path, err := f()
		if err != nil {
			return "", err
		}

		if path == "" {
			continue
		}

		return path, err
	}

	return "", fmt.Errorf("chart %q not found", l.name)
}

func (l *chartLocator) locateChartPathInLocal() (string, error) {
	fi, err := os.Stat(l.name)
	if err != nil {
		if filepath.IsAbs(l.name) || strings.HasPrefix(l.name, ".") {
			return "", fmt.Errorf("path %q not found", l.name)
		}

		return "", nil
	}

	abs, err := filepath.Abs(l.name)
	if err != nil {
		return "", err
	}

	if l.verify {
		if fi.IsDir() {
			return "", fmt.Errorf("cannot verify a directory")
		}

		if _, err := downloader.VerifyChart(abs, l.keyring); err != nil {
			return "", err
		}
	}

	return abs, nil
}

func (l *chartLocator) locateChartPathInLocalRepository() (string, error) {
	repo := filepath.Join(l.meta.Settings.Home.Repository(), l.name)
	if _, err := os.Stat(repo); err == nil {
		return filepath.Abs(repo)
	}

	return "", nil
}

func (l *chartLocator) locateChartPathInRepository() (string, error) {
	ref, err := l.retrieveChartURL(l.repositoryURL, l.name, l.version)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %q, %s", l.name, err)
	}

	if _, err := os.Stat(l.meta.Settings.Home.Archive()); os.IsNotExist(err) {
		if err := os.MkdirAll(l.meta.Settings.Home.Archive(), 0744); err != nil {
			return "", fmt.Errorf("failed to create archive folder, %s", err)
		}
	}

	return l.downloadChart(ref)
}

func (l *chartLocator) retrieveChartURL(repositoryURL, name, version string) (string, error) {
	if repositoryURL == "" {
		return name, nil
	}

	return repo.FindChartInRepoURL(
		repositoryURL, name, version,
		tlsCertFile, tlsKeyFile, tlsCaCertFile, getter.All(*l.meta.Settings),
	)
}

func (l *chartLocator) downloadChart(ref string) (string, error) {
	dl := downloader.ChartDownloader{
		HelmHome: l.meta.Settings.Home,
		Out:      os.Stdout,
		Keyring:  l.keyring,
		Getters:  getter.All(*l.meta.Settings),
	}

	if l.verify {
		dl.Verify = downloader.VerifyAlways
	}

	filename, _, err := dl.DownloadTo(ref, l.version, l.meta.Settings.Home.Archive())
	if err != nil {
		return "", err
	}

	debug("Fetched %s to %s\n", ref, filename)
	return filepath.Abs(filename)
}

// from helm
func checkDependencies(ch *chart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}
