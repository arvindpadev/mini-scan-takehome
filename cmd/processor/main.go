package main

import (
	"context"
	"flag"

	"cloud.google.com/go/pubsub"
	"github.com/censys/scan-takehome/pkg/processor"
)

func main() {
	projectId := flag.String("project", "test-project", "GCP Project ID")
	subName := flag.String("subscription", "scan-sub", "GCP PubSub Subscription ID")
	instanceId := flag.String("instance", "test-instance", "GCP Bigtable Instance ID")

	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, *projectId)
	if err != nil {
		panic(err)
	}

	sub := client.Subscription(*subName)
	store := processor.NewBigTableStore(projectId, instanceId)
	processor.Listen(sub, store)
}
