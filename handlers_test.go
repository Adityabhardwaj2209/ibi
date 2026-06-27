package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestRegisterAndLogin(t *testing.T) {
	store := NewStore()

	// 1. Test register regular user
	regPayload := RegisterRequest{
		Name:    "Alice",
		Email:   "alice@example.com",
		IsAdmin: false,
	}
	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handleRegister(store)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", w.Code)
	}

	var user User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to parse register response: %v", err)
	}

	if user.Name != "Alice" || user.Email != "alice@example.com" || user.IsAdmin != false {
		t.Errorf("unexpected user registration fields: %+v", user)
	}
	if user.ID == "" || user.SecretCode == "" {
		t.Errorf("missing server generated ID or SecretCode")
	}

	// 2. Test login with correct secret code
	loginPayload := LoginRequest{
		SecretCode: user.SecretCode,
	}
	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w = httptest.NewRecorder()

	handleLogin(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", w.Code)
	}

	var loggedUser User
	if err := json.Unmarshal(w.Body.Bytes(), &loggedUser); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	if loggedUser.ID != user.ID || loggedUser.SecretCode != user.SecretCode {
		t.Errorf("logged in user does not match registered user")
	}

	// 3. Test login with wrong secret code
	loginPayloadWrong := LoginRequest{
		SecretCode: "invalidcode123",
	}
	body, _ = json.Marshal(loginPayloadWrong)
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w = httptest.NewRecorder()

	handleLogin(store)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected login with invalid code to fail with 401, got %d", w.Code)
	}

	// 4. Test register invalid payloads
	invalidPayload := RegisterRequest{
		Name:  "",
		Email: "invalid@example.com",
	}
	body, _ = json.Marshal(invalidPayload)
	req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handleRegister(store)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 bad request for empty name, got %d", w.Code)
	}
}

func TestSubmitAndQueryComplaints(t *testing.T) {
	store := NewStore()

	// Register Alice (User)
	alice, err := store.CreateUser("Alice", "alice@example.com", false)
	if err != nil {
		t.Fatalf("failed to seed Alice: %v", err)
	}

	// Register Bob (Admin)
	bob, err := store.CreateUser("Bob", "bob@example.com", true)
	if err != nil {
		t.Fatalf("failed to seed Bob: %v", err)
	}

	// 1. Submit complaint (Alice)
	complaintPayload := SubmitComplaintRequest{
		Title:    "No coffee in breakroom",
		Summary:  "There has been no coffee in the kitchen since Monday morning.",
		Severity: 4,
	}
	body, _ := json.Marshal(complaintPayload)
	req := httptest.NewRequest(http.MethodPost, "/submitComplaint", bytes.NewBuffer(body))
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w := httptest.NewRecorder()

	handleSubmitComplaint(store)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected submitComplaint status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var complaint Complaint
	if err := json.Unmarshal(w.Body.Bytes(), &complaint); err != nil {
		t.Fatalf("failed to parse complaint response: %v", err)
	}

	if complaint.Title != "No coffee in breakroom" || complaint.Severity != 4 || complaint.Status != "Pending" {
		t.Errorf("unexpected complaint properties: %+v", complaint)
	}

	// 2. Submit complaint with missing secret code (unauthorized)
	req = httptest.NewRequest(http.MethodPost, "/submitComplaint", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handleSubmitComplaint(store)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected unauthorized 401, got %d", w.Code)
	}

	// 3. Submit complaint with invalid severity
	badComplaintPayload := SubmitComplaintRequest{
		Title:    "Broken Chair",
		Summary:  "The chair is broken.",
		Severity: 10,
	}
	body, _ = json.Marshal(badComplaintPayload)
	req = httptest.NewRequest(http.MethodPost, "/submitComplaint", bytes.NewBuffer(body))
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w = httptest.NewRecorder()
	handleSubmitComplaint(store)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected bad request 400 for severity out of bounds, got %d", w.Code)
	}

	// 4. Get all complaints for user (Alice)
	req = httptest.NewRequest(http.MethodGet, "/getAllComplaintsForUser", nil)
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w = httptest.NewRecorder()
	handleGetAllComplaintsForUser(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected getAllComplaintsForUser 200, got %d", w.Code)
	}

	var userComplaints []ComplaintForUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &userComplaints); err != nil {
		t.Fatalf("failed to parse user complaints list: %v", err)
	}

	if len(userComplaints) != 1 || userComplaints[0].Title != "No coffee in breakroom" {
		t.Errorf("unexpected user complaints: %+v", userComplaints)
	}

	// 5. Get all complaints for admin (Alice tries - should fail)
	req = httptest.NewRequest(http.MethodGet, "/getAllComplaintsForAdmin", nil)
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w = httptest.NewRecorder()
	handleGetAllComplaintsForAdmin(store)(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected Alice (non-admin) to get 403 Forbidden for admin endpoint, got %d", w.Code)
	}

	// 6. Get all complaints for admin (Bob tries - should succeed)
	req = httptest.NewRequest(http.MethodGet, "/getAllComplaintsForAdmin", nil)
	req.Header.Set("X-Secret-Code", bob.SecretCode)
	w = httptest.NewRecorder()
	handleGetAllComplaintsForAdmin(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected admin request 200, got %d", w.Code)
	}

	var adminComplaints []ComplaintForAdminResponse
	if err := json.Unmarshal(w.Body.Bytes(), &adminComplaints); err != nil {
		t.Fatalf("failed to parse admin complaints list: %v", err)
	}

	if len(adminComplaints) != 1 || adminComplaints[0].SubmitterName != "Alice" {
		t.Errorf("unexpected admin complaints response: %+v", adminComplaints)
	}
}

