package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Yamashou/gqlgenc/v3/client"
	"github.com/Yamashou/gqlgenc/v3/example/input-scalar/gen"
)

func main() {
	c := client.Client{
		Client:             http.DefaultClient,
		BaseURL:            "http://localhost:8080/query",
		RequestInterceptor: client.ChainInterceptor(),
		CustomDo: func(ctx context.Context, req *http.Request, gqlInfo *client.GQLRequestInfo, res any) error {
			fmt.Println("Do request")
			r, err := io.ReadAll(req.Body)
			if err != nil {
				return err
			}
			// ex: {"operationName":"GetNumber","query":"query GetNumber ($number: Number!) {\n\tenumToNum(number: $number)\n}\n","variables":{"number":"ONE"}}
			fmt.Println(string(r))

			return nil
		},
	}

	client := gen.Client{
		Client: &c,
	}

	_, err := client.GetNumber(context.Background(), 1)
	if err != nil {
		fmt.Println(err)

		return
	}
}
