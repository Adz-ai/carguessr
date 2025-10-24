import { useState, type FormEvent, useEffect, useRef } from 'react';
import { useGameStore } from '../../stores/gameStore';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';

interface NameInputModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

export const NameInputModal = ({ onClose, onSuccess }: NameInputModalProps) => {
  const { user, isAuthenticated } = useAuthStore();
  const { pendingLeaderboardData } = useGameStore();
  const [name, setName] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const hasSubmittedRef = useRef(false);

  useEffect(() => {
    if (isAuthenticated && user && pendingLeaderboardData && !hasSubmittedRef.current) {
      // Auto-submit for logged-in users
      hasSubmittedRef.current = true;
      setIsLoading(true);

      const submitScore = async () => {
        try {
          await apiClient.submitScore({
            name: user.displayName || user.username,
            score: pendingLeaderboardData.score,
            gameMode: pendingLeaderboardData.gameMode,
            difficulty: pendingLeaderboardData.difficulty,
            sessionId: pendingLeaderboardData.sessionId
          });

          showToast('Score submitted to leaderboard!', 'success');
          onSuccess();
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : 'Failed to submit score';
          showToast(errorMessage, 'error');
          hasSubmittedRef.current = false;
          setIsLoading(false);
        }
      };

      submitScore();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user, pendingLeaderboardData]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!pendingLeaderboardData) {
      showToast('No score data available', 'error');
      return;
    }

    if (!name.trim()) {
      showToast('Please enter a name', 'error');
      return;
    }

    setIsLoading(true);

    try {
      await apiClient.submitScore({
        name: name.trim(),
        score: pendingLeaderboardData.score,
        gameMode: pendingLeaderboardData.gameMode,
        difficulty: pendingLeaderboardData.difficulty,
        sessionId: pendingLeaderboardData.sessionId
      });

      showToast('Score submitted to leaderboard!', 'success');
      onSuccess();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to submit score';
      showToast(errorMessage, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  if (isAuthenticated) {
    return (
      <div className="modal" style={{ display: 'flex' }}>
        <div className="modal-content">
          <h2>Submitting Score...</h2>
          <p>Please wait while we submit your score to the leaderboard.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content">
        <h2>Submit to Leaderboard</h2>
        <p style={{ marginBottom: '20px' }}>Enter your name to save your score</p>
        <form id="nameSubmitForm" onSubmit={handleSubmit}>
          <div className="form-group" style={{ marginBottom: '20px' }}>
            <input
              type="text"
              id="playerName"
              className="name-input"
              placeholder="Your name"
              maxLength={30}
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isLoading}
              autoFocus
            />
          </div>
          <div className="modal-buttons">
            <button type="submit" className="submit-button" disabled={isLoading}>
              {isLoading ? 'Submitting...' : 'Submit Score'}
            </button>
            <button
              type="button"
              className="skip-button"
              onClick={onClose}
              disabled={isLoading}
            >
              Skip
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
