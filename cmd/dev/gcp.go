package main

import (
	"context"
	"fmt"
	"log"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"google.golang.org/api/iterator"
)

func listServiceAccounts() error {
	ctx := context.Background()
	c, err := admin.NewIamClient(ctx)
	if err != nil {
		return err
	}

	req := &adminpb.ListServiceAccountsRequest{
		Name: fmt.Sprintf("projects/%s", "cf-dev-368022"),
	}
	it := c.ListServiceAccounts(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		// TODO: Use resp.
		fmt.Printf("\nservice account: %s\n", resp.GetName())
	}
	return nil
}

func main() {

	err := listServiceAccounts()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

}
