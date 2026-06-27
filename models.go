package main

// User represents a user in the system.
type User struct {
	ID         string       `json:"id"`
	SecretCode string       `json:"secretCode"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Complaints []*Complaint `json:"complaints"`
	IsAdmin    bool         `json:"isAdmin"`
}

// Complaint represents a complaint submitted by a user.
type Complaint struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	Severity      int    `json:"severity"` // Severity rating (e.g., 1 to 5)
	Status        string `json:"status"`   // "Pending" or "Resolved"
	UserID        string `json:"userId"`
	SubmitterName string `json:"submitterName"`
}

// RegisterRequest is the payload for /register
type RegisterRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"isAdmin"` // Optional: to facilitate testing admin endpoints
}

// LoginRequest is the payload for /login
type LoginRequest struct {
	SecretCode string `json:"secretCode"`
}

// SubmitComplaintRequest is the payload for /submitComplaint
type SubmitComplaintRequest struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Severity int    `json:"severity"`
}

// ResolveComplaintRequest is the payload for /resolveComplaint
type ResolveComplaintRequest struct {
	ID string `json:"id"`
}

// ComplaintForUserResponse represents a complaint returned to a normal user
type ComplaintForUserResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Severity int    `json:"severity"`
	Status   string `json:"status"`
}

// ComplaintForAdminResponse represents a complaint returned to an admin
type ComplaintForAdminResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	Severity      int    `json:"severity"`
	Status        string `json:"status"`
	SubmitterName string `json:"submitterName"`
}
