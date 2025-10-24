// Game Types
export type GameMode = 'zero' | 'streak' | 'challenge';
export type Difficulty = 'easy' | 'hard';

export interface CarListing {
  id: string;
  make: string;
  model: string;
  year: number;
  trim?: string;
  engine: string;
  mileage: number;
  mileageFormatted: string;
  fuelType: string;
  gearbox: string;
  bodyColour: string;
  exteriorColor?: string;
  owners?: string;
  bodyType?: string;
  doors?: string;
  location?: string;
  price: number;
  images: string[];
  fullTitle?: string;

  // Hard mode (auction) specific fields
  auctionDetails?: boolean;
  saleDate?: string;
  interiorColor?: string;
  steering?: string;
  keyFacts?: string[];
  originalUrl?: string;
}

export interface GameState {
  mode: GameMode | null;
  difficulty: Difficulty;
  currentListing: CarListing | null;
  score: number;
  sessionId: string;
  challengeSession: ChallengeSession | null;
  challengeGuesses: GuessResult[];
  pendingLeaderboardData: LeaderboardData | null;
  leaderboardShownAfterSubmission: boolean;
}

export interface ChallengeSession {
  sessionId: string;
  cars: CarListing[];
  currentCar: number;
  totalScore: number;
  guesses: GuessResult[];
  isComplete: boolean;
  difficulty: Difficulty;
}

export interface GuessResult {
  actualPrice: number;
  guessedPrice: number;
  difference: number;
  percentage: number;
  message: string;
  correct: boolean;
  score: number;
  gameOver?: boolean;
  points?: number;
  originalUrl?: string;
  totalScore?: number;
  isLastCar?: boolean;
  sessionComplete?: boolean;
  nextCarNumber?: number;
}

export interface LeaderboardData {
  gameMode: GameMode;
  score: number;
  sessionId: string;
  difficulty: Difficulty;
}

export interface LeaderboardEntry {
  name: string;
  score: number;
  gameMode: GameMode;
  difficulty: Difficulty;
  date: string;
}

// Auth Types
export interface User {
  id: string;
  username: string;
  displayName: string;
  createdAt: string;
  totalGamesPlayed: number;
  favoriteDifficulty?: Difficulty;
  leaderboardStats?: LeaderboardStats;
}

export interface LeaderboardStats {
  challenge_easy_registered_rank?: number;
  challenge_easy_overall_rank?: number;
  challenge_hard_registered_rank?: number;
  challenge_hard_overall_rank?: number;
}

export interface AuthState {
  user: User | null;
  sessionToken: string | null;
  isAuthenticated: boolean;
}

// Challenge Types
export interface Challenge {
  id: string;
  challengeCode: string;
  title: string;
  difficulty: Difficulty;
  maxParticipants: number;
  creatorId: string;
  creatorDisplayName: string;
  createdAt: string;
  expiresAt: string;
  isActive: boolean;
  isComplete?: boolean;
  participantCount: number;
  joinedAt?: string;
}

export interface ChallengeParticipant {
  userId: string;
  username: string;
  displayName: string;
  userDisplayName?: string;
  finalScore: number | null;
  isComplete: boolean;
  completedAt: string | null;
}
