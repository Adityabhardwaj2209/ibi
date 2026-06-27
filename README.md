# Complaint Portal Backend API

A robust, thread-safe, and dependency-free Go HTTP JSON API for a Complaint Portal. It allows users to register, log in, submit complaints, and check their status, while enabling administrators to review all complaints and resolve them.

This application is built **strictly using Go's standard library** (`net/http`, `encoding/json`, `sync`, `crypto/rand`, etc.) with zero external dependencies.

---

## Architecture & Design Decisions

1. **In-Memory Store (`sync.RWMutex`)**:
   - The data is kept in-memory to simplify persistence.
   - Operations are protected by a `sync.RWMutex` which handles safe concurrent access. Multiple concurrent read actions (e.g. checking/viewing complaints) hold a read-lock (`RLock`), while writes (e.g. registration, submission, and resolving) hold a write-lock (`Lock`).
   - Slices returned from the store are copied to prevent memory shared references / data races.

2. **Access Control & Secret Code Check**:
   - Every user has a server-generated unique `Secret Code` and `User ID`.
   - Access to user and admin APIs is authenticated by verifying this `Secret Code`.
   - The API accepts this secret code in the request header `X-Secret-Code` (recommended) or as a query parameter `?secretCode=<code_here>` (for quick browser testing).

3. **No External Packages**:
   - **Routing**: `net/http.HandleFunc`.
   - **ID Generation**: Cryptographically secure pseudo-random hex strings generated via `crypto/rand`.

---

## File Structure

- `main.go`: Server startup, registers routes, and binds the HTTP server to port `8080`.
- `models.go`: Holds struct definitions for database models and HTTP request/response payloads.
- `store.go`: Defines the concurrency-safe, in-memory store containing users and complaints.
- `handlers.go`: Contains controllers for validation, authentication, and HTTP routing logic.
- `handlers_test.go`: Implements unit tests and concurrency safety tests (with race detection).

---

## Getting Started

### Prerequisites
- Go version 1.20 or higher installed.

### 1. Run the Server
From the project root directory, run:
```bash
go run .
```
This will start the server listening at `http://localhost:8080`.

### 2. Run Tests
To run unit and concurrency tests with the Go race detector:
```bash
go test -v -race ./...
```

---

## API Documentation

### 1. Register User
- **Route**: `POST /register`
- **Body**:
```json
{
  "name": "Alice Smith",
  "email": "alice@example.com",
  "isAdmin": false
}
```
*(Set `"isAdmin": true` to register an admin user)*
- **Response** (201 Created):
```json
{
  "id": "e391b1efda0be402",
  "secretCode": "fa6bcfd8bde6374da5cf21ab66ed56ef",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "complaints": [],
  "isAdmin": false
}
```

### 2. Login User
- **Route**: `POST /login`
- **Body**:
```json
{
  "secretCode": "fa6bcfd8bde6374da5cf21ab66ed56ef"
}
```
- **Response** (200 OK): Returns complete details of the user.

### 3. Submit Complaint
- **Route**: `POST /submitComplaint`
- **Headers**: `X-Secret-Code: fa6bcfd8bde6374da5cf21ab66ed56ef`
- **Body**:
```json
{
  "title": "Unusable office chair",
  "summary": "The backrest on the desk chair in office room 3B is broken.",
  "severity": 4
}
```
- **Response** (201 Created):
```json
{
  "id": "31b0923cefa0d11b",
  "title": "Unusable office chair",
  "summary": "The backrest on the desk chair in office room 3B is broken.",
  "severity": 4,
  "status": "Pending",
  "userId": "e391b1efda0be402",
  "submitterName": "Alice Smith"
}
```

### 4. Get All Complaints for User
- **Route**: `GET /getAllComplaintsForUser`
- **Headers**: `X-Secret-Code: fa6bcfd8bde6374da5cf21ab66ed56ef`
- **Response** (200 OK):
```json
[
  {
    "id": "31b0923cefa0d11b",
    "title": "Unusable office chair",
    "summary": "The backrest on the desk chair in office room 3B is broken.",
    "severity": 4,
    "status": "Pending"
  }
]
```

### 5. Get All Complaints for Admin
- **Route**: `GET /getAllComplaintsForAdmin`
- **Headers**: `X-Secret-Code: <admin_secret_code>`
- **Response** (200 OK):
```json
[
  {
    "id": "31b0923cefa0d11b",
    "title": "Unusable office chair",
    "summary": "The backrest on the desk chair in office room 3B is broken.",
    "severity": 4,
    "status": "Pending",
    "submitterName": "Alice Smith"
  }
]
```

### 6. View Complaint
- **Route**: `GET /viewComplaint`
- **Headers**: `X-Secret-Code: <secret_code>` *(Allowed for the submitting user or any administrator)*
- **Query Params**: `?id=31b0923cefa0d11b`
- **Response** (200 OK):
```json
{
  "id": "31b0923cefa0d11b",
  "title": "Unusable office chair",
  "summary": "The backrest on the desk chair in office room 3B is broken.",
  "severity": 4,
  "status": "Pending",
  "userId": "e391b1efda0be402",
  "submitterName": "Alice Smith"
}
```

### 7. Resolve Complaint
- **Route**: `POST /resolveComplaint`
- **Headers**: `X-Secret-Code: <admin_secret_code>` *(Only administrators can perform this action)*
- **Body**:
```json
{
  "id": "31b0923cefa0d11b"
}
```
- **Response** (200 OK):
```json
{
  "id": "31b0923cefa0d11b",
  "title": "Unusable office chair",
  "summary": "The backrest on the desk chair in office room 3B is broken.",
  "severity": 4,
  "status": "Resolved",
  "userId": "e391b1efda0be402",
  "submitterName": "Alice Smith"
}
```
