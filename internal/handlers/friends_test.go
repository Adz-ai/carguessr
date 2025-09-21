package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"autotraderguesser/internal/database"
	"autotraderguesser/internal/models"
)

type sequenceReader struct {
	sequences [][]byte
	idx       int
}

func (s *sequenceReader) Read(p []byte) (int, error) {
	if s.idx >= len(s.sequences) {
		for i := range p {
			p[i] = 0
		}
	} else {
		seq := s.sequences[s.idx]
		copy(p, seq)
		s.idx++
	}
	return len(p), nil
}

type fakeGameHandler struct {
	session        *models.ChallengeSession
	err            error
	lastDifficulty string
	lastUserID     int
}

func (f *fakeGameHandler) CreateTemplateChallenge(difficulty string, userID int) (*models.ChallengeSession, error) {
	f.lastDifficulty = difficulty
	f.lastUserID = userID
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

func setupFriendsHandler(t *testing.T, gh GameHandlerInterface) (*FriendsHandler, *database.Database, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "friends.db")
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	return NewFriendsHandler(db, gh), db, func() { _ = db.Close() }
}

func createUserForFriends(t *testing.T, db *database.Database, username, displayName string) *models.User {
	t.Helper()
	user := &models.User{
		Username:           username,
		PasswordHash:       "hash",
		DisplayName:        displayName,
		SecurityQuestion:   "Question?",
		SecurityAnswerHash: "answer",
		SessionToken:       username + "-token",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func invokeFriendsHandler(t *testing.T, handler gin.HandlerFunc, method, path string, params gin.Params, body interface{}, user *models.User) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = params
	if user != nil {
		c.Set("user", user)
	}
	handler(c)
	return rec
}

func TestCreateFriendChallenge(t *testing.T) {
	origChallengeReader := challengeRandReader
	origSessionReader := sessionRandReader
	defer func() {
		challengeRandReader = origChallengeReader
		sessionRandReader = origSessionReader
	}()

	seq := &sequenceReader{sequences: [][]byte{{'A', 'A', 'A', 'A', 'A', 'A'}, {'B', 'B', 'B', 'B', 'B', 'B'}}}
	challengeRandReader = seq
	sessionRandReader = &sequenceReader{sequences: [][]byte{{'T', 'E', 'M', 'P', 'L', 'A', 'T', 'E', '1', '2', '3', '4', '5', '6', '7', '8'}}}

	template := &models.ChallengeSession{SessionID: "template-session", Difficulty: "easy"}
	game := &fakeGameHandler{session: template}
	handler, db, cleanup := setupFriendsHandler(t, game)
	if err := db.CreateChallengeSession(template); err != nil {
		t.Fatalf("failed to store template session: %v", err)
	}
	defer cleanup()

	creator := createUserForFriends(t, db, "creator", "Creator")

	// Pre-create challenge to force collision on first generated code
	if err := db.CreateChallengeSession(&models.ChallengeSession{SessionID: "template-old", Difficulty: "easy", Cars: []*models.EnhancedCar{{ID: "seed"}}}); err != nil {
		t.Fatalf("failed to create seed template: %v", err)
	}

	if err := db.CreateFriendChallenge(&models.FriendChallenge{
		ChallengeCode:     "333333",
		Title:             "Existing",
		CreatorUserID:     creator.ID,
		TemplateSessionID: "template-old",
		Difficulty:        "easy",
		MaxParticipants:   5,
		IsActive:          true,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("failed to seed challenge: %v", err)
	}

	req := models.CreateFriendChallengeRequest{Title: "Fun Challenge", Difficulty: "easy", MaxParticipants: 5}
	rec := invokeFriendsHandler(t, handler.CreateFriendChallenge, http.MethodPost, "/friends", nil, req, creator)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	if game.lastDifficulty != "easy" || game.lastUserID != creator.ID {
		t.Fatalf("game handler not invoked correctly: %+v", game)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	code, ok := resp["challengeCode"].(string)
	if !ok || code == "" {
		t.Fatalf("expected challengeCode string in response")
	}
	if code == "333333" {
		t.Fatalf("challenge code should differ from preexisting")
	}

	saved, err := db.GetFriendChallengeByCode(code)
	if err != nil {
		t.Fatalf("challenge not stored: %v", err)
	}
	if saved.TemplateSessionID != "template-session" {
		t.Fatalf("expected template session to match")
	}

	// Invalid payload
	rec = invokeFriendsHandler(t, handler.CreateFriendChallenge, http.MethodPost, "/friends", nil, gin.H{"title": 123}, creator)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}

	// Unauthorized
	rec = invokeFriendsHandler(t, handler.CreateFriendChallenge, http.MethodPost, "/friends", nil, req, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when unauthenticated, got %d", rec.Code)
	}

	// Template failure
	game.err = fmt.Errorf("template error")
	rec = invokeFriendsHandler(t, handler.CreateFriendChallenge, http.MethodPost, "/friends", nil, req, creator)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when template fails, got %d", rec.Code)
	}
}

func seedFriendChallenge(t *testing.T, db *database.Database, creator *models.User, code string, maxParticipants int, expires time.Time) *models.FriendChallenge {
	t.Helper()
	cars := []*models.EnhancedCar{{ID: "car1"}}
	template := &models.ChallengeSession{
		SessionID:  code + "-TEMPLATE",
		UserID:     creator.ID,
		Difficulty: "easy",
		Cars:       cars,
	}
	if err := db.CreateChallengeSession(template); err != nil {
		t.Fatalf("failed to create template session: %v", err)
	}

	challenge := &models.FriendChallenge{
		ChallengeCode:     code,
		Title:             "Join Challenge",
		CreatorUserID:     creator.ID,
		TemplateSessionID: template.SessionID,
		Difficulty:        "easy",
		MaxParticipants:   maxParticipants,
		IsActive:          true,
		CreatedAt:         time.Now(),
		ExpiresAt:         expires,
	}
	if err := db.CreateFriendChallenge(challenge); err != nil {
		t.Fatalf("failed to create friend challenge: %v", err)
	}

	if err := db.AddChallengeParticipant(&models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            creator.ID,
		SessionID:         template.SessionID,
		JoinedAt:          time.Now(),
	}); err != nil {
		t.Fatalf("failed to add creator participant: %v", err)
	}
	return challenge
}

