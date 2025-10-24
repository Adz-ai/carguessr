import { useEffect, useState } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';

interface ProfileModalProps {
  onClose: () => void;
}

interface UserStats {
  totalGamesPlayed: number;
  challengesCreated: number;
  challengesJoined: number;
  favoriteDifficulty: string;
  challenge_easy_registered_rank: string;
  challenge_easy_overall_rank: string;
  challenge_hard_registered_rank: string;
  challenge_hard_overall_rank: string;
}

export const ProfileModal = ({ onClose }: ProfileModalProps) => {
  const { user } = useAuthStore();
  const [stats, setStats] = useState<UserStats>({
    totalGamesPlayed: 0,
    challengesCreated: 0,
    challengesJoined: 0,
    favoriteDifficulty: 'None yet',
    challenge_easy_registered_rank: 'N/A',
    challenge_easy_overall_rank: 'N/A',
    challenge_hard_registered_rank: 'N/A',
    challenge_hard_overall_rank: 'N/A'
  });

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      // Fetch fresh profile data
      const profileData = await apiClient.getProfile();

      // Fetch challenges data
      const challengesData = await apiClient.getMyChallenges();

      // Get favorite difficulty display text
      let favDiff = 'None yet';
      if (profileData.user.favoriteDifficulty === 'easy') {
        favDiff = 'Easy';
      } else if (profileData.user.favoriteDifficulty === 'hard') {
        favDiff = 'Hard';
      }

      setStats({
        totalGamesPlayed: profileData.user.totalGamesPlayed || 0,
        challengesCreated: challengesData.created?.length || 0,
        challengesJoined: challengesData.participating?.length || 0,
        favoriteDifficulty: favDiff,
        challenge_easy_registered_rank: profileData.leaderboardStats?.challenge_easy_registered_rank || 'N/A',
        challenge_easy_overall_rank: profileData.leaderboardStats?.challenge_easy_overall_rank || 'N/A',
        challenge_hard_registered_rank: profileData.leaderboardStats?.challenge_hard_registered_rank || 'N/A',
        challenge_hard_overall_rank: profileData.leaderboardStats?.challenge_hard_overall_rank || 'N/A'
      });
    } catch (error) {
      console.error('Failed to load user stats:', error);
    }
  };

  if (!user) return null;

  const memberSince = new Date(user.createdAt);

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content profile-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>My Profile</h2>

        <div className="profile-info">
          <div className="profile-field">
            <label>Display Name:</label>
            <span id="profileDisplayName">{user.displayName || user.username}</span>
          </div>
          <div className="profile-field">
            <label>Username:</label>
            <span id="profileUsername">{user.username}</span>
          </div>
          <div className="profile-field">
            <label>Account Type:</label>
            <span id="profileAccountType">Registered Account</span>
          </div>
          <div className="profile-field">
            <label>Member Since:</label>
            <span id="profileMemberSince">{memberSince.toLocaleDateString()}</span>
          </div>
        </div>

        <div className="profile-stats">
          <h3>Game Statistics</h3>
          <div className="stats-grid">
            <div className="stat-item">
              <span className="stat-value" id="totalGamesPlayed">{stats.totalGamesPlayed}</span>
              <span className="stat-label">Games Played</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengesCreated">{stats.challengesCreated}</span>
              <span className="stat-label">Challenges Created</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengesJoined">{stats.challengesJoined}</span>
              <span className="stat-label">Challenges Joined</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="favoriteDifficulty">{stats.favoriteDifficulty}</span>
              <span className="stat-label">Most Played Difficulty</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengeEasyRegisteredRank">{stats.challenge_easy_registered_rank}</span>
              <span className="stat-label">Challenge Easy (Registered)</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengeEasyOverallRank">{stats.challenge_easy_overall_rank}</span>
              <span className="stat-label">Challenge Easy (Overall)</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengeHardRegisteredRank">{stats.challenge_hard_registered_rank}</span>
              <span className="stat-label">Challenge Hard (Registered)</span>
            </div>
            <div className="stat-item">
              <span className="stat-value" id="challengeHardOverallRank">{stats.challenge_hard_overall_rank}</span>
              <span className="stat-label">Challenge Hard (Overall)</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
