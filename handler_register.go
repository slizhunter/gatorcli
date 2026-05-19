package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/slizhunter/gatorcli/internal/database"
)

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
