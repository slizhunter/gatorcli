!!Postgres and Go must be installed to run the program!!

Install the gatorCLI using:

```bash
go install github.com/slizhunter/gatorcli@latest
```

This command downloads and compiles the latest version of gatorCLI, then installs the binary to your `$GOPATH/bin` directory (or `$HOME/go/bin` by default). Make sure this directory is in your `$PATH` to run `gatorcli` from anywhere.
_________________________________________________

How to set up the config file:

Create a `.gatorconfig.json` file in your home directory. The config file should contain your database connection URL:

```json
{
  "db_url": "postgres://username:password@localhost:5432/database_name"
}
```

- `db_url`: PostgreSQL connection string (required)

The config file will be read from your home directory when gatorCLI runs.
________________________________________________

How to run the program:

Once installed and configured, run gatorCLI commands from your terminal:

```bash
gatorcli <command> [arguments]
```

**Available commands:**

- `login <username>` - Log in as an existing user
- `register <username>` - Create a new user account
- `users` - List all users
- `agg` - Aggregate RSS feeds (requires login)
- `addfeed <feed_name> <feed_url>` - Add an RSS feed to follow (requires login)
- `feeds` - List all available feeds
- `follow <feed_name>` - Follow a feed (requires login)
- `following` - Show feeds you're following (requires login)
- `unfollow <feed_name>` - Unfollow a feed (requires login)
- `browse` - Browse feed posts (requires login)

**Example:**

```bash
gatorcli register john_doe
gatorcli login john_doe
gatorcli addfeed "Tech News" "https://example.com/feed.xml"
gatorcli feeds
gatorcli browse
```
