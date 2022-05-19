package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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

	getProfile, err := annictClient.GetProfile(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println(*getProfile.Viewer.AvatarURL, getProfile.Viewer.RecordsCount, getProfile.Viewer.WatchedCount)

	list, err := annictClient.SearchWorks(ctx, []string{"2017-spring"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	for _, node := range list.SearchWorks.Nodes {
		fmt.Println(node.ID, node.AnnictID, node.Title, *node.Work.Image.RecommendedImageURL)
	}

	getWork, err := annictClient.GetWork(ctx, []int64{list.SearchWorks.Nodes[0].AnnictID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	work := getWork.SearchWorks.Nodes[0]
	_, err = annictClient.UpdateWorkStatus(ctx, work.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	_, err = annictClient.CreateRecordMutation(ctx, work.Episodes.Nodes[0].ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	getProfile2, err := annictClient.GetProfile(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println(getProfile2.Viewer.RecordsCount, getProfile2.Viewer.WatchedCount)

	res, err := annictClient.ListWorks(ctx, nil, nil, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println(res.Viewer.Works.Edges[0].Node.Title, res.Viewer.Works.Edges[0].Cursor, len(res.Viewer.Works.Edges))
}

func NewAnnictClient(c *client.Client) *gen.Client {
	return &gen.Client{Client: c}
}
