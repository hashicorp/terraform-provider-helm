package capsules

import "github.com/gophercloud/gophercloud"

func getURL(client *gophercloud.ServiceClient, id string) string {
	return client.ServiceURL("capsules", id)
}

func createURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("capsules")
}
