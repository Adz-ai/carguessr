import { useEffect } from 'react';
import { useAuthStore } from '../stores/authStore';
import { apiClient } from '../api/client';

export const useAuth = () => {
  const { user, setUser, isAuthenticated, login, logout, sessionToken } = useAuthStore();

  useEffect(() => {
    const checkAuth = async () => {
      if (sessionToken && !user) {
        try {
          const { user: userData, leaderboardStats } = await apiClient.getProfile();
          userData.leaderboardStats = leaderboardStats;
          setUser(userData);
        } catch (error) {
          console.error('Auth check failed:', error);
          logout();
        }
      }
    };

    checkAuth();
  }, [sessionToken, user, setUser, logout]);

  return {
    user,
    isAuthenticated,
    login,
    logout,
  };
};
