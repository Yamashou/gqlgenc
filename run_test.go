package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/Yamashou/gqlgenc/v3/client"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/domain"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/query"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/schema"
	"github.com/google/go-cmp/cmp"
)

// テスト用のカスタムトランスポート
type customRoundTripper struct {
	base http.RoundTripper
}

func (c *customRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// TODO: これなしで動くようにする
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.base.RoundTrip(req)
}

func Test_IntegrationTest(t *testing.T) {
	type want struct {
		file          string
		userOperation *domain.UserOperation
	}
	tests := []struct {
		name    string
		testDir string
		want    want
	}{
		{
			name:    "fragment test",
			testDir: "testdata/integration/fragment/",
			want: want{
				file: "./want/query_gen.go.txt",
				userOperation: &domain.UserOperation{
					OptionalUser: &domain.UserOperation_OptionalUser{
						Name: "Sam Smith",
					},
					User: domain.UserOperation_User{
						User: struct {
							domain.UserFragment2
							Name string "json:\"name,omitempty,omitzero\" graphql:\"name\""
						}{
							UserFragment2: domain.UserFragment2{Name: "John Doe"},
							Name:          "John Doe",
						},
						UserFragment1: domain.UserFragment1{
							User: struct {
								Name string "json:\"name,omitempty,omitzero\" graphql:\"name\""
							}{
								Name: "John Doe",
							},
							Name: "John Doe",
							Profile: domain.UserFragment1_Profile{
								PrivateProfile: struct {
									Age *int "json:\"age\" graphql:\"age\""
								}{
									Age: func() *int { i := 30; return &i }(),
								},
							},
						},
						UserFragment2: domain.UserFragment2{Name: "John Doe"},
						Name:          "John Doe",
						Address: domain.UserOperation_User_Address{
							Street: "123 Main St",
							PrivateAddress: struct {
								Private bool   "json:\"private,omitempty,omitzero\" graphql:\"private\""
								Street  string "json:\"street,omitempty,omitzero\" graphql:\"street\""
							}{
								Street: "123 Main St",
							},
							PublicAddress: struct {
								Public bool   "json:\"public,omitempty,omitzero\" graphql:\"public\""
								Street string "json:\"street,omitempty,omitzero\" graphql:\"street\""
							}{
								Street: "123 Main St",
							},
						},
						Profile: domain.UserOperation_User_Profile{
							PrivateProfile: struct {
								Age *int "json:\"age\" graphql:\"age\""
							}{
								Age: func() *int { i := 30; return &i }(),
							},
						},
						OptionalProfile: &domain.UserOperation_User_OptionalProfile{
							PublicProfile: struct {
								Status domain.Status "json:\"status,omitempty,omitzero\" graphql:\"status\""
							}{
								Status: domain.StatusActive,
							},
						},
						OptionalAddress: &domain.UserOperation_User_OptionalAddress{
							Street: "456 Elm St",
							PrivateAddress: struct {
								Private bool   "json:\"private,omitempty,omitzero\" graphql:\"private\""
								Street  string "json:\"street,omitempty,omitzero\" graphql:\"street\""
							}{
								Street: "456 Elm St",
							},
							PublicAddress: struct {
								Public bool   "json:\"public,omitempty,omitzero\" graphql:\"public\""
								Street string "json:\"street,omitempty,omitzero\" graphql:\"street\""
							}{
								Street: "456 Elm St",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic: %v", r)
				}
			}()

			////////////////////////////////////////////////////////////////////////////////////////////////////////////
			// Query and client generation
			if err := os.Chdir(tt.testDir); err != nil {
				t.Errorf("run() error = %v", err)
			}
			if err := run(); err != nil {
				t.Errorf("run() error = %v", err)
			}

			// 生成されたファイルとwantファイルの内容を比較
			actualFilePath := "domain/query_gen.go"
			wantFilePath := tt.want.file

			// 両方のファイルを読み込む
			actualContent, err := os.ReadFile(actualFilePath)
			if err != nil {
				t.Errorf("ファイル読み込みエラー（実際のファイル）: %v", err)
				return
			}

			wantContent, err := os.ReadFile(wantFilePath)
			if err != nil {
				t.Errorf("ファイル読み込みエラー（期待されるファイル）: %v", err)
				return
			}

			// ファイルの内容を比較
			if diff := cmp.Diff(string(wantContent), string(actualContent)); diff != "" {
				t.Errorf("ファイルの内容が異なります:\n%s", diff)
			}
			addImport(t, "query/client_gen.go")

			////////////////////////////////////////////////////////////////////////////////////////////////////////////
			// send request test
			ctx := t.Context()

			// Server
			es := schema.NewExecutableSchema(schema.Config{Resolvers: &schema.Resolver{}})
			srv := handler.New(es)
			srv.AddTransport(transport.POST{})
			http.Handle("/graphql", srv)
			port := "8080"
			go func() {
				listenAndServe(ctx, t, port)
			}()

			// サーバーが起動するのを少し待つ
			time.Sleep(500 * time.Millisecond)

			// カスタムトランスポートでContent-Typeヘッダーを設定
			httpClient := &http.Client{
				Timeout: time.Second * 5,
				Transport: &customRoundTripper{
					base: http.DefaultTransport,
				},
			}

			// Client
			c := query.NewClient(client.NewClient(
				fmt.Sprintf("http://127.0.0.1:%s/graphql", port),
				client.WithHTTPClient(httpClient),
			))

			userOperation, err := c.UserOperation(ctx)
			if err != nil {
				t.Errorf("リクエスト失敗: %v", err)
			}

			if diff := cmp.Diff(tt.want.userOperation, userOperation); diff != "" {
				t.Errorf("UserOperation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func listenAndServe(ctx context.Context, t *testing.T, port string) {
	t.Helper()
	addr := net.JoinHostPort("", port)
	srv := server(addr)
	// Graceful Shutdown
	// 引数で指定したSignalを受け取る or stop関数を実行するとctx.Done()をCloseする
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt, os.Kill)

	// Start
	go func() {
		defer stop() // この関数の最後でstopを実行しないと、ListenAndServer()がエラーのときに<-ctx.Done()でブロックし続けてしまう
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Instance error: %v", err)
		}
	}()
	// ブロックする。ctx.Done()をCloseするとブロックを解除する。
	<-ctx.Done()

	// Shutdown処理
	// Shutdownが10秒で終わらなかったら強制終了する
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Shutdownは、ListenAndServerがエラーの時も実行されるが、そのときShutdownはerrをnilで返すため問題ない。
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func server(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		ReadTimeout:       2 * time.Minute,
		ReadHeaderTimeout: 1 * time.Minute,
		WriteTimeout:      60 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
}

func addImport(t *testing.T, clientGenFilePath string) {
	t.Helper()
	// client_gen.goファイルに新しいimport文を追加
	content, err := os.ReadFile(clientGenFilePath)
	if err != nil {
		t.Errorf("読み込みエラー: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 1 {
		t.Errorf("ファイルが不正な形式です")
		return
	}

	// package query宣言の行を探す
	packageLineIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageLineIndex = i
			break
		}
	}

	if packageLineIndex == -1 {
		t.Errorf("package query宣言が見つかりません")
		return
	}

	// package query宣言の次の行に新しいimportを追加
	modifiedContent := append(
		lines[:packageLineIndex+1],
		append(
			[]string{"", "import \"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/domain\""},
			lines[packageLineIndex+1:]...,
		)...,
	)

	// ファイルに書き戻す
	if err := os.WriteFile(clientGenFilePath, []byte(strings.Join(modifiedContent, "\n")), 0644); err != nil {
		t.Errorf("client_gen.go書き込みエラー: %v", err)
		return
	}

}
