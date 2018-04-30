package testing

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/messaging/v2/messages"
	th "github.com/gophercloud/gophercloud/testhelper"
	fake "github.com/gophercloud/gophercloud/testhelper/client"
)

// QueueName is the name of the queue
var QueueName = "FakeTestQueue"

// CreateMessageResponse is a sample response to a Create message.
const CreateMessageResponse = `
{
  "resources": [
    "/v2/queues/demoqueue/messages/51db6f78c508f17ddc924357",
    "/v2/queues/demoqueue/messages/51db6f78c508f17ddc924358"
  ]
}`

// CreateMessageRequest is a sample request to create a message.
const CreateMessageRequest = `
{
  "messages": [
	{
	  "body": {
		"backup_id": "c378813c-3f0b-11e2-ad92-7823d2b0f3ce",
		"event": "BackupStarted"
	  },
	  "delay": 20,
	  "ttl": 300
	},
	{
	  "body": {
		"current_bytes": "0",
		"event": "BackupProgress",
		"total_bytes": "99614720"
	  }
	}
  ]
}`

// ListMessagesResponse is a sample response to list messages.
const ListMessagesResponse1 = `
{
    "messages": [
        {
            "body": {
                "current_bytes": "0",
                "event": "BackupProgress",
                "total_bytes": "99614720"
            },
            "age": 482,
            "href": "/v2/queues/FakeTestQueue/messages/578edfe6508f153f256f717b",
            "id": "578edfe6508f153f256f717b",
            "ttl": 3600,
            "checksum": "MD5:abf7213555626e29c3cb3e5dc58b3515"
        }
    ],
    "links": [
        {
            "href": "/v2/queues/FakeTestQueue/messages?marker=1",
            "rel": "next"
        }
    ]
}`

// ListMessagesResponse is a sample response to list messages.
const ListMessagesResponse2 = `
{
    "messages": [
        {
            "body": {
                "current_bytes": "0",
                "event": "BackupProgress",
                "total_bytes": "99614720"
            },
            "age": 456,
            "href": "/v2/queues/FakeTestQueue/messages/578ee000508f153f256f717d",
            "id": "578ee000508f153f256f717d",
            "ttl": 3600,
            "checksum": "MD5:abf7213555626e29c3cb3e5dc58b3515"
        }
    ],
    "links": [
        {
            "href": "/v2/queues/FakeTestQueue/messages?marker=2",
            "rel": "next"
        }
    ]

}`

// ExpectedResources is the expected result in Create
var ExpectedResources = messages.ResourceList{
	Resources: []string{
		"/v2/queues/demoqueue/messages/51db6f78c508f17ddc924357",
		"/v2/queues/demoqueue/messages/51db6f78c508f17ddc924358",
	},
}

// FirstMessage is the first result in a List.
var FirstMessage = messages.Message{
	Body: map[string]interface{}{
		"current_bytes": "0",
		"event":         "BackupProgress",
		"total_bytes":   "99614720",
	},
	Age:      482,
	Href:     fmt.Sprintf("/v2/queues/%s/messages/578edfe6508f153f256f717b", QueueName),
	ID:       "578edfe6508f153f256f717b",
	TTL:      3600,
	Checksum: "MD5:abf7213555626e29c3cb3e5dc58b3515",
}

// SecondMessage is the second result in a List.
var SecondMessage = messages.Message{
	Body: map[string]interface{}{
		"current_bytes": "0",
		"event":         "BackupProgress",
		"total_bytes":   "99614720",
	},
	Age:      456,
	Href:     fmt.Sprintf("/v2/queues/%s/messages/578ee000508f153f256f717d", QueueName),
	ID:       "578ee000508f153f256f717d",
	TTL:      3600,
	Checksum: "MD5:abf7213555626e29c3cb3e5dc58b3515",
}

// ExpectedMessagesSlice is the expected result in a List.
var ExpectedMessagesSlice = [][]messages.Message{{FirstMessage}, {SecondMessage}}

// HandleCreateSuccessfully configures the test server to respond to a Create request.
func HandleCreateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc(fmt.Sprintf("/v2/queues/%s/messages", QueueName),
		func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "POST")
			th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
			th.TestJSONRequest(t, r, CreateMessageRequest)

			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, CreateMessageResponse)
		})
}

// HandleListSuccessfully configures the test server to respond to a List request.
func HandleListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc(fmt.Sprintf("/v2/queues/%s/messages", QueueName),
		func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "GET")
			th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

			w.Header().Add("Content-Type", "application/json")
			next := r.RequestURI

			switch next {
			case fmt.Sprintf("/v2/queues/%s/messages?limit=1", QueueName):
				fmt.Fprintf(w, ListMessagesResponse1)
			case fmt.Sprintf("/v2/queues/%s/messages?marker=1", QueueName):
				fmt.Fprint(w, ListMessagesResponse2)
			case fmt.Sprintf("/v2/queues/%s/messages?marker=2", QueueName):
				fmt.Fprint(w, `{ "messages": [] }`)
			}
		})
}
