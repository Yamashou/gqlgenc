package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/opentracing/opentracing-go/log"

	"github.com/Yamashou/gqlgenc/client"
	"github.com/Yamashou/gqlgenc/example/annict/gen"
)

func main() {
	key := os.Getenv("ANNICT_KEY")
	authHeader := func(req *http.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	}

	annictClient := NewAnnictClient(client.NewClient(http.DefaultClient, "https://api.annict.com/graphql", authHeader))
	ctx := context.Background()
	res, err := annictClient.GetProfile(ctx)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	fmt.Println(*res.Viewer.AvatarURL)
}

func NewAnnictClient(c *client.Client) *gen.Client {
	return &gen.Client{Client: c}
}
