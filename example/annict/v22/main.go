package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Yamashou/gqlgenc/clientV2"

	"github.com/Yamashou/gqlgenc/example/annict/gen"
)

func main() {
	key := os.Getenv("ANNICT_KEY")

	ctx := context.Background()
	annictClientV2 := clientV2.NewClient(http.DefaultClient, "https://api.annict.com/graphql", func(ctx context.Context, req *http.Request, gqlInfo *clientV2.GQLRequestInfo, res interface{}, next clientV2.RequestInterceptorFunc) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
		return next(ctx, req, gqlInfo, res)
	}, func(ctx context.Context, req *http.Request, gqlInfo *clientV2.GQLRequestInfo, res interface{}, next clientV2.RequestInterceptorFunc) error {
		err := next(ctx, req, gqlInfo, res)
		if err != nil {
			return err
		}
		return nil
	})
	vars := map[string]interface{}{}

	var res gen.GetProfile
	if err := annictClientV2.Post(ctx, "GetProfile", gen.GetProfileQuery, &res, vars); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Println(*res.Viewer.AvatarURL, res.Viewer.RecordsCount, res.Viewer.WatchedCount)
}
