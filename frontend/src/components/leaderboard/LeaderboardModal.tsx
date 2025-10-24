import { useEffect, useState, useCallback } from 'react';
import { apiClient } from '../../api/client';
import type { LeaderboardEntry, GameMode, Difficulty } from '../../types';
import { useGameStore } from '../../stores/gameStore';

interface LeaderboardModalProps {
  onClose: () => void;
}

export const LeaderboardModal = ({ onClose }: LeaderboardModalProps) => {
  const { difficulty: globalDifficulty } = useGameStore();
  const [selectedMode, setSelectedMode] = useState<GameMode>('challenge');
  const [selectedDifficulty, setSelectedDifficulty] = useState<Difficulty>(globalDifficulty);
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const loadLeaderboard = useCallback(async () => {
    setIsLoading(true);

    try {
      const data = await apiClient.getLeaderboard({
        mode: selectedMode,
        difficulty: selectedDifficulty,
        limit: 100
      });
      setEntries(data || []);
    } catch (error) {
      console.error('Failed to load leaderboard:', error);
      setEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, [selectedMode, selectedDifficulty]);

  useEffect(() => {
    loadLeaderboard();
  }, [loadLeaderboard]);

  const formatScore = (score: number) => {
    return score.toLocaleString();
  };

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content leaderboard-content">
        <h2>🏆 Leaderboard</h2>
        <div className="leaderboard-tabs">
          <div className="tab-row">
            <button
              className={`tab-button ${selectedMode === 'challenge' ? 'active' : ''}`}
              onClick={() => setSelectedMode('challenge')}
            >
              Challenge Mode
            </button>
            <button
              className={`tab-button ${selectedMode === 'streak' ? 'active' : ''}`}
              onClick={() => setSelectedMode('streak')}
            >
              Streak Mode
            </button>
          </div>
          <div className="tab-row difficulty-tabs">
            <button
              className={`difficulty-tab-button ${selectedDifficulty === 'hard' ? 'active' : ''}`}
              onClick={() => setSelectedDifficulty('hard')}
            >
              Hard Mode
            </button>
            <button
              className={`difficulty-tab-button ${selectedDifficulty === 'easy' ? 'active' : ''}`}
              onClick={() => setSelectedDifficulty('easy')}
            >
              Easy Mode
            </button>
          </div>
        </div>

        {isLoading ? (
          <div className="loading-state">Loading leaderboard...</div>
        ) : (
          <div id="leaderboardContent" className="leaderboard-content-area">
            {entries.length === 0 ? (
              <p className="leaderboard-empty">No entries yet. Be the first!</p>
            ) : (
              entries.map((entry, index) => (
                <div key={index} className={`leaderboard-entry ${index < 3 ? 'top-3' : ''}`}>
                  <div className={`leaderboard-rank ${index < 3 ? 'top-3' : ''}`}>
                    {index === 0 && '🥇'}
                    {index === 1 && '🥈'}
                    {index === 2 && '🥉'}
                    {index > 2 && `#${index + 1}`}
                  </div>
                  <div className="leaderboard-name">{entry.name}</div>
                  <div className="leaderboard-score">{formatScore(entry.score)}</div>
                  <div className="leaderboard-date">{new Date(entry.date).toLocaleDateString()}</div>
                </div>
              ))
            )}
          </div>
        )}

        <div className="modal-buttons">
          <button onClick={onClose} className="close-button">Close</button>
        </div>
      </div>
    </div>
  );
};
