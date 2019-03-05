package helm

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	extensions "k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/cmd/helm/installer"
	tiller_env "k8s.io/helm/pkg/tiller/environment"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

const (
	deploymentName = "tiller-deploy"
)

func resourceTiller() *schema.Resource {
	return &schema.Resource{
		Create: resourceTillerCreate,
		Read:   resourceTillerRead,
		Update: resourceTillerUpdate,
		Delete: resourceTillerDelete,
		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     tiller_env.DefaultTillerNamespace,
				Description: "Set an alternative Tiller namespace.",
			},
			"tiller_image": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "gcr.io/kubernetes-helm/tiller:v2.11.0",
				Description: "Tiller image to install.",
			},
			"service_account": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				Description: "Service account to install Tiller with.",
			},
			"automount_service_account_token": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "Auto-mount the given service account to tiller.",
			},
			"max_history": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Default:     0,
				Description: "Maximum number of release versions stored per release.",
			},
			"override": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "Override values for the Tiller Deployment manifest.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"verify_tls": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Whether server should be accessed without verifying the TLS certificate.",
			},
			"enable_tls": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Enables TLS communications with the Tiller.",
			},
			"client_key": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "$HELM_HOME/key.pem",
				Description: "PEM-encoded client certificate key for TLS authentication.",
			},
			"client_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "$HELM_HOME/cert.pem",
				Description: "PEM-encoded client certificate for TLS authentication.",
			},
			"ca_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "$HELM_HOME/ca.pem",
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
			},
			"listen_localhost": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Let Tiller only listen on localhost.",
			},
			"metadata": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Status of the tiller deployment.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the Tiller Deployment.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace of the Tiller Deployment.",
						},
						"generation": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The generation number of the Tiller Deployment.",
						},
						"resource_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource version of the Tiller Deployment.",
						},
						"self_link": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The self link URL of the Tiller Deployment.",
						},
						"uid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique ID of the Tiller Deployment.",
						},
					},
				},
			},
		},
	}
}

func resourceTillerCreate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	o := buildInstallerOptions(d)
	if err := installer.Install(m.K8sClient, o); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return nil
		}

		return fmt.Errorf("error installing: %s", err)
	}

	if err := m.waitForTiller(o); err != nil {
		return err
	}

	debug("Tiller has been installed into your Kubernetes Cluster.")
	id := fmt.Sprintf("%s/%s", d.Get("namespace"), deploymentName)
	d.SetId(id)

	return resourceTillerRead(d, meta)
}

func resourceTillerUpdate(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)
	debug("Upgrading tiller.")

	o := buildInstallerOptions(d)
	o.ForceUpgrade = true
	if err := installer.Upgrade(m.K8sClient, o); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return nil
		}

		return fmt.Errorf("error installing: %s", err)
	}

	if err := m.waitForTiller(o); err != nil {
		return err
	}
	return resourceTillerRead(d, m)
}

func resourceTillerRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	deployment, err := getTiller(d, m)
	if err != nil {
		return err
	}

	d.Set("metadata", []map[string]interface{}{{
		"name":             deployment.GetName(),
		"namespace":        deployment.GetNamespace(),
		"generation":       deployment.GetGeneration(),
		"resource_version": deployment.GetResourceVersion(),
		"self_link":        deployment.GetSelfLink(),
		"uid":              deployment.GetUID(),
	}})
	return nil
}

func resourceTillerDelete(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)
	debug("Uninstalling tiller.")

	clientSet, err := internalclientset.NewForConfig(m.K8sConfig)
	if err != nil {
		return err
	}

	o := buildInstallerOptions(d)
	if err := installer.Uninstall(clientSet, o); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return nil
		}

		return fmt.Errorf("error uninstalling: %s", err)
	}
	d.SetId("")
	return nil
}

func buildInstallerOptions(d *schema.ResourceData) *installer.Options {
	o := &installer.Options{}
	o.Namespace = d.Get("namespace").(string)
	o.ImageSpec = d.Get("tiller_image").(string)
	o.ServiceAccount = d.Get("service_account").(string)
	o.AutoMountServiceAccountToken = d.Get("automount_service_account_token").(bool)
	o.MaxHistory = d.Get("max_history").(int)

	for _, rule := range d.Get("override").([]interface{}) {
		o.Values = append(o.Values, rule.(string))
	}

	o.EnableTLS = d.Get("enable_tls").(bool)
	if o.EnableTLS {
		o.TLSCertFile = d.Get("client_certificate").(string)
		o.TLSKeyFile = d.Get("client_key").(string)
		o.VerifyTLS = d.Get("verify_tls").(bool)
		if o.VerifyTLS {
			o.TLSCaCertFile = d.Get("ca_certificate").(string)
		}
	}
	if d.Get("listen_localhost").(bool) {
		o.Values = []string{
			"spec.template.spec.containers[0].command={/tiller,--listen=localhost:44134}",
		}
	}
	return o
}

func getTiller(d *schema.ResourceData, m *Meta) (*extensions.Deployment, error) {
	namespace := d.Get("namespace").(string)

	obj, err := m.K8sClient.Extensions().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return obj, nil
}
