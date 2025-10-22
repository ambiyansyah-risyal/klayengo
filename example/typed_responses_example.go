// Example demonstrating klayengo's typed response capabilities.
// This shows how to automatically unmarshal JSON responses into struct types
// instead of manually handling JSON parsing.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ambiyansyah-risyal/klayengo"
)

// User represents a user from the API
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Website  string `json:"website"`
}

// Post represents a post from the API
type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// APIResponse represents a generic API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

func main() {
	// Create a resilient HTTP client with typed response support
	client := klayengo.New(
		klayengo.WithMaxRetries(3),
		klayengo.WithInitialBackoff(100),
		klayengo.WithRateLimiter(10, 1), // 10 requests per second
		klayengo.WithCache(120),         // 2 minutes cache
		klayengo.WithSimpleLogger(),
	)

	ctx := context.Background()

	// Example 1: Using GetJSON to automatically unmarshal JSON response
	fmt.Println("=== Example 1: GetJSON - Automatic JSON Unmarshaling ===")
	var user User
	err := client.GetJSON(ctx, "https://jsonplaceholder.typicode.com/users/1", &user)
	if err != nil {
		log.Printf("GetJSON failed: %v", err)
	} else {
		fmt.Printf("User: %+v\n\n", user)
	}

	// Example 2: Using PostJSON to send and receive JSON
	fmt.Println("=== Example 2: PostJSON - Send and Receive JSON ===")
	createUserReq := CreateUserRequest{
		Name:     "John Doe",
		Username: "johndoe",
		Email:    "john@example.com",
	}
	var createdUser User
	err = client.PostJSON(ctx, "https://jsonplaceholder.typicode.com/users", createUserReq, &createdUser)
	if err != nil {
		log.Printf("PostJSON failed: %v", err)
	} else {
		fmt.Printf("Created User: %+v\n\n", createdUser)
	}

	// Example 3: Using GetTyped to get both response metadata and typed data
	fmt.Println("=== Example 3: GetTyped - Get Response with Metadata ===")
	var posts []Post
	typedResp, err := client.GetTyped(ctx, "https://jsonplaceholder.typicode.com/posts?userId=1", &posts)
	if err != nil {
		log.Printf("GetTyped failed: %v", err)
	} else {
		fmt.Printf("Status Code: %d\n", typedResp.StatusCode)
		fmt.Printf("Content-Type: %s\n", typedResp.Header.Get("Content-Type"))
		fmt.Printf("Number of Posts: %d\n", len(posts))
		if len(posts) > 0 {
			fmt.Printf("First Post: %+v\n\n", posts[0])
		}
	}

	// Example 4: Using PostTyped for complete request/response handling
	fmt.Println("=== Example 4: PostTyped - Complete POST with Metadata ===")
	newPost := Post{
		UserID: 1,
		Title:  "My New Post",
		Body:   "This is the content of my new post.",
	}
	var responsePost Post
	postResp, err := client.PostTyped(ctx, "https://jsonplaceholder.typicode.com/posts", newPost, &responsePost)
	if err != nil {
		log.Printf("PostTyped failed: %v", err)
	} else {
		fmt.Printf("Response Status: %d\n", postResp.StatusCode)
		fmt.Printf("Created Post: %+v\n\n", responsePost)
	}

	// Example 5: Error handling with typed responses
	fmt.Println("=== Example 5: Error Handling ===")
	var invalidUser User
	err = client.GetJSON(ctx, "https://jsonplaceholder.typicode.com/users/999999", &invalidUser)
	if err != nil {
		fmt.Printf("Expected error for non-existent user: %v\n", err)
	}

	// Example 6: Comparing traditional vs typed approaches
	fmt.Println("=== Example 6: Traditional vs Typed Approach Comparison ===")
	
	// Traditional approach (still supported)
	fmt.Println("Traditional approach:")
	resp, err := client.Get(ctx, "https://jsonplaceholder.typicode.com/users/2")
	if err != nil {
		log.Printf("Traditional Get failed: %v", err)
	} else {
		fmt.Printf("Status: %d, Content-Type: %s\n", resp.StatusCode, resp.Header.Get("Content-Type"))
		// Developer would need to manually read and parse JSON here
		resp.Body.Close()
	}

	// Typed approach (new feature)
	fmt.Println("Typed approach:")
	var user2 User
	err = client.GetJSON(ctx, "https://jsonplaceholder.typicode.com/users/2", &user2)
	if err != nil {
		log.Printf("Typed GetJSON failed: %v", err)
	} else {
		fmt.Printf("User directly parsed: %+v\n", user2)
	}

	fmt.Println("\n=== Typed Response Feature Benefits ===")
	fmt.Println("✓ Automatic JSON unmarshaling")
	fmt.Println("✓ Type safety at compile time")
	fmt.Println("✓ No manual JSON parsing required")
	fmt.Println("✓ Consistent error handling")
	fmt.Println("✓ All existing resilience features (retry, cache, circuit breaker, etc.)")
	fmt.Println("✓ Backward compatible with existing code")
}