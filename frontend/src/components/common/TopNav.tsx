import { useState } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { useGameStore } from '../../stores/gameStore';

interface TopNavProps {
  onOpenLeaderboard: () => void;
}

export const TopNav = ({ onOpenLeaderboard }: TopNavProps) => {
  const { user, isAuthenticated } = useAuthStore();
  const score = useGameStore((state) => state.score);
  const mode = useGameStore((state) => state.mode);
  const [showDropdown, setShowDropdown] = useState(false);

  const resetGame = useGameStore((state) => state.resetGame);

  const handleGoHome = () => {
    resetGame();
    window.location.reload();
  };

  const getScoreLabel = () => {
    if (!mode) return '';
    if (mode === 'zero') return 'Total Difference: £';
    if (mode === 'streak') return 'Streak: ';
    return 'Score: ';
  };

  return (
    <div className="top-nav">
      <button className="home-button" onClick={handleGoHome} title="Home">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
          <polyline points="9 22 9 12 15 12 15 22"></polyline>
        </svg>
      </button>

      <div className="score-display" id="scoreDisplay" style={{ display: mode ? 'inline-block' : 'none' }}>
        <span id="scoreLabel">{getScoreLabel()}</span>
        <span id="scoreValue">{score.toLocaleString()}</span>
      </div>

      <button onClick={onOpenLeaderboard} className="leaderboard-button" title="View Leaderboard">
        🏆 Leaderboard
      </button>

      <div className="nav-buttons">
        <div className="auth-section" id="authSection">
          {!isAuthenticated ? (
            <button onClick={() => window.dispatchEvent(new CustomEvent('showLoginModal'))} className="auth-button" id="loginBtn">
              Login
            </button>
          ) : (
            <div className="user-menu" id="userMenu">
              <button className="user-button" onClick={() => setShowDropdown(!showDropdown)}>
                <span id="userDisplayName">{user?.displayName || user?.username}</span>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <polyline points="6 9 12 15 18 9"></polyline>
                </svg>
              </button>
              {showDropdown && (
                <div className="user-dropdown" id="userDropdown">
                  <a href="#" onClick={(e) => { e.preventDefault(); window.dispatchEvent(new CustomEvent('showProfile')); }}>
                    My Profile
                  </a>
                  <a href="#" onClick={(e) => { e.preventDefault(); window.dispatchEvent(new CustomEvent('showMyChallenges')); }}>
                    My Challenges
                  </a>
                  <a href="#" onClick={(e) => { e.preventDefault(); window.dispatchEvent(new CustomEvent('logout')); }}>
                    Logout
                  </a>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
