/*
Package messages provides information and interaction with the messages through
the OpenStack Messaging(Zaqar) service.

Example to List Messages

	listOpts := messages.ListOpts{
		Limit: 10,
	}

	queueName := "my_queue"

	pager := messages.List(client, queueName, listOpts)

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		allMessages, err := queues.ExtractQueues(page)
		if err != nil {
			panic(err)
		}

		for _, message := range allMessages {
			fmt.Printf("%+v\n", message)
		}

		return true, nil
	})

Example to Create Messages

	createOpts := messages.CreateOpts{
		Messages:     []messages.Messages{
			{
				TTL:   300,
				Delay: 20,
				Body: map[string]interface{}{
					"event": "BackupStarted",
					"backup_id": "c378813c-3f0b-11e2-ad92-7823d2b0f3ce",
				},
			},
			{
				Body: map[string]interface{}{
					"event": "BackupProgress",
					"current_bytes": "0",
					"total_bytes": "99614720",
				},
			},
		},
	}

	queueName = "my_queue"

	resources, err := messages.Create(client, queueName, createOpts).Extract()
	if err != nil {
		panic(err)
	}
*/
package messages
