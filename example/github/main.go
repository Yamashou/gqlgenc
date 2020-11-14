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
	getUser, err := githubClient.GetUser(ctx, 10, 10)
	if err != nil {
		if handledError, ok := err.(*client.ErrorResponse); ok {
			fmt.Fprintf(os.Stderr, "handled error: %s\n", handledError.Error())
		} else {
			fmt.Fprintf(os.Stderr, "unhandled error: %s\n", err.Error())
		}
		os.Exit(1)
	}

	fmt.Println(*getUser.Viewer.Name, getUser.Viewer.Repositories.Nodes[0].Name)
	for _, repository := range getUser.Viewer.Repositories.Nodes {
		fmt.Println(repository.Name)
		for _, language := range repository.Languages.Nodes {
			fmt.Println(language.Name)
		}
	}
}
