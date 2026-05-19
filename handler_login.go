package main

import (
	"context"
	"fmt"
)

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
