package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/slizhunter/gatorcli/internal/config"
	"github.com/slizhunter/gatorcli/internal/database"
)

func main() {
	// Read configuration
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	//fmt.Printf("Read config: %+v\n", cfg)

	// Connect to the database
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize database queries
	dbQueries := database.New(db)

	// Initialize application state
	programState := &state{
		db:     dbQueries,
		config: &cfg,
	}

	// Register commands
	cmds := &commands{commandMap: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerGetFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))

	// Parse and execute command
	if len(os.Args) < 2 {
		log.Fatalf("No command provided")
	}
	// The first argument is the command name, and the rest are its arguments
	if err := cmds.run(programState, command{Name: os.Args[1], Args: os.Args[2:]}); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}
