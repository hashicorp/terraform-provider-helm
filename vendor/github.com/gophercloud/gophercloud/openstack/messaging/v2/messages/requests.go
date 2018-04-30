package messages

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// ListOptsBuilder allows extensions to add additional parameters to the
// List request.
type ListOptsBuilder interface {
	ToMessageListQuery() (string, error)
}

// ListOpts params to be used with List.
type ListOpts struct {
	// Limit instructs List to refrain from sending excessively large lists of queues
	Limit int `q:"limit,omitempty"`

	// Marker and Limit control paging. Marker instructs List where to start listing from.
	Marker string `q:"marker,omitempty"`

	// Indicate if the messages can be echoed back to the client that posted them.
	Echo bool `q:"echo,omitempty"`

	// Indicate if the messages list should include the claimed messages.
	IncludeClaimed bool `q:"include_claimed,omitempty"`

	//Indicate if the messages list should include the delayed messages.
	IncludeDelayed bool `q:"include_delayed,omitempty"`
}

func (opts ListOpts) ToMessageListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

// ListMessages lists messages on a specific queue based off queue name.
func List(client *gophercloud.ServiceClient, queueName string, opts ListOptsBuilder) pagination.Pager {
	url := listURL(client, queueName)
	if opts != nil {
		query, err := opts.ToMessageListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	pager := pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return MessagePage{pagination.LinkedPageBase{PageResult: r}}
	})
	return pager
}

// CreateOptsBuilder Builder.
type CreateOptsBuilder interface {
	ToMessageCreateMap() (map[string]interface{}, error)
}

// BatchCreateOpts is an array of CreateOpts.
type BatchCreateOpts []CreateOpts

// CreateOpts params to be used with Create.
type CreateOpts struct {
	// TTL specifies how long the server waits before marking the message
	// as expired and removing it from the queue.
	TTL int `json:"ttl,omitempty"`

	// Delay specifies how long the message can be claimed.
	Delay int `json:"delay,omitempty"`

	// Body specifies an arbitrary document that constitutes the body of the message being sent.
	Body map[string]interface{} `json:"body" required:"true"`
}

// ToMessageCreateMap constructs a request body from BatchCreateOpts.
func (opts BatchCreateOpts) ToMessageCreateMap() (map[string]interface{}, error) {
	messages := make([]map[string]interface{}, len(opts))
	for i, message := range opts {
		messageMap, err := message.ToMap()
		if err != nil {
			return nil, err
		}
		messages[i] = messageMap
	}
	return map[string]interface{}{"messages": messages}, nil
}

// ToMap constructs a request body from UpdateOpts.
func (opts CreateOpts) ToMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "")
}

// Create creates a message on a specific queue based of off queue name.
func Create(client *gophercloud.ServiceClient, queueName string, opts CreateOptsBuilder) (r CreateResult) {
	b, err := opts.ToMessageCreateMap()
	if err != nil {
		r.Err = err
		return
	}

	_, r.Err = client.Post(createURL(client, queueName), b, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{201},
	})
	return
}
