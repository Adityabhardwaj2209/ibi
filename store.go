package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

// Store implements a concurrency-safe in-memory data store.
type Store struct {
	mu         sync.RWMutex
	users      map[string]*User      // Map of User ID -> User
	codeToUser map[string]*User      // Map of Secret Code -> User
	complaints map[string]*Complaint // Map of Complaint ID -> Complaint
}

// NewStore initializes and returns a new Store.
func NewStore() *Store {
	return &Store{
		users:      make(map[string]*User),
		codeToUser: make(map[string]*User),
		complaints: make(map[string]*Complaint),
	}
}

// generateRandomHex generates a random hex string of n bytes.
func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateUser creates a new user, generates unique ID and secret code, and stores them.
func (s *Store) CreateUser(name, email string, isAdmin bool) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate inputs
	if name == "" {
		return nil, errors.New("name is required")
	}
	if email == "" {
		return nil, errors.New("email address is required")
	}

	var userID, secretCode string
	var err error

	// Ensure unique ID
	for {
		userID, err = generateRandomHex(8) // 16 hex chars
		if err != nil {
			return nil, err
		}
		if _, exists := s.users[userID]; !exists {
			break
		}
	}

	// Ensure unique secret code
	for {
		secretCode, err = generateRandomHex(16) // 32 hex chars
		if err != nil {
			return nil, err
		}
		if _, exists := s.codeToUser[secretCode]; !exists {
			break
		}
	}

	newUser := &User{
		ID:         userID,
		SecretCode: secretCode,
		Name:       name,
		Email:      email,
		Complaints: make([]*Complaint, 0),
		IsAdmin:    isAdmin,
	}

	s.users[userID] = newUser
	s.codeToUser[secretCode] = newUser

	return newUser, nil
}

// GetUserBySecretCode retrieves a user by their secret code.
func (s *Store) GetUserBySecretCode(secretCode string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.codeToUser[secretCode]
	return user, exists
}

// GetUserByID retrieves a user by their unique ID.
func (s *Store) GetUserByID(userID string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	return user, exists
}

// CreateComplaint creates a complaint, associates it with the submitting user, and stores it.
func (s *Store) CreateComplaint(userID, title, summary string, severity int) (*Complaint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Retrieve user
	user, exists := s.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Validate inputs
	if title == "" {
		return nil, errors.New("complaint title is required")
	}
	if summary == "" {
		return nil, errors.New("complaint summary is required")
	}
	if severity < 1 || severity > 5 {
		return nil, errors.New("severity rating must be between 1 and 5")
	}

	var complaintID string
	var err error

	// Ensure unique ID
	for {
		complaintID, err = generateRandomHex(8) // 16 hex chars
		if err != nil {
			return nil, err
		}
		if _, exists := s.complaints[complaintID]; !exists {
			break
		}
	}

	newComplaint := &Complaint{
		ID:            complaintID,
		Title:         title,
		Summary:       summary,
		Severity:      severity,
		Status:        "Pending",
		UserID:        userID,
		SubmitterName: user.Name,
	}

	s.complaints[complaintID] = newComplaint
	user.Complaints = append(user.Complaints, newComplaint)

	return newComplaint, nil
}

// GetComplaintsForUser returns all complaints submitted by a user.
func (s *Store) GetComplaintsForUser(userID string) ([]*Complaint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Return a copy of the slice to avoid race conditions
	complaints := make([]*Complaint, len(user.Complaints))
	copy(complaints, user.Complaints)
	return complaints, nil
}

// GetAllComplaints returns all complaints in the store.
func (s *Store) GetAllComplaints() []*Complaint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Complaint, 0, len(s.complaints))
	for _, c := range s.complaints {
		list = append(list, c)
	}
	return list
}

// GetComplaintByID retrieves a complaint by its ID.
func (s *Store) GetComplaintByID(complaintID string) (*Complaint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	complaint, exists := s.complaints[complaintID]
	return complaint, exists
}

// ResolveComplaint marks a complaint as resolved if it exists.
func (s *Store) ResolveComplaint(complaintID string) (*Complaint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	complaint, exists := s.complaints[complaintID]
	if !exists {
		return nil, errors.New("complaint not found")
	}

	complaint.Status = "Resolved"
	return complaint, nil
}
