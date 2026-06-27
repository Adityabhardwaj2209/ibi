package main

import (
	"log"
	"net/http"
)

func main() {
	// Initialize the database store
	store := NewStore()

	// Register API endpoints
	http.HandleFunc("/register", handleRegister(store))
	http.HandleFunc("/login", handleLogin(store))
	http.HandleFunc("/submitComplaint", handleSubmitComplaint(store))
	http.HandleFunc("/getAllComplaintsForUser", handleGetAllComplaintsForUser(store))
	http.HandleFunc("/getAllComplaintsForAdmin", handleGetAllComplaintsForAdmin(store))
	http.HandleFunc("/viewComplaint", handleViewComplaint(store))
	http.HandleFunc("/resolveComplaint", handleResolveComplaint(store))

	// Start server on port 8080
	log.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
