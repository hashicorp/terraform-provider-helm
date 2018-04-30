package capsules

import (
	"encoding/json"
	"time"

	"github.com/gophercloud/gophercloud"
)

type commonResult struct {
	gophercloud.Result
}

// Extract is a function that accepts a result and extracts a capsule resource.
func (r commonResult) Extract() (*Capsule, error) {
	var s *Capsule
	err := r.ExtractInto(&s)
	return s, err
}

// GetResult represents the result of a get operation.
type GetResult struct {
	commonResult
}

// CreateResult is the response from a Create operation. Call its Extract
// method to interpret it as a Server.
type CreateResult struct {
	gophercloud.ErrResult
}

// Represents a Container Orchestration Engine Bay, i.e. a cluster
type Capsule struct {
	// UUID for the capsule
	UUID string `json:"uuid"`

	// ID for the capsule
	ID int `json:"id"`

	// User ID for the capsule
	UserID string `json:"user_id"`

	// Project ID for the capsule
	ProjectID string `json:"project_id"`

	// cpu for the capsule
	CPU float64 `json:"cpu"`

	// Memory for the capsule
	Memory string `json:"memory"`

	// The name of the capsule
	MetaName string `json:"meta_name"`

	// Indicates whether capsule is currently operational. Possible values include:
	// Running,
	Status string `json:"status"`

	// The created time of the capsule.
	CreatedAt time.Time `json:"-"`

	// The updated time of the capsule.
	UpdatedAt time.Time `json:"-"`

	// Links includes HTTP references to the itself, useful for passing along to
	// other APIs that might want a server reference.
	Links []interface{} `json:"links"`

	// The capsule version
	CapsuleVersion string `json:"capsule_version"`

	// The capsule restart policy
	RestartPolicy string `json:"restart_policy"`

	// The capsule metadata labels
	MetaLabels map[string]string `json:"meta_labels"`

	// The list of containers uuids inside capsule.
	ContainersUUIDs []string `json:"containers_uuids"`

	// The capsule IP addresses
	Addresses map[string][]Address `json:"addresses"`

	// The capsule volume attached information
	VolumesInfo map[string][]string `json:"volumes_info"`
}

type Address struct {
	PreserveOnDelete bool    `json:"preserve_on_delete"`
	Addr             string  `json:"addr"`
	Port             string  `json:"port"`
	Version          float64 `json:"version"`
	SubnetID         string  `json:"subnet_id"`
}

func (r *Capsule) UnmarshalJSON(b []byte) error {
	type tmp Capsule
	var s struct {
		tmp
		CreatedAt gophercloud.JSONRFC3339ZNoT `json:"created_at"`
		UpdatedAt gophercloud.JSONRFC3339ZNoT `json:"updated_at"`
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*r = Capsule(s.tmp)

	r.CreatedAt = time.Time(s.CreatedAt)
	r.UpdatedAt = time.Time(s.UpdatedAt)

	return nil
}
