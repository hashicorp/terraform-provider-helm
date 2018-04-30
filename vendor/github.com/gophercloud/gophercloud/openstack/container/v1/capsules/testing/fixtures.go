package testing

import (
	"fmt"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
	fakeclient "github.com/gophercloud/gophercloud/testhelper/client"
)

// HandleImageGetSuccessfully test setup
func HandleCapsuleGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fakeclient.TokenID)

		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"uuid": "cc654059-1a77-47a3-bfcf-715bde5aad9e",
			"status": "Running",
			"id": 1,
			"user_id": "d33b18c384574fd2a3299447aac285f0",
			"project_id": "6b8ffef2a0ac42ee87887b9cc98bdf68",
			"cpu": 1,
			"memory": "1024M",
			"meta_name": "test",
			"meta_labels": {"web": "app"},
			"created_at": "2018-01-12 09:37:25+00:00",
			"updated_at": "2018-01-12 09:37:25+01:00",
			"links": [
				{
				  "href": "http://10.10.10.10/v1/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
				  "rel": "self"
				},
				{
				  "href": "http://10.10.10.10/capsules/cc654059-1a77-47a3-bfcf-715bde5aad9e",
				  "rel": "bookmark"
				}
			],
			"capsule_version": "beta",
			"restart_policy":  "always",
			"containers_uuids": ["1739e28a-d391-4fd9-93a5-3ba3f29a4c9b", "d1469e8d-bcbc-43fc-b163-8b9b6a740930"],
			"addresses": {
				"b1295212-64e1-471d-aa01-25ff46f9818d": [
					{
						"version": 4,
						"preserve_on_delete": false,
						"addr": "172.24.4.11",
						"port": "8439060f-381a-4386-a518-33d5a4058636",
						"subnet_id": "4a2bcd64-93ad-4436-9f48-3a7f9b267e0a"
					}
				]
			},
			"volumes_info": {
				"67618d54-dd55-4f7e-91b3-39ffb3ba7f5f": [
					"4b725a92-2197-497b-b6b1-fb8caa4cb99b"
				]
			}
		}`)
	})
}

// HandleCapsuleCreateSuccessfully creates an HTTP handler at `/capsules` on the test handler mux
// that responds with a `Create` response.
func HandleCapsuleCreateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/capsules", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fakeclient.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{}`)
	})
}
