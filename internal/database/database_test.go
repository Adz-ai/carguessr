package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"autotraderguesser/internal/models"
)

type failingReader struct{}

func (f failingReader) Read(p []byte) (int, error) {
	return 0, errors.New("forced failure")
}

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

func newTestDatabase(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	return db
}

func TestGenerateSessionTokenFallback(t *testing.T) {
	token := generateSessionToken()
	if len(token) == 0 {
		t.Fatalf("expected token, got empty string")
	}

	original := randomReader
	randomReader = failingReader{}
	defer func() { randomReader = original }()
	fallback := generateSessionToken()
	if len(fallback) == 0 {
		t.Fatalf("expected fallback token")
	}
	if fallback == token {
		t.Fatalf("expected fallback token to differ")
	}
}

func TestDatabaseOperations(t *testing.T) {
	db := newTestDatabase(t)
	defer db.Close()

	user := &models.User{
		Username:           "user1",
		PasswordHash:       "hash",
		DisplayName:        "User One",
		SessionToken:       "token1",
		SecurityQuestion:   "Q?",
		SecurityAnswerHash: "answer",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("expected user ID to be set")
	}

	if fetched, err := db.GetUserByUsername("user1"); err != nil || fetched.ID != user.ID {
		t.Fatalf("GetUserByUsername mismatch: user=%v err=%v", fetched, err)
	}

	if fetched, err := db.GetUserByDisplayName("user one"); err != nil || fetched.ID != user.ID {
		t.Fatalf("GetUserByDisplayName mismatch: user=%v err=%v", fetched, err)
	}

	if fetched, err := db.GetUserBySessionToken("token1"); err != nil || fetched.ID != user.ID {
		t.Fatalf("GetUserBySessionToken mismatch: user=%v err=%v", fetched, err)
	}

	if err := db.UpdateUserSession(user.ID, "token2"); err != nil {
		t.Fatalf("UpdateUserSession failed: %v", err)
	}
	if _, err := db.GetUserBySessionToken("token2"); err != nil {
		t.Fatalf("expected updated session token: %v", err)
	}

	if err := db.UpdateUserLastActive(user.ID); err != nil {
		t.Fatalf("UpdateUserLastActive failed: %v", err)
	}

	if err := db.UpdateUserPassword(user.ID, "hash2"); err != nil {
		t.Fatalf("UpdateUserPassword failed: %v", err)
	}

	user.DisplayName = "User Prime"
	user.AvatarURL = "https://avatar"
	user.FavoriteDifficulty = "hard"
	if err := db.UpdateUser(user); err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if fetched, err := db.GetUserByDisplayName("user prime"); err != nil || fetched.AvatarURL != "https://avatar" {
		t.Fatalf("UpdateUser not persisted: %v err=%v", fetched, err)
	}

	if err := db.UpdateUserActivity(user.ID); err != nil {
		t.Fatalf("UpdateUserActivity failed: %v", err)
	}

	if err := db.IncrementUserGamesPlayed(user.ID); err != nil {
		t.Fatalf("IncrementUserGamesPlayed failed: %v", err)
	}

	user2 := &models.User{
		Username:           "user2",
		PasswordHash:       "hash",
		DisplayName:        "User Two",
		SessionToken:       "token3",
		SecurityQuestion:   "Q2?",
		SecurityAnswerHash: "answer2",
	}
	if err := db.CreateUser(user2); err != nil {
		t.Fatalf("CreateUser second failed: %v", err)
	}

	// Leaderboard entries for favorite difficulty and ranking
	entries := []struct {
		userID     interface{}
		username   string
		score      int
		gameMode   string
		difficulty string
	}{
		{user.ID, "User Prime", 300, "challenge", "easy"},
		{user.ID, "User Prime", 250, "challenge", "easy"},
		{user.ID, "User Prime", 200, "challenge", "hard"},
		{user.ID, "User Prime", 15, "streak", "easy"},
		{user2.ID, "User Two", 400, "challenge", "easy"},
		{nil, "Guest", 500, "challenge", "easy"},
	}

	for _, e := range entries {
		if _, err := db.db.Exec(`INSERT INTO leaderboard_entries (user_id, username, score, game_mode, difficulty) VALUES (?, ?, ?, ?, ?)`, e.userID, e.username, e.score, e.gameMode, e.difficulty); err != nil {
			t.Fatalf("insert leaderboard entry failed: %v", err)
		}
	}

	if err := db.UpdateUserFavoriteDifficulty(user.ID); err != nil {
		t.Fatalf("UpdateUserFavoriteDifficulty failed: %v", err)
	}
	if err := db.UpdateAllUsersFavoriteDifficulty(); err != nil {
		t.Fatalf("UpdateAllUsersFavoriteDifficulty failed: %v", err)
	}
	if updated, err := db.GetUserByUsername("user1"); err != nil || updated.FavoriteDifficulty != "easy" {
		t.Fatalf("expected favorite difficulty easy, got %v err=%v", updated.FavoriteDifficulty, err)
	}

	// Challenge sessions and guesses
	cars := []*models.EnhancedCar{{ID: "car1", Make: "Make", Price: 1000}}
	session := &models.ChallengeSession{
		SessionID:  "session1",
		UserID:     user.ID,
		Difficulty: "easy",
		Cars:       cars,
	}
	if err := db.CreateChallengeSession(session); err != nil {
		t.Fatalf("CreateChallengeSession failed: %v", err)
	}

	// Not found path
	if s, err := db.GetChallengeSession("missing"); err != nil || s != nil {
		t.Fatalf("expected missing session nil, got %v err=%v", s, err)
	}

	retrieved, err := db.GetChallengeSession("session1")
	if err != nil || retrieved == nil || len(retrieved.Cars) != 1 {
		t.Fatalf("GetChallengeSession mismatch: %v err=%v", retrieved, err)
	}

	guess := &models.ChallengeGuess{
		CarIndex:     0,
		CarID:        "car1",
		GuessedPrice: 1100,
		ActualPrice:  1200,
		Points:       10,
		Percentage:   90,
	}
	if err := db.AddChallengeGuess("session1", guess); err != nil {
		t.Fatalf("AddChallengeGuess failed: %v", err)
	}

	session.CurrentCar = 1
	session.TotalScore = 10
	session.IsComplete = false
	if err := db.UpdateChallengeSession(session); err != nil {
		t.Fatalf("UpdateChallengeSession (progress) failed: %v", err)
	}

	session.IsComplete = true
	if err := db.UpdateChallengeSession(session); err != nil {
		t.Fatalf("UpdateChallengeSession (complete) failed: %v", err)
	}

	retrieved, err = db.GetChallengeSession("session1")
	if err != nil || !retrieved.IsComplete || len(retrieved.Guesses) != 1 {
		t.Fatalf("expected completed session with guesses: %v err=%v", retrieved, err)
	}

	session2 := &models.ChallengeSession{
		SessionID:  "session2",
		UserID:     user2.ID,
		Difficulty: "hard",
		Cars:       cars,
	}
	if err := db.CreateChallengeSession(session2); err != nil {
		t.Fatalf("CreateChallengeSession second failed: %v", err)
	}

	// Friend challenge lifecycle
	challenge := &models.FriendChallenge{
		ChallengeCode:     "ABC123",
		Title:             "Test Challenge",
		CreatorUserID:     user.ID,
		TemplateSessionID: session.SessionID,
		Difficulty:        "easy",
		MaxParticipants:   5,
		IsActive:          true,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(48 * time.Hour),
	}
	if err := db.CreateFriendChallenge(challenge); err != nil {
		t.Fatalf("CreateFriendChallenge failed: %v", err)
	}
	if challenge.ID == 0 {
		t.Fatalf("expected challenge ID set")
	}

	if exists, err := db.ChallengeCodeExists("ABC123"); err != nil || !exists {
		t.Fatalf("ChallengeCodeExists true case failed: %v err=%v", exists, err)
	}
	if exists, err := db.ChallengeCodeExists("ZZZZZZ"); err != nil || exists {
		t.Fatalf("ChallengeCodeExists false case failed: %v err=%v", exists, err)
	}

	if _, err := db.GetFriendChallengeByCode("BAD999"); err == nil {
		t.Fatalf("expected error for missing challenge")
	}

	stored, err := db.GetFriendChallengeByCode("ABC123")
	if err != nil || stored.ID != challenge.ID {
		t.Fatalf("GetFriendChallengeByCode mismatch: %v err=%v", stored, err)
	}

	participant1 := &models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            user.ID,
		SessionID:         session.SessionID,
		JoinedAt:          time.Now(),
	}
	if err := db.AddChallengeParticipant(participant1); err != nil {
		t.Fatalf("AddChallengeParticipant1 failed: %v", err)
	}

	participant2 := &models.ChallengeParticipant{
		FriendChallengeID: challenge.ID,
		UserID:            user2.ID,
		SessionID:         session2.SessionID,
		JoinedAt:          time.Now().Add(time.Minute),
	}
	if err := db.AddChallengeParticipant(participant2); err != nil {
		t.Fatalf("AddChallengeParticipant2 failed: %v", err)
	}

	participants, err := db.GetChallengeParticipants(challenge.ID)
	if err != nil || len(participants) != 2 {
		t.Fatalf("GetChallengeParticipants mismatch: %v err=%v", participants, err)
	}

	score1 := 220
	completedAt1 := time.Now()
	participants[0].FinalScore = &score1
	participants[0].CompletedAt = &completedAt1
	participants[0].IsComplete = true

	score2 := 200
	completedAt2 := completedAt1.Add(time.Minute)
	participants[1].FinalScore = &score2
	participants[1].CompletedAt = &completedAt2
	participants[1].IsComplete = true

	if err := db.CalculateChallengeRankings(challenge.ID, participants); err != nil {
		t.Fatalf("CalculateChallengeRankings failed: %v", err)
	}

	updatedParticipants, err := db.GetChallengeParticipants(challenge.ID)
	if err != nil {
		t.Fatalf("GetChallengeParticipants after ranking failed: %v", err)
	}
	if updatedParticipants[0].RankPosition == nil || *updatedParticipants[0].RankPosition != 1 {
		t.Fatalf("expected first participant rank 1: %v", updatedParticipants[0])
	}

	if participation, err := db.GetUserChallengeParticipation(challenge.ID, user2.ID); err != nil || participation.UserID != user2.ID {
		t.Fatalf("GetUserChallengeParticipation mismatch: %v err=%v", participation, err)
	}
	if _, err := db.GetUserChallengeParticipation(challenge.ID, 999); err == nil {
		t.Fatalf("expected error for missing participation")
	}

	created, err := db.GetUserCreatedChallenges(user.ID)
	if err != nil || len(created) != 1 {
		t.Fatalf("GetUserCreatedChallenges mismatch: %v err=%v", created, err)
	}

	participating, err := db.GetUserParticipatingChallenges(user2.ID)
	if err != nil || len(participating) != 1 {
		t.Fatalf("GetUserParticipatingChallenges mismatch: %v err=%v", participating, err)
	}

	if participating[0].JoinedAt == nil {
		t.Fatalf("expected joinedAt for participating challenge")
	}

	// Leaderboard rankings
	if rank, err := db.GetUserLeaderboardRank(user.ID, "challenge", "easy"); err != nil || rank == nil || *rank != 2 {
		t.Fatalf("unexpected registered rank: %v err=%v", rank, err)
	}
	if rank, err := db.GetUserLeaderboardRank(user2.ID, "challenge", "easy"); err != nil || rank == nil || *rank != 1 {
		t.Fatalf("unexpected rank for user2: %v err=%v", rank, err)
	}
	if rank, err := db.GetUserLeaderboardRank(user2.ID, "streak", "hard"); err != nil || rank != nil {
		t.Fatalf("expected nil rank for missing scores: %v err=%v", rank, err)
	}

	if rank, err := db.GetUserOverallLeaderboardRank(user.ID, "challenge", "easy"); err != nil || rank == nil || *rank != 3 {
		t.Fatalf("unexpected overall rank: %v err=%v", rank, err)
	}
	if rank, err := db.GetUserOverallLeaderboardRank(user2.ID, "streak", "hard"); err != nil || rank != nil {
		t.Fatalf("expected nil overall rank: %v err=%v", rank, err)
	}

	stats, err := db.GetUserLeaderboardStats(user.ID)
	if err != nil {
		t.Fatalf("GetUserLeaderboardStats failed: %v", err)
	}
	key := "challenge_easy_registered_rank"
	if v, ok := stats[key]; !ok || v.(int) != 2 {
		t.Fatalf("unexpected stats value: key=%s val=%v", key, v)
	}

	stats2, err := db.GetUserLeaderboardStats(user2.ID)
	if err != nil {
		t.Fatalf("GetUserLeaderboardStats second failed: %v", err)
	}
	for _, k := range []string{"challenge_hard_registered_rank", "streak_easy_registered_rank"} {
		if _, ok := stats2[k]; !ok {
			t.Fatalf("expected key %s in stats", k)
		}
	}
}

func TestCreateUserErrorPaths(t *testing.T) {
	db := newTestDatabase(t)
	defer db.Close()

	first := &models.User{
		Username:           "dupuser",
		PasswordHash:       "hash",
		DisplayName:        "Dup User",
		SessionToken:       "session",
		SecurityQuestion:   "Q?",
		SecurityAnswerHash: "answer",
	}
	if err := db.CreateUser(first); err != nil {
		t.Fatalf("failed to create initial user: %v", err)
	}

	dup := &models.User{
		Username:           "dupuser",
		PasswordHash:       "hash2",
		DisplayName:        "Dup User",
		SessionToken:       "session2",
		SecurityQuestion:   "Q?",
		SecurityAnswerHash: "answer",
	}
	if err := db.CreateUser(dup); err == nil {
		t.Fatalf("expected duplicate user creation to fail")
	}
}
