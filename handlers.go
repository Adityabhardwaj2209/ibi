package main

import (
	"encoding/json"
	"net/http"
)

// JSON helper response functions
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// authenticate checks the X-Secret-Code header to validate the user.
func authenticate(w http.ResponseWriter, r *http.Request, store *Store) (*User, bool) {
	secretCode := r.Header.Get("X-Secret-Code")
	if secretCode == "" {
		// Fallback to checking the query parameter "secretCode"
		secretCode = r.URL.Query().Get("secretCode")
	}

	if secretCode == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication secret code")
		return nil, false
	}

	user, exists := store.GetUserBySecretCode(secretCode)
	if !exists {
		writeError(w, http.StatusUnauthorized, "invalid secret code")
		return nil, false
	}

	return user, true
}

// handleRegister registers a new user.
func handleRegister(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		user, err := store.CreateUser(req.Name, req.Email, req.IsAdmin)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, user)
	}
}

// handleLogin logs in an existing user using their secret code.
func handleLogin(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		if req.SecretCode == "" {
			writeError(w, http.StatusBadRequest, "secretCode is required")
			return
		}

		user, exists := store.GetUserBySecretCode(req.SecretCode)
		if !exists {
			writeError(w, http.StatusUnauthorized, "invalid secret code")
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

// handleSubmitComplaint creates a new complaint.
func handleSubmitComplaint(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
			return
		}

		user, ok := authenticate(w, r, store)
		if !ok {
			return
		}

		var req SubmitComplaintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		complaint, err := store.CreateComplaint(user.ID, req.Title, req.Summary, req.Severity)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, complaint)
	}
}

// handleGetAllComplaintsForUser lists complaints submitted by the authenticated user.
func handleGetAllComplaintsForUser(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "only GET is supported")
			return
		}

		user, ok := authenticate(w, r, store)
		if !ok {
			return
		}

		complaints, err := store.GetComplaintsForUser(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Map to standard response details such as Complaint's Title
		res := make([]ComplaintForUserResponse, len(complaints))
		for i, c := range complaints {
			res[i] = ComplaintForUserResponse{
				ID:       c.ID,
				Title:    c.Title,
				Summary:  c.Summary,
				Severity: c.Severity,
				Status:   c.Status,
			}
		}

		writeJSON(w, http.StatusOK, res)
	}
}

// handleGetAllComplaintsForAdmin lists all complaints on the portal.
func handleGetAllComplaintsForAdmin(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "only GET is supported")
			return
		}

		user, ok := authenticate(w, r, store)
		if !ok {
			return
		}

		if !user.IsAdmin {
			writeError(w, http.StatusForbidden, "administrator role required")
			return
		}

		complaints := store.GetAllComplaints()

		// Map to response with submitter name
		res := make([]ComplaintForAdminResponse, len(complaints))
		for i, c := range complaints {
			res[i] = ComplaintForAdminResponse{
				ID:            c.ID,
				Title:         c.Title,
				Summary:       c.Summary,
				Severity:      c.Severity,
				Status:        c.Status,
				SubmitterName: c.SubmitterName,
			}
		}

		writeJSON(w, http.StatusOK, res)
	}
}

// handleViewComplaint displays details of a specific complaint.
func handleViewComplaint(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "only GET is supported")
			return
		}

		user, ok := authenticate(w, r, store)
		if !ok {
			return
		}

		complaintID := r.URL.Query().Get("id")
		if complaintID == "" {
			writeError(w, http.StatusBadRequest, "complaint id parameter is required")
			return
		}

		complaint, exists := store.GetComplaintByID(complaintID)
		if !exists {
			writeError(w, http.StatusNotFound, "complaint not found")
			return
		}

		// Allow if requestor is admin OR is the user who submitted it
		if !user.IsAdmin && complaint.UserID != user.ID {
			writeError(w, http.StatusForbidden, "unauthorized access to complaint details")
			return
		}

		writeJSON(w, http.StatusOK, complaint)
	}
}

// handleResolveComplaint resolves a complaint.
func handleResolveComplaint(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
			return
		}

		user, ok := authenticate(w, r, store)
		if !ok {
			return
		}

		if !user.IsAdmin {
			writeError(w, http.StatusForbidden, "administrator role required")
			return
		}

		var req ResolveComplaintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		if req.ID == "" {
			writeError(w, http.StatusBadRequest, "complaint ID is required")
			return
		}

		complaint, err := store.ResolveComplaint(req.ID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, complaint)
	}
}
