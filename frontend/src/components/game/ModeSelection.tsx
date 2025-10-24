import { useGameStore } from '../../stores/gameStore';
import { useAuthStore } from '../../stores/authStore';
import type { GameMode } from '../../types';

export const ModeSelection = () => {
  const { difficulty, setDifficulty, setMode } = useGameStore();
  const { isAuthenticated } = useAuthStore();

  const selectDifficulty = (newDifficulty: typeof difficulty) => {
    setDifficulty(newDifficulty);
  };

  const startGame = (mode: GameMode) => {
    setMode(mode);
  };

  const showCreateChallengeModal = () => {
    window.dispatchEvent(new CustomEvent('showCreateChallengeModal'));
  };

  const showJoinChallengeModal = () => {
    window.dispatchEvent(new CustomEvent('showJoinChallengeModal'));
  };

  const showChallengeGuide = () => {
    window.dispatchEvent(new CustomEvent('showChallengeGuide'));
  };

  const showRegisterModal = () => {
    window.dispatchEvent(new CustomEvent('showRegisterModal'));
  };

  const showLoginModal = () => {
    window.dispatchEvent(new CustomEvent('showLoginModal'));
  };

  return (
    <div id="modeSelection" className="mode-selection">
      <h2>Choose Your Game Mode</h2>

      {/* Difficulty Toggle */}
      <div className="difficulty-selector">
        <h3>Difficulty Level</h3>
        <div className="difficulty-toggle">
          <button
            id="easyButton"
            className={`difficulty-button ${difficulty === 'easy' ? 'active' : ''}`}
            onClick={() => selectDifficulty('easy')}
          >
            🟢 Easy Mode
            <small>Lookers Dealership Cars</small>
          </button>
          <button
            id="hardButton"
            className={`difficulty-button ${difficulty === 'hard' ? 'active' : ''}`}
            onClick={() => selectDifficulty('hard')}
          >
            🔴 Hard Mode
            <small>Bonhams Auction Cars</small>
          </button>
        </div>
        <p className="difficulty-description" id="difficultyDescription">
          {difficulty === 'easy' ? (
            <>
              <strong>Easy Mode:</strong> Modern used cars from Lookers dealership. Realistic pricing from a major UK car dealer - perfect for beginners!
            </>
          ) : (
            <>
              <strong>Hard Mode:</strong> Classic and exotic cars from Bonhams auction house. Higher value cars with unique characteristics - the ultimate challenge!
            </>
          )}
        </p>
      </div>

      <div className="mode-cards">
        <div className="mode-card" onClick={() => startGame('challenge')}>
          <span className="recommended-badge">Recommended</span>
          <h3>Challenge Mode</h3>
          <p>10 cars, highest score wins!</p>
          <p className="mode-description">
            GeoGuessr-style scoring! Get up to 5000 points per car based on accuracy. Perfect guess = 5000 points!
          </p>
        </div>
        <div className="mode-card" onClick={() => startGame('streak')}>
          <span className="difficult-badge">Difficult</span>
          <h3>Streak Mode</h3>
          <p>Guess within 10% or game over!</p>
          <p className="mode-description">
            Build your streak by guessing within 10% of the actual price. One wrong guess ends the game!
          </p>
        </div>
        <div className="mode-card" onClick={() => startGame('zero')}>
          <span className="endless-badge">Endless</span>
          <h3>Stay at Zero</h3>
          <p>Accumulate the lowest total difference</p>
          <p className="mode-description">
            Guess car prices and try to keep your cumulative error as close to zero as possible! Play endlessly - the game continues forever!
          </p>
        </div>
      </div>

      {/* Signup Promotion Section (for unauthenticated users) */}
      {!isAuthenticated && (
        <div className="signup-promotion-section" id="signupPromotionSection">
          <div className="challenge-divider">
            <span>UNLOCK MULTIPLAYER</span>
          </div>
          <div className="signup-promotion-box">
            <div className="promotion-content">
              <h3>🎯 Challenge Your Friends!</h3>
              <p>Create your free account to unlock:</p>
              <ul className="feature-list">
                <li>🏆 <strong>Friend Challenges</strong> - Compete on identical car sets</li>
                <li>📊 <strong>Personal Leaderboards</strong> - Track your rankings</li>
                <li>📈 <strong>Game Statistics</strong> - View your progress and achievements</li>
                <li>👥 <strong>Challenge Management</strong> - Create and join unlimited challenges</li>
              </ul>
              <div className="promotion-actions">
                <button className="cta-button" onClick={showRegisterModal}>
                  Create Free Account
                </button>
                <button className="secondary-button" onClick={showLoginModal}>
                  Already have an account?
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Friend Challenge Section (for authenticated users) */}
      {isAuthenticated && (
        <div className="friend-challenge-section" id="friendChallengeSection">
          <div className="challenge-divider">
            <span>OR</span>
          </div>
          <div className="challenge-options">
            <button className="challenge-action-btn create-btn" onClick={showCreateChallengeModal}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              Create Friend Challenge
            </button>
            <button className="challenge-action-btn join-btn" onClick={showJoinChallengeModal}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                <circle cx="8.5" cy="7" r="4"></circle>
                <line x1="20" y1="8" x2="20" y2="14"></line>
                <line x1="23" y1="11" x2="17" y2="11"></line>
              </svg>
              Join with Code
            </button>
            <button className="challenge-action-btn guide-btn" onClick={showChallengeGuide}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10"></circle>
                <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
                <line x1="12" y1="17" x2="12.01" y2="17"></line>
              </svg>
              How It Works
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
