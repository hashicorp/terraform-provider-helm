package helm

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/yaml.v1"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
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
			"value": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom values to be merge with the values.yaml.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"content": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
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
	client := meta.(helm.Interface)

	if r, err := getRelease(client, d); err == nil {
		return setIdAndMetadataFromRelease(d, r)
	}

	chart, _, err := getChart(d)
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

	res, err := client.InstallReleaseFromChart(chart, getNamespace(d), opts...)
	if err != nil {
		return err
	}

	return setIdAndMetadataFromRelease(d, res.Release)
}

func resourceChartRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(helm.Interface)

	r, err := getRelease(client, d)
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
	client := meta.(helm.Interface)

	values, err := getValues(d)
	if err != nil {
		return err
	}

	_, path, err := getChart(d)
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
	res, err := client.UpdateRelease(name, path, opts...)
	if err != nil {
		return err
	}

	return setIdAndMetadataFromRelease(d, res.Release)
}
func resourceChartDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(helm.Interface)

	opts := []helm.DeleteOption{
		helm.DeleteDisableHooks(d.Get("disable_webhooks").(bool)),
		helm.DeletePurge(true),
		helm.DeleteTimeout(int64(d.Get("timeout").(int))),
	}

	_, err := client.DeleteRelease(d.Get("name").(string), opts...)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceChartExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(helm.Interface)

	_, err := getRelease(client, d)
	if err == nil {
		return true, nil
	}

	if err == ErrReleaseNotFound {
		return false, nil
	}

	return false, err
}

func getNamespace(d *schema.ResourceData) string {
	namespace := d.Get("namespace").(string)
	if namespace != "" {
		return namespace
	}

	return defaultNamespace()
}

func getChart(d *schema.ResourceData) (c *chart.Chart, path string, err error) {
	path, err = locateChartPath(
		d.Get("repository").(string),
		d.Get("chart").(string),
		d.Get("version").(string),
		d.Get("verify").(bool),
		d.Get("keyring").(string),
	)

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

	for _, raw := range d.Get("value").(*schema.Set).List() {
		value := raw.(map[string]interface{})

		name := value["name"].(string)
		content := value["content"].(string)

		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, content), base); err != nil {
			return nil, fmt.Errorf("failed parsing key %q with value %s, %s", name, content, err)
		}
	}

	return yaml.Marshal(base)
}

func getRelease(client helm.Interface, d *schema.ResourceData) (*release.Release, error) {
	name := d.Get("name").(string)

	res, err := client.ListReleases(
		helm.ReleaseListFilter(name),
		helm.ReleaseListNamespace(getNamespace(d)),
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

type chartLocator func(repositoryURL, name, version string, verify bool, keyring string) (string, error)

func locateChartPath(repository, name, version string, verify bool, keyring string) (string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)

	repositoryURL, name, err := resolveChartName(repository, name)
	if err != nil {
		return "", err
	}

	pipeline := []chartLocator{
		locateChartPathInLocal,
		locateChartPathInLocalRepository,
		locateChartPathInRepository,
	}

	for _, f := range pipeline {
		path, err := f(repositoryURL, name, version, verify, keyring)
		if err != nil {
			return "", err
		}

		if path == "" {
			continue
		}

		return path, err
	}

	return "", fmt.Errorf("chart %q not found", name)
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

func locateChartPathInLocal(repositoryURL, name, version string, verify bool, keyring string) (string, error) {
	fi, err := os.Stat(name)
	if err != nil {
		if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
			return "", fmt.Errorf("path %q not found", name)
		}

		return "", nil
	}

	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	if !verify {
		if fi.IsDir() {
			return "", fmt.Errorf("cannot verify a directory")
		}

		if _, err := downloader.VerifyChart(abs, keyring); err != nil {
			return "", err
		}
	}

	return abs, nil
}

func locateChartPathInLocalRepository(
	repositoryURL, name, version string, verify bool, keyring string) (string, error) {

	crepo := filepath.Join(settings.Home.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	return "", nil
}

func locateChartPathInRepository(
	repositoryURL, name, version string, verify bool, keyring string) (string, error) {

	name, err := retrieveChartURL(repositoryURL, name, version)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %q, %s", name, err)
	}

	if _, err := os.Stat(settings.Home.Archive()); os.IsNotExist(err) {
		os.MkdirAll(settings.Home.Archive(), 0744)
	}

	return downloadChart(name, version, verify, keyring)
}

func retrieveChartURL(repositoryURL, name, version string) (string, error) {
	if repositoryURL == "" {
		return name, nil
	}

	return repo.FindChartInRepoURL(
		repositoryURL, name, version,
		tlsCertFile, tlsKeyFile, tlsCaCertFile, getter.All(settings),
	)
}

func downloadChart(name, version string, verify bool, keyring string) (string, error) {
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Keyring:  keyring,
		Getters:  getter.All(settings),
	}

	if verify {
		dl.Verify = downloader.VerifyAlways
	}

	filename, _, err := dl.DownloadTo(name, version, settings.Home.Archive())
	if err != nil {
		return "", err
	}

	debug("Fetched %s to %s\n", name, filename)
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

func defaultNamespace() string {
	if ns, _, err := kube.GetConfig(kubeContext).Namespace(); err == nil {
		return ns
	}
	return "default"
}
