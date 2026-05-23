package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/slizhunter/gatorcli/internal/database"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	// Create an HTTP request with the provided context
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &RSSFeed{}, err
	}

	// Send the HTTP request and get the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &RSSFeed{}, err
	}
	defer resp.Body.Close()

	// Set a User-Agent header to avoid potential blocking by some servers
	resp.Header.Set("User-Agent", "gator")

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RSSFeed{}, err
	}

	// Unmarshal the XML response into the RSSFeed struct
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return &RSSFeed{}, err
	}

	// Unescape HTML entities in the feed title, description, and item titles/descriptions
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
	}

	return &feed, nil
}

// handlerAddFeed handles the "addfeed" command, which allows the current user to add a new feed by providing its name and URL.
func handlerAddFeed(s *state, cmd command, user database.User) error {
	// Syntax: addfeed <feed name> <feed URL>
	// Get the current user from the database using the username stored in the configuration
	// currentUser, _ := s.db.GetUser(context.Background(), s.config.CurrentUserName)

	// Ensure a feed URL is provided as an argument
	if len(cmd.Args) < 2 {
		return fmt.Errorf("A feed URL is required!")
	}
	// Fetch the feed data from the provided URL
	feedURL := cmd.Args[1]
	_, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Failed to fetch feed: %v", err)
	}
	// Create a new feed in the database using the fetched feed data and the current user's ID
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      cmd.Args[0],
		Url:       feedURL,
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("Failed to create feed: %v", err)
	}
	fmt.Printf("Feed added: %s\n", feed.Name)
	err = handlerFollow(s, command{Args: []string{feedURL}}, user)
	if err != nil {
		return fmt.Errorf("Failed to follow feed: %v", err)
	}
	return nil
}

// handlerGetFeeds handles the "feeds" command, which retrieves and prints the list of all feeds from the database,
// including their names, URLs, and the users who created them.
func handlerGetFeeds(s *state, cmd command) error {
	// Syntax: feeds
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to retrieve feeds: %v", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	fmt.Printf("Found %d feeds:\n", len(feeds))
	for _, feed := range feeds {
		user, err := s.db.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("Failed to retrieve user for feed %s: %v", feed.Name, err)
		}
		fmt.Printf("* %s (%s) | Created by: %s\n", feed.Name, feed.Url, user.Name)
	}
	return nil
}

// handlerAgg handles the "agg" command, which starts the feed aggregation process by periodically fetching the latest items from the feeds that users are following.
func handlerAgg(s *state, cmd command) error {
	// Syntax: agg <time_between_reqs>
	if len(cmd.Args) < 1 {
		return fmt.Errorf("Time between requests is required! Usage: %v <time_between_reqs>", cmd.Name)
	}
	timeBetweenReqs, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("Invalid duration: %v", err)
	}
	ticker := time.NewTicker(timeBetweenReqs)
	fmt.Printf("Collecting feeds every %v\n", timeBetweenReqs)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	// Syntax: browse <number_of_posts (optional, default 2)>

	// Default to showing 2 posts if no argument is provided
	if len(cmd.Args) < 1 {
		cmd.Args = append(cmd.Args, "2")
	}
	// Parse the number of posts to show from the command arguments
	numPosts, err := strconv.Atoi(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("Invalid number of posts: %v", err)
	}

	// Retrieve the latest posts from the database
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(numPosts),
	})
	if err != nil {
		return fmt.Errorf("Failed to retrieve posts: %v", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	fmt.Printf("Showing %d latest posts:\n", len(posts))
	for _, post := range posts {
		fmt.Printf("* %s | %s\n", post.FeedName, post.Title)
	}
	return nil
}

// handlerFollow handles the "follow" command, which allows the current user to follow a feed by its name.
func handlerFollow(s *state, cmd command, user database.User) error {
	// Syntax: follow <feed URL>
	// Get the current user from the database using the username stored in the configuration
	// currentUser, _ := s.db.GetUser(context.Background(), s.config.CurrentUserName)

	// Ensure a feed URL is provided as an argument
	if len(cmd.Args) < 1 {
		return fmt.Errorf("A feed URL is required!")
	}
	feedURL := cmd.Args[0]

	// Retrieve the feed from the database using the provided feed URL
	feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Failed to retrieve feed: %v", err)
	}

	// Create a new feed follow in the database using the current user's ID and the retrieved feed's ID
	followParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		FeedID:    feed.ID,
		UserID:    user.ID,
	}
	_, err = s.db.CreateFeedFollow(context.Background(), followParams)
	if err != nil {
		return fmt.Errorf("Failed to follow feed: %v", err)
	}
	fmt.Printf("Now following feed: %s\n", feed.Name)
	return nil
}

