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

	"github.com/google/go-cmp/cmp"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/Yamashou/gqlgenc/v3/client"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/domain"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/query"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/schema"
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

			// Compare the content of the generated file with the want file
			actualFilePath := "domain/query_gen.go"
			wantFilePath := tt.want.file

			// Read both files
			actualContent, err := os.ReadFile(actualFilePath)
			if err != nil {
				t.Errorf("error reading file (actual file): %v", err)
				return
			}

			wantContent, err := os.ReadFile(wantFilePath)
			if err != nil {
				t.Errorf("error reading file (expected file): %v", err)
				return
			}

			// Compare file contents
			if diff := cmp.Diff(string(wantContent), string(actualContent)); diff != "" {
				t.Errorf("file contents differ:\n%s", diff)
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

			// Wait for server to start
			time.Sleep(500 * time.Millisecond)

			// Client
			c := query.NewClient(client.NewClient(
				fmt.Sprintf("http://127.0.0.1:%s/graphql", port),
			))

			userOperation, err := c.UserOperation(ctx)
			if err != nil {
				t.Errorf("request failed: %v", err)
			}

			if diff := cmp.Diff(tt.want.userOperation, userOperation); diff != "" {
				t.Errorf("integrationTest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func listenAndServe(ctx context.Context, t *testing.T, port string) {
	t.Helper()
	addr := net.JoinHostPort("", port)
	srv := server(addr)
	// Graceful Shutdown
	// Receives the signal specified by argument or closes ctx.Done() when the stop function is executed
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt, os.Kill)

	// Start
	go func() {
		defer stop() // If stop is not executed at the end of this function, ListenAndServer() will continue to block at <-ctx.Done() when there is an error
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("instance error: %v", err)
		}
	}()
	// Blocks. Unblocks when ctx.Done() is closed.
	<-ctx.Done()

	// Shutdown process
	// Force termination if Shutdown does not complete in 10 seconds
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Shutdown is also executed when ListenAndServer has an error, but in that case Shutdown returns nil for err, so there is no problem.
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
	// Add new import statement to client_gen.go file
	content, err := os.ReadFile(clientGenFilePath)
	if err != nil {
		t.Errorf("error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 1 {
		t.Errorf("file has invalid format")
		return
	}

	// Find the line with package query declaration
	packageLineIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageLineIndex = i
			break
		}
	}

	if packageLineIndex == -1 {
		t.Errorf("package query declaration not found")
		return
	}

	// Add new import after the package query declaration line
	modifiedContent := append(
		lines[:packageLineIndex+1],
		append(
			[]string{"", "import \"github.com/Yamashou/gqlgenc/v3/testdata/integration/fragment/domain\""},
			lines[packageLineIndex+1:]...,
		)...,
	)

	// Write back to file
	if err := os.WriteFile(clientGenFilePath, []byte(strings.Join(modifiedContent, "\n")), 0o644); err != nil {
		t.Errorf("error writing to client_gen.go: %v", err)
	}
}

func compareFiles(t *testing.T, wantFile, generatedFile string) {
	t.Helper()

	// Compare file contents
	want, err := os.ReadFile(wantFile)
	if err != nil {
		t.Errorf("error reading file (expected file): %v", err)
		return
	}

	generated, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Errorf("error reading file (actual file): %v", err)
		return
	}

	if diff := cmp.Diff(string(want), string(generated)); diff != "" {
		t.Errorf("file contents differ:\n%s", diff)
	}
}
