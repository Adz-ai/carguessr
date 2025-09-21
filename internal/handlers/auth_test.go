package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
)

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	root := filepath.Join(cwd, "..", "..")
	if err := os.Chdir(root); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAuthHandler(t *testing.T) (*AuthHandler, *database.Database, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "auth.db")
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	return NewAuthHandler(db), db, func() {
		_ = db.Close()
	}
}

func performJSONRequest(router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegisterFlow(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()

	r := gin.New()
	r.POST("/register", handler.Register)

	reqBody := RegisterRequest{
		Username:         "newuser",
		Password:         "password123",
		DisplayName:      "New User",
		SecurityQuestion: "What is your pet?",
		SecurityAnswer:   "Answer",
	}

	rec := performJSONRequest(r, http.MethodPost, "/register", reqBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	created, err := db.GetUserByUsername("newuser")
	if err != nil {
		t.Fatalf("expected user to be created: %v", err)
	}
	if created.SessionToken == "" {
		t.Fatalf("expected session token to be generated")
	}

	// Invalid username
	badReq := RegisterRequest{Username: "ab", Password: "password", DisplayName: "User", SecurityQuestion: "What is your pet?", SecurityAnswer: "A"}
	rec = performJSONRequest(r, http.MethodPost, "/register", badReq, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid username, got %d", rec.Code)
	}

	// Duplicate username
	rec = performJSONRequest(r, http.MethodPost, "/register", reqBody, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d", rec.Code)
	}

	// Duplicate display name
	reqBody.Username = "newuser2"
	rec = performJSONRequest(r, http.MethodPost, "/register", reqBody, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate display name, got %d", rec.Code)
	}
}

func createUser(t *testing.T, db *database.Database, username, displayName, password, sessionToken string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	answerHash, err := bcrypt.GenerateFromPassword([]byte("answer"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash answer: %v", err)
	}
	user := &models.User{
		Username:           username,
		PasswordHash:       string(hash),
		DisplayName:        displayName,
		SecurityQuestion:   "Question?",
		SecurityAnswerHash: string(answerHash),
		SessionToken:       sessionToken,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func TestLoginFlow(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	_ = createUser(t, db, "loginuser", "Login User", "password123", "session-old")

	r := gin.New()
	r.POST("/login", handler.Login)

	loginReq := LoginRequest{Username: "loginuser", Password: "password123"}
	rec := performJSONRequest(r, http.MethodPost, "/login", loginReq, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", rec.Code)
	}

	fresh, err := db.GetUserByUsername("loginuser")
	if err != nil || fresh.SessionToken == "session-old" {
		t.Fatalf("expected session token to be updated: %+v err=%v", fresh, err)
	}

	// Wrong password
	badReq := LoginRequest{Username: "loginuser", Password: "wrong"}
	rec = performJSONRequest(r, http.MethodPost, "/login", badReq, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rec.Code)
	}

	// Unknown user
	badReq = LoginRequest{Username: "missing", Password: "password"}
	rec = performJSONRequest(r, http.MethodPost, "/login", badReq, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user, got %d", rec.Code)
	}

	// Invalid payload
	rec = performJSONRequest(r, http.MethodPost, "/login", gin.H{"username": 123}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}

	// Verify password hash not returned
	var resp AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.User != nil && resp.User.PasswordHash != "" {
		t.Fatalf("password hash should be omitted")
	}
}

func TestLogoutFlow(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	user := createUser(t, db, "logout", "Logout User", "password123", "token123")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	c.Set("user", user)

	handler.Logout(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	refresh, err := db.GetUserByUsername("logout")
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}
	if refresh.SessionToken != "" {
		t.Fatalf("expected session token to be cleared")
	}
}

func TestProfileEndpoints(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	user := createUser(t, db, "profile", "Profile User", "password123", "token")

	// seed leaderboard
	if err := db.CreateChallengeSession(&models.ChallengeSession{SessionID: "s1", UserID: user.ID, Difficulty: "easy", Cars: []*models.EnhancedCar{}}); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
	if err := db.AddLeaderboardEntry(&models.LeaderboardEntry{UserID: &user.ID, Name: "Profile User", Score: 100, GameMode: "challenge", Difficulty: "easy"}); err != nil {
		t.Fatalf("failed to add leaderboard entry: %v", err)
	}

	r := gin.New()
	r.GET("/profile", handler.RequireAuth(), handler.GetProfile)
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set("user", user)
	handler.GetProfile(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile success, got %d", rec.Code)
	}

	// Not authenticated
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/profile", nil)
	handler.GetProfile(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user missing, got %d", rec.Code)
	}

	// Stats error path
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/profile", nil)
	c.Set("user", user)
	_ = db.Close()
	handler.GetProfile(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when stats fail, got %d", rec.Code)
	}
}

func TestUpdateProfile(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	_ = createUser(t, db, "update", "Original", "password123", "token")
	createUser(t, db, "other", "Taken", "password123", "token2")
	user, err := db.GetUserByUsername("update")
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}

	r := gin.New()
	r.PUT("/profile", handler.RequireAuth(), handler.UpdateProfile)

	// Not authenticated
	rec := performJSONRequest(r, http.MethodPut, "/profile", gin.H{"displayName": "Any"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Invalid payload
	req := httptest.NewRequest(http.MethodPut, "/profile", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set("user", user)
	handler.UpdateProfile(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}

	// Duplicate display name
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	reqBody := gin.H{"displayName": "Taken"}
	payload, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user", user)
	handler.UpdateProfile(c)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when display name taken, got %d", rec.Code)
	}

	// Successful update
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	reqBody = gin.H{"displayName": "Updated", "avatarUrl": "https://avatar"}
	payload, _ = json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user", user)
	handler.UpdateProfile(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d", rec.Code)
	}

	fresh, err := db.GetUserByUsername("update")
	if err != nil || fresh.DisplayName != "Updated" || fresh.AvatarURL != "https://avatar" {
		t.Fatalf("profile update not persisted: %+v err=%v", fresh, err)
	}
}

func TestResetPassword(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	answerHash, _ := bcrypt.GenerateFromPassword([]byte("answer"), bcrypt.MinCost)
	user := &models.User{
		Username:           "reset",
		PasswordHash:       string(hash),
		DisplayName:        "Reset User",
		SecurityQuestion:   "Question?",
		SecurityAnswerHash: string(answerHash),
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	r := gin.New()
	r.POST("/reset", handler.ResetPassword)

	resetReq := models.PasswordResetRequest{
		Username:       "reset",
		DisplayName:    "Reset User",
		SecurityAnswer: "answer",
		NewPassword:    "newpass",
	}
	rec := performJSONRequest(r, http.MethodPost, "/reset", resetReq, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected password reset success, got %d", rec.Code)
	}

	fresh, err := db.GetUserByUsername("reset")
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(fresh.PasswordHash), []byte("newpass")) != nil {
		t.Fatalf("expected password to be updated")
	}

	// Wrong answer
	resetReq.SecurityAnswer = "WRONG"
	rec = performJSONRequest(r, http.MethodPost, "/reset", resetReq, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong answer, got %d", rec.Code)
	}

	// Invalid payload
	rec = performJSONRequest(r, http.MethodPost, "/reset", gin.H{"username": 1}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}
}

func TestGetSecurityQuestion(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	createUser(t, db, "security", "Display", "password123", "token")

	r := gin.New()
	r.POST("/question", handler.GetSecurityQuestion)

	req := gin.H{"username": "security", "displayName": "display"}
	rec := performJSONRequest(r, http.MethodPost, "/question", req, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected question success, got %d", rec.Code)
	}

	req = gin.H{"username": "security", "displayName": "wrong"}
	rec = performJSONRequest(r, http.MethodPost, "/question", req, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for mismatched display name, got %d", rec.Code)
	}

	rec = performJSONRequest(r, http.MethodPost, "/question", gin.H{"username": "missing", "displayName": "Display"}, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing user, got %d", rec.Code)
	}

	rec = performJSONRequest(r, http.MethodPost, "/question", gin.H{"username": 1}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAndRequire(t *testing.T) {
	handler, db, cleanup := setupAuthHandler(t)
	defer cleanup()
	user := createUser(t, db, "authmw", "Auth MW", "password123", "tok")

	// Update session for retrieval
	if err := db.UpdateUserSession(user.ID, "tok"+time.Now().Format("150405")); err != nil {
		t.Fatalf("failed to set session: %v", err)
	}
	fresh, _ := db.GetUserByUsername("authmw")

	r := gin.New()
	r.Use(handler.AuthMiddleware())
	r.GET("/protected", handler.RequireAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Missing token should be unauthorized
	rec := performJSONRequest(r, http.MethodGet, "/protected", nil, map[string]string{"User-Agent": "Mozilla"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when token missing, got %d", rec.Code)
	}

	// Provide via cookie
	rec = performJSONRequest(r, http.MethodGet, "/protected", nil, map[string]string{"Cookie": "session_token=" + fresh.SessionToken, "User-Agent": "Mozilla"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid cookie, got %d", rec.Code)
	}

	// Provide via header
	rec = performJSONRequest(r, http.MethodGet, "/protected", nil, map[string]string{"Authorization": "Bearer " + fresh.SessionToken, "User-Agent": "Mozilla"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with Authorization header, got %d", rec.Code)
	}

	// Invalid token
	rec = performJSONRequest(r, http.MethodGet, "/protected", nil, map[string]string{"Authorization": "Bearer invalid", "User-Agent": "Mozilla"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rec.Code)
	}
}
