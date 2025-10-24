import { create } from 'zustand';
import type {
  GameMode,
  Difficulty,
  CarListing,
  ChallengeSession,
  GuessResult,
  LeaderboardData
} from '../types';

interface GameState {
  mode: GameMode | null;
  difficulty: Difficulty;
  currentListing: CarListing | null;
  score: number;
  sessionId: string;
  challengeSession: ChallengeSession | null;
  challengeGuesses: GuessResult[];
  challengeCode: string | null; // For friend challenges
  pendingLeaderboardData: LeaderboardData | null;
  leaderboardShownAfterSubmission: boolean;

  // Actions
  setMode: (mode: GameMode | null) => void;
  setDifficulty: (difficulty: Difficulty) => void;
  setCurrentListing: (listing: CarListing | null) => void;
  setScore: (score: number) => void;
  setSessionId: (id: string) => void;
  setChallengeSession: (session: ChallengeSession | null) => void;
  addChallengeGuess: (guess: GuessResult) => void;
  setChallengeCode: (code: string | null) => void;
  setPendingLeaderboardData: (data: LeaderboardData | null) => void;
  setLeaderboardShownAfterSubmission: (shown: boolean) => void;
  resetGame: () => void;
}

const generateSessionId = (): string => {
  const letters = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let result = '';
  for (let i = 0; i < 16; i++) {
    result += letters.charAt(Math.floor(Math.random() * letters.length));
  }
  return result;
};

const loadDifficultyPreference = (): Difficulty => {
  try {
    const saved = localStorage.getItem('carguessr_difficulty');
    return (saved as Difficulty) || 'easy';
  } catch {
    return 'easy';
  }
};

export const useGameStore = create<GameState>((set) => ({
  mode: null,
  difficulty: loadDifficultyPreference(),
  currentListing: null,
  score: 0,
  sessionId: generateSessionId(),
  challengeSession: null,
  challengeGuesses: [],
  challengeCode: null,
  pendingLeaderboardData: null,
  leaderboardShownAfterSubmission: false,

  setMode: (mode) => set({ mode }),
  setDifficulty: (difficulty) => {
    try {
      localStorage.setItem('carguessr_difficulty', difficulty);
    } catch (e) {
      console.error('Failed to save difficulty preference:', e);
    }
    set({ difficulty });
  },
  setCurrentListing: (listing) => set({ currentListing: listing }),
  setScore: (score) => set({ score }),
  setSessionId: (id) => set({ sessionId: id }),
  setChallengeSession: (session) => set({ challengeSession: session }),
  addChallengeGuess: (guess) =>
    set((state) => ({
      challengeGuesses: [...state.challengeGuesses, guess],
    })),
  setChallengeCode: (code) => set({ challengeCode: code }),
  setPendingLeaderboardData: (data) => set({ pendingLeaderboardData: data }),
  setLeaderboardShownAfterSubmission: (shown) => set({ leaderboardShownAfterSubmission: shown }),
  resetGame: () =>
    set({
      mode: null,
      currentListing: null,
      score: 0,
      sessionId: generateSessionId(),
      challengeSession: null,
      challengeGuesses: [],
      challengeCode: null,
      pendingLeaderboardData: null,
      leaderboardShownAfterSubmission: false,
    }),
}));
