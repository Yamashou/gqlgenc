package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Yamashou/gqlgenc/client"
	"github.com/Yamashou/gqlgenc/example/github/gen"
)

func main() {
	// This example only read public repository. You don't need to select scopes.
	token := os.Getenv("GITHUB_TOKEN")
	authHeader := func(req *http.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	ctx := context.Background()

	githubClient := &gen.Client{
		Client: client.NewClient(http.DefaultClient, "https://api.github.com/graphql", authHeader),
	}
	getUser, err := githubClient.GetUser(ctx, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unhandled error: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("User: %s, CreatedAt: %s\n", *getUser.Viewer.Name, getUser.Viewer.CreatedAt.String())
	for _, repository := range getUser.Viewer.Repositories.Nodes {
		fmt.Printf("Repository: %s\n", repository.Name)
	}
}