func TestViewAndResolveComplaints(t *testing.T) {
	store := NewStore()

	// Register users
	alice, _ := store.CreateUser("Alice", "alice@example.com", false)
	bob, _ := store.CreateUser("Bob", "bob@example.com", true)
	charlie, _ := store.CreateUser("Charlie", "charlie@example.com", false)

	// Submit complaint from Alice
	comp, err := store.CreateComplaint(alice.ID, "Slow internet", "Internet speed is below 1 Mbps.", 3)
	if err != nil {
		t.Fatalf("failed to submit complaint: %v", err)
	}

	// 1. View complaint (Alice - owner - succeeds)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/viewComplaint?id=%s", comp.ID), nil)
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w := httptest.NewRecorder()
	handleViewComplaint(store)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected view by owner 200, got %d", w.Code)
	}

	// 2. View complaint (Bob - admin - succeeds)
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/viewComplaint?id=%s", comp.ID), nil)
	req.Header.Set("X-Secret-Code", bob.SecretCode)
	w = httptest.NewRecorder()
	handleViewComplaint(store)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected view by admin 200, got %d", w.Code)
	}

	// 3. View complaint (Charlie - other user - fails with 403)
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/viewComplaint?id=%s", comp.ID), nil)
	req.Header.Set("X-Secret-Code", charlie.SecretCode)
	w = httptest.NewRecorder()
	handleViewComplaint(store)(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected view by unrelated user to fail with 403, got %d", w.Code)
	}

	// 4. View non-existent complaint
	req = httptest.NewRequest(http.MethodGet, "/viewComplaint?id=nonexistent", nil)
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w = httptest.NewRecorder()
	handleViewComplaint(store)(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected view of nonexistent complaint to return 404, got %d", w.Code)
	}

	// 5. Resolve complaint (Alice tries - should fail with 403)
	resolvePayload := ResolveComplaintRequest{ID: comp.ID}
	body, _ := json.Marshal(resolvePayload)
	req = httptest.NewRequest(http.MethodPost, "/resolveComplaint", bytes.NewBuffer(body))
	req.Header.Set("X-Secret-Code", alice.SecretCode)
	w = httptest.NewRecorder()
	handleResolveComplaint(store)(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected resolve by non-admin to return 403, got %d", w.Code)
	}

	// 6. Resolve complaint (Bob - admin - succeeds)
	req = httptest.NewRequest(http.MethodPost, "/resolveComplaint", bytes.NewBuffer(body))
	req.Header.Set("X-Secret-Code", bob.SecretCode)
	w = httptest.NewRecorder()
	handleResolveComplaint(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected resolve by admin to return 200, got %d", w.Code)
	}

	var resolvedComp Complaint
	if err := json.Unmarshal(w.Body.Bytes(), &resolvedComp); err != nil {
		t.Fatalf("failed to parse resolve response: %v", err)
	}

	if resolvedComp.Status != "Resolved" {
		t.Errorf("expected status to be 'Resolved', got %q", resolvedComp.Status)
	}

	// Verify status updated in store
	compUpdated, exists := store.GetComplaintByID(comp.ID)
	if !exists || compUpdated.Status != "Resolved" {
		t.Errorf("complaint status not updated correctly in store")
	}
}

func TestStoreConcurrency(t *testing.T) {
	store := NewStore()

	// Seed one admin
	admin, err := store.CreateUser("Admin", "admin@example.com", true)
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	const numUsers = 50
	const complaintsPerUser = 5

	var wg sync.WaitGroup

	// Concurrently register users, submit complaints and read listings
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			name := fmt.Sprintf("User%d", id)
			email := fmt.Sprintf("user%d@example.com", id)

			// Register
			user, err := store.CreateUser(name, email, false)
			if err != nil {
				t.Errorf("concurrency registration failed: %v", err)
				return
			}

			// Submit multiple complaints
			for j := 0; j < complaintsPerUser; j++ {
				title := fmt.Sprintf("Issue %d-%d", id, j)
				summary := fmt.Sprintf("Detailed description of issue %d-%d", id, j)
				comp, err := store.CreateComplaint(user.ID, title, summary, 3)
				if err != nil {
					t.Errorf("concurrency submit complaint failed: %v", err)
					return
				}

				// Resolve the first complaint of each user concurrently from admin
				if j == 0 {
					_, err := store.ResolveComplaint(comp.ID)
					if err != nil {
						t.Errorf("concurrency resolve failed: %v", err)
					}
				}
			}

			// Read user complaints list
			list, err := store.GetComplaintsForUser(user.ID)
			if err != nil {
				t.Errorf("concurrency get complaints failed: %v", err)
			}
			if len(list) != complaintsPerUser {
				t.Errorf("expected %d complaints, got %d", complaintsPerUser, len(list))
			}
		}(i)
	}

	// Concurrently read all complaints as admin
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = store.GetAllComplaints()
		}
	}()

	wg.Wait()

	// Validate final counts
	allComplaints := store.GetAllComplaints()
	expectedTotalComplaints := numUsers * complaintsPerUser
	if len(allComplaints) != expectedTotalComplaints {
		t.Errorf("expected total complaints %d, got %d", expectedTotalComplaints, len(allComplaints))
	}

	// Count resolved
	resolvedCount := 0
	for _, c := range allComplaints {
		if c.Status == "Resolved" {
			resolvedCount++
		}
	}
	if resolvedCount != numUsers {
		t.Errorf("expected resolved complaints to be %d, got %d", numUsers, resolvedCount)
	}

	// Verify admin can fetch all
	req := httptest.NewRequest(http.MethodGet, "/getAllComplaintsForAdmin", nil)
	req.Header.Set("X-Secret-Code", admin.SecretCode)
	w := httptest.NewRecorder()
	handleGetAllComplaintsForAdmin(store)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected admin read to succeed, got %d", w.Code)
	}
}