// handlerUnfollow handles the "unfollow" command, which allows the current user to unfollow a feed by its name.
func handlerUnfollow(s *state, cmd command, user database.User) error {
	// Syntax: unfollow <feed URL>
	// Get the current user from the database using the username stored in the configuration

	// Ensure a feed URL is provided as an argument
	if len(cmd.Args) < 1 {
		return fmt.Errorf("A feed URL is required!")
	}
	feedURL := cmd.Args[0]

	// Retrieve the feed from the database using the provided feed URL
	feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Failed to retrieve feed: %v", err)
	}

	// Delete the feed follow from the database using the current user's ID and the retrieved feed's ID
	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		FeedID: feed.ID,
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("Failed to unfollow feed: %v", err)
	}
	fmt.Printf("%s unfollowed feed: %s\n", user.Name, feed.Name)
	return nil
}

// handlerFollowing handles the "following" command, which retrieves and prints the list of feeds that the current user is following.
func handlerFollowing(s *state, cmd command, user database.User) error {
	// Syntax: following
	// Get the current user from the database using the username stored in the configuration
	// currentUser, _ := s.db.GetUser(context.Background(), s.config.CurrentUserName)

	// Retrieve the list of feeds that the current user is following from the database
	following, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("Failed to retrieve following feeds: %v", err)
	}

	if len(following) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}

	fmt.Printf("You are following %d feeds:\n", len(following))
	for _, feed := range following {
		fmt.Printf("* %s\n", feed.FeedName)
	}
	return nil
}

func scrapeFeeds(s *state) {
	// This function will be responsible for periodically fetching the latest items from the feeds that users are following
	feedToFetch, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		log.Printf("Failed to get next feed to fetch: %v", err)
		return
	}
	fmt.Printf("Fetching feed: %s\n", feedToFetch.Name)
	_, err = s.db.MarkFeedFetched(context.Background(), feedToFetch.ID)
	if err != nil {
		log.Printf("Failed to mark feed as fetched: %v", err)
		return
	}
	feed, err := fetchFeed(context.Background(), feedToFetch.Url)
	if err != nil {
		log.Printf("Failed to fetch feed: %v", err)
		return
	}
	log.Printf("Feed %s collected, %v posts found", feed.Channel.Title, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		//fmt.Printf("New post: %s\n", item.Title)
		_, err := s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: sql.NullTime{Time: parseTime(item.PubDate), Valid: item.PubDate != ""},
			FeedID:      feedToFetch.ID,
		})
		if err != nil {
			if err.(*pq.Error).Code == "23505" {
				// Duplicate URL, ignore
			} else {
				log.Printf("Failed to create post: %v", err)
			}
		}
		//log.Printf("Post created: %s", post.Title)
	}
}

// parseTime attempts to parse a time string using multiple common RSS date formats, returning the parsed time or the current time if parsing fails.
func parseTime(timeStr string) time.Time {
	// Common RSS date formats to try
	layouts := []string{
		time.RFC1123Z, // Example: "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,  // Example: "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,  // Example: "02 Jan 06 15:04 -0700"
		time.RFC822,   // Example: "02 Jan 06 15:04 MST"
		time.RFC850,   // Example: "Monday, 02-Jan-06 15:04:05 MST"
		time.ANSIC,    // Example: "Mon Jan _2 15:04:05 2006"
	}
	// Try parsing the time string with each layout until one succeeds
	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t
		}
	}
	log.Printf("Failed to parse time: %s", timeStr)
	return sql.NullTime{Valid: false}.Time
}
