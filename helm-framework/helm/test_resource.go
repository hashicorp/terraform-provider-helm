package helm

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var (
	_ resource.ResourceWithConfigure = &helmReleaseResource{}
)

type helmReleaseResource struct {
}

func NewHelmReleaseResource() resource.Resource {
	return &helmReleaseResource{}
}

func (r *helmReleaseResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
}

func (r *helmReleaseResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "helm_release"
}

func (r *helmReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are available in the resource",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Release name. The length must not be longer than 53 characters",
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Namespace to install the release into",
			},
		},
	}
}

func (r *helmReleaseResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {

}

func (r *helmReleaseResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
}

func (r *helmReleaseResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
}

func (r *helmReleaseResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
}
