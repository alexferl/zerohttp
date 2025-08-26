package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	zh "github.com/alexferl/zerohttp"
)

// User represents a user in our system
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UserStore handles CRUD operations for users using an in-memory map
type UserStore struct {
	mu     sync.RWMutex
	users  map[int]User
	nextID int
}

func NewUserStore() *UserStore {
	return &UserStore{
		users:  make(map[int]User),
		nextID: 1,
	}
}

// GetUser handles GET /users/{id}
func (s *UserStore) GetUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid user ID"))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(404, "User not found"))
	}

	return zh.R.JSON(w, 200, user)
}

// ListUsers handles GET /users
func (s *UserStore) ListUsers(w http.ResponseWriter, r *http.Request) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return zh.R.JSON(w, 200, zh.M{"users": users, "count": len(users)})
}

// CreateUser handles POST /users
func (s *UserStore) CreateUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	if err := zh.B.JSON(r.Body, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid JSON"))
	}

	if req.Name == "" {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Name is required"))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := User{
		ID:   s.nextID,
		Name: req.Name,
		Age:  req.Age,
	}
	s.users[s.nextID] = user
	s.nextID++

	return zh.R.JSON(w, 201, user)
}

// UpdateUser handles PUT /users/{id}
func (s *UserStore) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid user ID"))
	}

	var req struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	if err := zh.B.JSON(r.Body, &req); err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid JSON"))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(404, "User not found"))
	}

	// Update fields if provided
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Age > 0 {
		user.Age = req.Age
	}

	s.users[userID] = user
	return zh.R.JSON(w, 200, user)
}

// DeleteUser handles DELETE /users/{id}
func (s *UserStore) DeleteUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid user ID"))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[userID]; !exists {
		return zh.R.ProblemDetail(w, zh.NewProblemDetail(404, "User not found"))
	}

	delete(s.users, userID)
	return zh.R.JSON(w, 200, zh.M{"message": "User deleted successfully"})
}

func main() {
	// Create the user store
	store := NewUserStore()

	// Add some sample users
	store.users[1] = User{ID: 1, Name: "Alice", Age: 30}
	store.users[2] = User{ID: 2, Name: "Bob", Age: 25}
	store.nextID = 3

	// Create the app
	app := zh.New()

	// Register CRUD routes
	app.GET("/users", zh.HandlerFunc(store.ListUsers))
	app.GET("/users/{id}", zh.HandlerFunc(store.GetUser))
	app.POST("/users", zh.HandlerFunc(store.CreateUser))
	app.PUT("/users/{id}", zh.HandlerFunc(store.UpdateUser))
	app.DELETE("/users/{id}", zh.HandlerFunc(store.DeleteUser))

	// Health check
	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"status": "ok"})
	}))

	fmt.Println("Try these endpoints:")
	fmt.Println("  GET    /users      - List all users")
	fmt.Println("  GET    /users/1    - Get user by ID")
	fmt.Println("  POST   /users      - Create user")
	fmt.Println("  PUT    /users/1    - Update user")
	fmt.Println("  DELETE /users/1    - Delete user")
	fmt.Println("  GET    /health     - Health check")

	log.Fatal(app.Start())
}
