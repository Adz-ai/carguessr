import { useState, type FormEvent } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';

interface JoinChallengeModalProps {
  onClose: () => void;
  onSuccess: (sessionId: string, difficulty: string, challengeCode: string) => void;
}

export const JoinChallengeModal = ({ onClose, onSuccess }: JoinChallengeModalProps) => {
  const { isAuthenticated } = useAuthStore();
  const [code, setCode] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!isAuthenticated) {
      showToast('Please login to join challenges', 'error');
      return;
    }

    const upperCode = code.toUpperCase().trim();
    if (!upperCode) {
      showToast('Please enter a challenge code', 'error');
      return;
    }

    setIsLoading(true);

    try {
      const data = await apiClient.joinChallenge(upperCode);
      showToast(data.message || 'Joined challenge successfully!', 'success');
      onSuccess(data.sessionId, data.challenge.difficulty, upperCode);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to join challenge';
      showToast(message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content challenge-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>Join Friend Challenge</h2>
        <form id="joinChallengeForm" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="challengeCode">Challenge Code</label>
            <input
              type="text"
              id="challengeCode"
              name="code"
              required
              pattern="[A-Za-z0-9]{6}"
              maxLength={6}
              placeholder="ABC123"
              style={{ textTransform: 'uppercase' }}
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              disabled={isLoading}
              autoFocus
            />
            <small>Enter the 6-character code shared by your friend</small>
          </div>
          <button type="submit" className="challenge-submit-btn" disabled={isLoading}>
            {isLoading ? 'Joining...' : 'Join Challenge'}
          </button>
        </form>
      </div>
    </div>
  );
};
