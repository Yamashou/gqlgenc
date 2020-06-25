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

	list, err := annictClient.SearchWorks(ctx, []string{"2017-spring"})
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	for _, node := range list.SearchWorks.Nodes {
		fmt.Println(node.ID, node.AnnictID, node.Title, *node.Work.Image.RecommendedImageURL)
	}

	getWork, err := annictClient.GetWork(ctx, []int64{list.SearchWorks.Nodes[0].AnnictID})
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	for _, node := range getWork.SearchWorks.Nodes {
		for _, e := range node.Episodes.Nodes {
			fmt.Println(e.ID, e.AnnictID, *e.Title, e.AnnictID, e.SortNumber)
		}
	}
}

func NewAnnictClient(c *client.Client) *gen.Client {
	return &gen.Client{Client: c}
}
