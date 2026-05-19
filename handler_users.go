package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/slizhunter/gatorcli/internal/database"
)

// handlerRegister handles the "register" command, which creates a new user in the database and sets it as the current user in the configuration.
func handlerRegister(s *state, cmd command) error {
	// Ensure a username is provided as an argument
	if len(cmd.Args) < 1 {
		return fmt.Errorf("A username is required!")
	}

	// Create a new user in the database
	params := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
	}

	// Call the CreateUser method to insert the new user into the database
	user, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Failed to create user: %v", err)
	}
	fmt.Printf("User created: %+v\n", user)
	// Set the current user in the configuration
	s.config.SetUser(cmd.Args[0])
	return nil
}

// handlerLogin handles the "login" command, allowing the user to set their username in the configuration.
func handlerLogin(s *state, cmd command) error {
	// Ensure a username is provided as an argument
	if len(cmd.Args) < 1 {
		return fmt.Errorf("A username is required!")
	}

	// Check if the user exists in the database
	user, err := s.db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		return fmt.Errorf("User %v doesn't exist!", cmd.Args[0])
	}
	fmt.Printf("User found: %+v\n", user)

	// Set the current user in the configuration
	s.config.SetUser(cmd.Args[0])
	fmt.Printf("Current user: %v\n", cmd.Args[0])
	return nil
}

// handlerUsers handles the "users" command, which retrieves and prints the list of all users from the database, indicating the current user.
func handlerUsers(s *state, cmd command) error {
	// Retrieve all users from the database
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to retrieve users: %v", err)
	}

	// Print the list of users
	current := ""
	fmt.Println("Users:")
	for _, user := range users {
		if user.Name == s.config.CurrentUserName {
			current = "(current)"
		}
		fmt.Printf("* %v %v\n", user.Name, current)
		current = ""
	}
	return nil
}
