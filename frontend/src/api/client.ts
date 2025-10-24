import type {
  CarListing,
  GuessResult,
  LeaderboardEntry,
  Challenge,
  ChallengeParticipant,
  User,
  Difficulty,
  GameMode,
  ChallengeSession
} from '../types';

class APIClient {

  private getAuthHeaders(): HeadersInit {
    const token = localStorage.getItem('sessionToken');
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  // Game API
  async getRandomCar(difficulty: Difficulty, sessionId: string): Promise<CarListing> {
    const response = await fetch(`/api/random-enhanced-listing?difficulty=${difficulty}`, {
      headers: {
        'X-Session-ID': sessionId,
      },
    });
    if (!response.ok) {
      throw new Error('Failed to load car listing');
    }
    return response.json();
  }

  async submitGuess(data: {
    listingId: string;
    guessedPrice: number;
    gameMode: GameMode;
    difficulty: Difficulty;
    sessionId: string;
  }): Promise<GuessResult> {
    const response = await fetch('/api/check-guess', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Session-ID': data.sessionId,
      },
      body: JSON.stringify({
        listingId: data.listingId,
        guessedPrice: data.guessedPrice,
        gameMode: data.gameMode,
        difficulty: data.difficulty,
      }),
    });
    if (!response.ok) {
      throw new Error('Failed to submit guess');
    }
    return response.json();
  }

  // Challenge API
  async startChallenge(difficulty: Difficulty): Promise<ChallengeSession> {
    const response = await fetch(`/api/challenge/start?difficulty=${difficulty}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!response.ok) {
      throw new Error('Failed to start challenge');
    }
    return response.json();
  }

  async getChallengeSession(sessionId: string): Promise<ChallengeSession> {
    const response = await fetch(`/api/challenge/${sessionId}`);
    if (!response.ok) {
      throw new Error('Failed to load challenge session');
    }
    return response.json();
  }

  async submitChallengeGuess(sessionId: string, guessedPrice: number): Promise<GuessResult> {
    const response = await fetch(`/api/challenge/${sessionId}/guess`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ guessedPrice }),
    });
    if (!response.ok) {
      throw new Error('Failed to submit challenge guess');
    }
    return response.json();
  }

  // Leaderboard API
  async getLeaderboard(params: {
    mode: GameMode;
    difficulty: Difficulty;
    limit?: number;
  }): Promise<LeaderboardEntry[]> {
    const { mode, difficulty, limit = 10 } = params;
    const response = await fetch(
      `/api/leaderboard?mode=${mode}&difficulty=${difficulty}&limit=${limit}`
    );
    if (!response.ok) {
      throw new Error('Failed to load leaderboard');
    }
    return response.json();
  }

  async submitScore(data: {
    name: string;
    score: number;
    gameMode: GameMode;
    difficulty: Difficulty;
    sessionId: string;
  }): Promise<{ success: boolean; entry: LeaderboardEntry; position: number }> {
    const response = await fetch('/api/leaderboard/submit', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to submit score');
    }
    return response.json();
  }

  // Auth API
  async login(username: string, password: string): Promise<{ success: boolean; sessionToken: string; user: User }> {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    });
    const data = await response.json();
    if (!data.success) {
      throw new Error(data.message || 'Login failed');
    }
    return data;
  }

  async register(userData: {
    username: string;
    displayName: string;
    password: string;
    securityQuestion: string;
    securityAnswer: string;
  }): Promise<{ success: boolean; sessionToken: string; user: User }> {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(userData),
    });
    const data = await response.json();
    if (!data.success) {
      throw new Error(data.message || 'Registration failed');
    }
    return data;
  }

  async logout(): Promise<void> {
    const token = localStorage.getItem('sessionToken');
    if (token) {
      await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
    }
  }

  async getProfile(): Promise<{ user: User; leaderboardStats: any }> {
    const response = await fetch('/api/auth/profile', {
      headers: this.getAuthHeaders(),
    });
    if (!response.ok) {
      throw new Error('Failed to load profile');
    }
    return response.json();
  }

  async getSecurityQuestion(username: string, displayName: string): Promise<{ success: boolean; securityQuestion: string }> {
    const response = await fetch('/api/auth/security-question', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, displayName }),
    });
    const data = await response.json();
    if (!data.success) {
      throw new Error(data.message || 'Failed to get security question');
    }
    return data;
  }

  async resetPassword(data: {
    username: string;
    displayName: string;
    securityAnswer: string;
    newPassword: string;
  }): Promise<{ success: boolean }> {
    const response = await fetch('/api/auth/reset-password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });
    const result = await response.json();
    if (!result.success) {
      throw new Error(result.message || 'Password reset failed');
    }
    return result;
  }

  // Friend Challenges API
  async createChallenge(data: {
    title: string;
    difficulty: Difficulty;
    maxParticipants: number;
  }): Promise<{ success: boolean; challengeCode: string; sessionId: string; shareMessage: string }> {
    const response = await fetch('/api/friends/challenges', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(data),
    });
    const result = await response.json();
    if (!result.success) {
      throw new Error(result.message || 'Failed to create challenge');
    }
    return result;
  }

  async joinChallenge(challengeCode: string): Promise<{ success: boolean; sessionId: string; challenge: Challenge; message: string }> {
    const response = await fetch(`/api/friends/challenges/${challengeCode}/join`, {
      method: 'POST',
      headers: this.getAuthHeaders(),
    });
    const result = await response.json();
    if (!result.success) {
      throw new Error(result.message || 'Failed to join challenge');
    }
    return result;
  }

  async getMyChallenges(): Promise<{ success: boolean; created: Challenge[]; participating: Challenge[] }> {
    const response = await fetch('/api/friends/challenges/my-challenges', {
      headers: this.getAuthHeaders(),
    });
    const result = await response.json();
    if (!result.success) {
      throw new Error(result.message || 'Failed to load challenges');
    }
    return result;
  }

  async getChallengeLeaderboard(challengeCode: string): Promise<{ success: boolean; participants: ChallengeParticipant[]; challenge: Challenge }> {
    const response = await fetch(`/api/friends/challenges/${challengeCode}/leaderboard`);
    const result = await response.json();
    if (!result.success) {
      throw new Error('Failed to load challenge leaderboard');
    }
    return result;
  }

  async getChallengeParticipation(challengeCode: string): Promise<{ success: boolean; session: any; challenge: Challenge }> {
    const response = await fetch(`/api/friends/challenges/${challengeCode}/participation`, {
      headers: this.getAuthHeaders(),
    });
    if (!response.ok) {
      throw new Error('Failed to check participation');
    }
    return response.json();
  }
}

export const apiClient = new APIClient();
