import { create } from 'zustand';
import type { User } from '../types';

interface AuthState {
  user: User | null;
  sessionToken: string | null;
  isAuthenticated: boolean;

  // Actions
  setUser: (user: User | null) => void;
  setSessionToken: (token: string | null) => void;
  login: (user: User, token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  sessionToken: localStorage.getItem('sessionToken'),
  isAuthenticated: !!localStorage.getItem('sessionToken'),

  setUser: (user) => set({ user, isAuthenticated: !!user }),

  setSessionToken: (token) => {
    if (token) {
      localStorage.setItem('sessionToken', token);
    } else {
      localStorage.removeItem('sessionToken');
    }
    set({ sessionToken: token, isAuthenticated: !!token });
  },

  login: (user, token) => {
    localStorage.setItem('sessionToken', token);
    set({ user, sessionToken: token, isAuthenticated: true });
  },

  logout: () => {
    localStorage.removeItem('sessionToken');
    set({ user: null, sessionToken: null, isAuthenticated: false });
  },
}));