func TestJoinFriendChallenge(t *testing.T) {
	origSessionReader := sessionRandReader
	defer func() { sessionRandReader = origSessionReader }()

	t.Run("success", func(t *testing.T) {
		sessionRandReader = &sequenceReader{sequences: [][]byte{{'S', 'E', 'S', 'S', 'I', 'O', 'N', 'J', 'O', 'I', 'N', '0', '0', '1', '0', '0'}}}
		handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		creator := createUserForFriends(t, db, "creator", "Creator")
		participant := createUserForFriends(t, db, "joiner", "Joiner")
		_ = seedFriendChallenge(t, db, creator, "JOIN01", 3, time.Now().Add(time.Hour))

		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN01", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, participant)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected join success, got %d", rec.Code)
		}
	})

	t.Run("already participating", func(t *testing.T) {
		handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		creator := createUserForFriends(t, db, "creator", "Creator")
		_ = seedFriendChallenge(t, db, creator, "JOIN02", 3, time.Now().Add(time.Hour))

		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN02", gin.Params{{Key: "code", Value: "JOIN02"}}, nil, creator)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 when already participating, got %d", rec.Code)
		}
	})

	t.Run("challenge full", func(t *testing.T) {
		handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		creator := createUserForFriends(t, db, "creator", "Creator")
		challenge := seedFriendChallenge(t, db, creator, "JOIN03", 1, time.Now().Add(time.Hour))

		other := createUserForFriends(t, db, "other", "Other")
		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN03", gin.Params{{Key: "code", Value: "JOIN03"}}, nil, other)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 when challenge full, got %d", rec.Code)
		}

		_ = challenge
	})

	t.Run("expired", func(t *testing.T) {
		handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		creator := createUserForFriends(t, db, "creator", "Creator")
		participant := createUserForFriends(t, db, "joiner", "Joiner")
		_ = seedFriendChallenge(t, db, creator, "JOIN04", 3, time.Now().Add(-time.Hour))

		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN04", gin.Params{{Key: "code", Value: "JOIN04"}}, nil, participant)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for expired challenge, got %d", rec.Code)
		}
	})

	t.Run("invalid code", func(t *testing.T) {
		handler, _, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		participant := &models.User{ID: 1}

		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/bad", gin.Params{{Key: "code", Value: "bad"}}, nil, participant)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid code, got %d", rec.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		handler, _, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN05", gin.Params{{Key: "code", Value: "JOIN05"}}, nil, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when unauthenticated, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler, _, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
		defer cleanup()
		participant := &models.User{ID: 1}

		rec := invokeFriendsHandler(t, handler.JoinFriendChallenge, http.MethodPost, "/join/JOIN09", gin.Params{{Key: "code", Value: "JOIN09"}}, nil, participant)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for missing challenge, got %d", rec.Code)
		}
	})

}

