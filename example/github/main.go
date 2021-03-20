package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Yamashou/gqlgenc/client"
	"github.com/Yamashou/gqlgenc/clientv2"
	"github.com/Yamashou/gqlgenc/example/github/gen"
)

func main() {
	// This example only read public repository. You don't need to select scopes.
	token := os.Getenv("GITHUB_TOKEN")
	githubClient := gen.NewClient(http.DefaultClient, "https://api.github.com/graphql", func(ctx context.Context, req *http.Request, gqlInfo *clientv2.GQLRequestInfo, res interface{}, next clientv2.RequestInterceptorFunc) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		return next(ctx, req, gqlInfo, res)
	})

	ctx := context.Background()
	getUser, err := githubClient.GetUser(ctx, 10, 10)
	if err != nil {
		if handledError, ok := err.(*client.ErrorResponse); ok {
			fmt.Fprintf(os.Stderr, "handled error: %s\n", handledError.Error())
		} else {
			fmt.Fprintf(os.Stderr, "unhandled error: %s\n", err.Error())
		}
		os.Exit(1)
	}

	fmt.Println(*getUser.Viewer.Name, getUser.Viewer.Repositories.Nodes[0].Name, *getUser.Viewer.Name, *getUser.Viewer.Repositories.Nodes[0].Typename)
	for _, repository := range getUser.Viewer.Repositories.Nodes {
		getRepo, err := githubClient.GetRepo(ctx, repository.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unhandled error: %s\n", err.Error())
			os.Exit(1)
		}

		fmt.Println(getRepo.Node.Repository.Name, *getRepo.Node.Typename)
	}
}
