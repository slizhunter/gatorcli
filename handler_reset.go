package main

import (
	"context"
	"fmt"
)

// handlerReset handles the "reset" command, which deletes all users from the database and clears the current user from the configuration.
func handlerReset(s *state, cmd command) error {
	// Delete all users from the database
	if err := s.db.DeleteUsers(context.Background()); err != nil {
		return fmt.Errorf("Failed to delete users: %v", err)
	}
	fmt.Println("All users have been deleted.")

	// Clear the current user from the configuration
	s.config.SetUser("")
	fmt.Println("Current user has been cleared.")
	return nil
}