func TestGetFriendChallengeAndLeaderboard(t *testing.T) {
	handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
	defer cleanup()
	creator := createUserForFriends(t, db, "creator", "Creator")
	seedFriendChallenge(t, db, creator, "JOIN01", 3, time.Now().Add(2*time.Hour))

	rec := invokeFriendsHandler(t, handler.GetFriendChallenge, http.MethodGet, "/challenge/JOIN01", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for get challenge, got %d", rec.Code)
	}

	// Bad code format
	rec = invokeFriendsHandler(t, handler.GetFriendChallenge, http.MethodGet, "/challenge/bad", gin.Params{{Key: "code", Value: "bad"}}, nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid code, got %d", rec.Code)
	}

	// Leaderboard success (no completions yet)
	rec = invokeFriendsHandler(t, handler.GetChallengeLeaderboard, http.MethodGet, "/challenge/JOIN01/leaderboard", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for leaderboard, got %d", rec.Code)
	}

	// Close DB to trigger participant fetch error
	_ = db.Close()
	rec = invokeFriendsHandler(t, handler.GetFriendChallenge, http.MethodGet, "/challenge/JOIN01", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when db closed, got %d", rec.Code)
	}
}

func TestGetUserParticipationAndMyChallenges(t *testing.T) {
	handler, db, cleanup := setupFriendsHandler(t, &fakeGameHandler{})
	defer cleanup()
	creator := createUserForFriends(t, db, "creator", "Creator")
	participant := createUserForFriends(t, db, "joiner", "Joiner")
	outsider := createUserForFriends(t, db, "outsider", "Outsider")
	seedFriendChallenge(t, db, creator, "JOIN01", 3, time.Now().Add(2*time.Hour))

	challenge, err := db.GetFriendChallengeByCode("JOIN01")
	if err != nil {
		t.Fatalf("failed to fetch challenge: %v", err)
	}

	// Add participant entry manually
	session := &models.ChallengeSession{
		SessionID:  "participant-session",
		UserID:     participant.ID,
		Difficulty: "easy",
		Cars:       []*models.EnhancedCar{{ID: "car1"}},
	}
	if err := db.CreateChallengeSession(session); err != nil {
		t.Fatalf("failed to create participant session: %v", err)
	}
	if err := db.AddChallengeParticipant(&models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            participant.ID,
		SessionID:         session.SessionID,
		JoinedAt:          time.Now(),
	}); err != nil {
		t.Fatalf("failed to add participant: %v", err)
	}

	rec := invokeFriendsHandler(t, handler.GetUserParticipation, http.MethodGet, "/participation/JOIN01", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, participant)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for participation, got %d", rec.Code)
	}

	// Not participating
	rec = invokeFriendsHandler(t, handler.GetUserParticipation, http.MethodGet, "/participation/JOIN01", gin.Params{{Key: "code", Value: "JOIN01"}}, nil, outsider)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non participant, got %d", rec.Code)
	}

	// Invalid code format
	rec = invokeFriendsHandler(t, handler.GetUserParticipation, http.MethodGet, "/participation/bad", gin.Params{{Key: "code", Value: "bad"}}, nil, participant)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid code, got %d", rec.Code)
	}

	// GetMyChallenges success
	rec = invokeFriendsHandler(t, handler.GetMyChallenges, http.MethodGet, "/my", nil, nil, participant)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for my challenges, got %d", rec.Code)
	}

	// Unauthenticated access
	rec = invokeFriendsHandler(t, handler.GetMyChallenges, http.MethodGet, "/my", nil, nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when unauthenticated, got %d", rec.Code)
	}
}
