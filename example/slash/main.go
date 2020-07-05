package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Yamashou/gqlgenc/client"
	"github.com/Yamashou/gqlgenc/example/slash/gen"
)

func main() {
	token := os.Getenv("SLASH_TOKEN")
	endpoint := os.Getenv("SLASH_ENDPOINT")
	authHeader := func(req *http.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	ctx := context.Background()

	slashClient := &gen.Client{
		Client: client.NewClient(http.DefaultClient, endpoint, authHeader),
	}

	name := "yooo"

	user := &gen.AddUserInput{
		Username: "Yamada",
		Name:     &name,
		Tasks:    nil,
	}
	addUsers, err := slashClient.AddUsersMutation(ctx, []*gen.AddUserInput{user})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println(addUsers.AddUser.User[0].Username)

	task := &gen.AddTaskInput{
		Title:     "Homework",
		Completed: false,
		User:      &gen.UserRef{Username: &addUsers.AddUser.User[0].Username, Name: addUsers.AddUser.User[0].Name},
	}
	addTasks, err := slashClient.AddTasks(ctx, []*gen.AddTaskInput{task})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println(addTasks.AddTask.Task[0].ID, addTasks.AddTask.Task[0].Title)

	_, err = slashClient.DeleteTaskMutation(ctx, addTasks.AddTask.Task[0].ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	_, err = slashClient.DeleteUserMutation(ctx, addUsers.AddUser.User[0].Username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Done!!!")
}
