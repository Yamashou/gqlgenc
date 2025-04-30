package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/query"
	"github.com/vektah/gqlparser/v2/ast"
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
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/schema"
	"github.com/google/go-cmp/cmp"
)

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
			srv.AddTransport(transport.Websocket{
				KeepAlivePingInterval: 10 * time.Second,
			})
			srv.AddTransport(transport.Options{})
			srv.AddTransport(transport.GET{})
			srv.AddTransport(transport.POST{})
			srv.AddTransport(transport.MultipartForm{})

			srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

			srv.Use(extension.Introspection{})
			srv.Use(extension.AutomaticPersistedQuery{
				Cache: lru.New[string](100),
			})
			http.Handle("/graphql", srv)
			port := "8080"
			go func() {
				listenAndServe(ctx, t, port)
			}()

			// Client
			c := query.NewClient(client.NewClient(fmt.Sprintf("http://127.0.0.1:%s/graphql", port)))
			userOperation, err := c.UserOperation(ctx)
			if err != nil {
				t.Errorf("failed to post request: %v", err)
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
