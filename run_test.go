package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/Yamashou/gqlgenc/v3/client"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/basic/domain"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/basic/query"
	"github.com/Yamashou/gqlgenc/v3/testdata/integration/basic/schema"
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
			name:    "basic test",
			testDir: "testdata/integration/basic/",
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
						Name2:         "John Doe",
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
						Profile2: domain.UserOperation_User_Profile2{
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
			t.Chdir(tt.testDir)
			if err := run(); err != nil {
				t.Errorf("run() error = %v", err)
			}

			// Compare the content of the generated file with the want file
			actualFilePath := "domain/query_gen.go"
			wantFilePath := tt.want.file
			compareFiles(t, wantFilePath, actualFilePath)

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
			time.Sleep(50 * time.Millisecond)

			// Client
			c := query.NewClient(client.NewClient(
				fmt.Sprintf("http://localhost:%s/graphql", port),
			))

			// Query
			{
				userOperation, err := c.UserOperation(ctx)
				if err != nil {
					t.Errorf("request failed: %v", err)
				}
				if diff := cmp.Diff(tt.want.userOperation, userOperation); diff != "" {
					t.Errorf("integrationTest mismatch (-want +got):\n%s", diff)
				}
			}

			// Mutation
			{
				input := domain.UpdateUserInput{
					ID:   "1",
					Name: graphql.OmittableOf[*string](nil),
				}
				updateUser, err := c.UpdateUser(ctx, input)
				if err != nil {
					t.Errorf("request failed: %v", err)
				}
				if updateUser.GetUpdateUser().User.Name != "nil" {
					t.Errorf("expected name to be 'nil', got '%s'", updateUser.GetUpdateUser().User.Name)
				}
			}
			{
				input := domain.UpdateUserInput{
					ID:   "1",
					Name: graphql.Omittable[*string]{},
				}
				updateUser, err := c.UpdateUser(ctx, input)
				if err != nil {
					t.Errorf("request failed: %v", err)
				}
				if updateUser.GetUpdateUser().User.Name != "undefined" {
					t.Errorf("expected name to be 'undefined', got '%s'", updateUser.GetUpdateUser().User.Name)
				}
			}
			{
				input := domain.UpdateUserInput{
					ID:   "1",
					Name: graphql.OmittableOf[*string](ptr("Sam Smith")),
				}
				updateUser, err := c.UpdateUser(ctx, input)
				if err != nil {
					t.Errorf("request failed: %v", err)
				}
				if updateUser.GetUpdateUser().User.Name != "Sam Smith" {
					t.Errorf("expected name to be 'Sam Smith', got '%s'", updateUser.GetUpdateUser().User.Name)
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
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
