import { useState, type FormEvent } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { useGameStore } from '../../stores/gameStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';
import type { Difficulty } from '../../types';

interface CreateChallengeModalProps {
  onClose: () => void;
  onSuccess: (challengeCode: string, sessionId: string) => void;
}

export const CreateChallengeModal = ({ onClose, onSuccess }: CreateChallengeModalProps) => {
  const { isAuthenticated } = useAuthStore();
  const globalDifficulty = useGameStore(state => state.difficulty);

  const [title, setTitle] = useState('');
  const [difficulty, setDifficulty] = useState<Difficulty>(globalDifficulty);
  const [maxParticipants, setMaxParticipants] = useState(10);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!isAuthenticated) {
      showToast('Please login to create challenges', 'error');
      return;
    }

    setIsLoading(true);

    try {
      const data = await apiClient.createChallenge({
        title,
        difficulty,
        maxParticipants
      });

      showToast('Challenge created successfully!', 'success');
      onSuccess(data.challengeCode, data.sessionId);
    } catch (error: any) {
      showToast(error.message || 'Failed to create challenge', 'error');
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
        <h2>Create Friend Challenge</h2>
        <form id="createChallengeForm" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="challengeTitle">Challenge Title</label>
            <input
              type="text"
              id="challengeTitle"
              name="title"
              required
              minLength={1}
              maxLength={100}
              placeholder="Friday Night Challenge"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={isLoading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="challengeDifficulty">Difficulty</label>
            <select
              id="challengeDifficulty"
              name="difficulty"
              value={difficulty}
              onChange={(e) => setDifficulty(e.target.value as Difficulty)}
              disabled={isLoading}
              required
            >
              <option value="easy">🟢 Easy Mode (Lookers Cars)</option>
              <option value="hard">🔴 Hard Mode (Bonhams Cars)</option>
            </select>
          </div>
          <div className="form-group">
            <label htmlFor="maxParticipants">Max Participants</label>
            <input
              type="number"
              id="maxParticipants"
              name="maxParticipants"
              min="2"
              max="50"
              value={maxParticipants}
              onChange={(e) => setMaxParticipants(parseInt(e.target.value))}
              disabled={isLoading}
              required
            />
          </div>
          <button type="submit" className="challenge-submit-btn" disabled={isLoading}>
            {isLoading ? 'Creating...' : 'Create Challenge'}
          </button>
        </form>
      </div>
    </div>
  );
};
