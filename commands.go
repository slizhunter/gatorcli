package main

import (
	"fmt"

	"github.com/slizhunter/gatorcli/internal/config"
	"github.com/slizhunter/gatorcli/internal/database"
)

type state struct {
	db     *database.Queries
	config *config.Config
}

// command represents a user command with its name and arguments
type command struct {
	Name string
	Args []string
}

type commands struct {
	commandMap map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandMap[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.commandMap[cmd.Name]
	if !exists {
		return fmt.Errorf("Unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}
