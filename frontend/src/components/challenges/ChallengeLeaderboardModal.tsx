import { useEffect, useState, useCallback } from 'react';
import { apiClient } from '../../api/client';
import type { ChallengeParticipant, Challenge } from '../../types';

interface ChallengeLeaderboardModalProps {
  challengeCode: string;
  onClose: () => void;
}

export const ChallengeLeaderboardModal = ({ challengeCode, onClose }: ChallengeLeaderboardModalProps) => {
  const [participants, setParticipants] = useState<ChallengeParticipant[]>([]);
  const [challenge, setChallenge] = useState<Challenge | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const loadLeaderboard = useCallback(async () => {
    setIsLoading(true);

    try {
      const data = await apiClient.getChallengeLeaderboard(challengeCode);
      setParticipants(data.participants || []);
      setChallenge(data.challenge);
    } catch (error) {
      console.error('Failed to load challenge leaderboard:', error);
    } finally {
      setIsLoading(false);
    }
  }, [challengeCode]);

  useEffect(() => {
    loadLeaderboard();
  }, [loadLeaderboard]);

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content leaderboard-content">
        <h2 id="challengeLeaderboardTitle">
          {challenge ? `🏆 ${challenge.title}` : '🏆 Challenge Leaderboard'}
        </h2>
        <p style={{
          textAlign: 'center',
          marginBottom: '24px',
          fontSize: '14px',
          color: 'rgba(255, 255, 255, 0.8)'
        }}>
          Code: <strong style={{
            color: '#fff',
            fontSize: '16px',
            letterSpacing: '2px'
          }}>{challengeCode}</strong>
        </p>

        {isLoading ? (
          <div className="loading-state">Loading leaderboard...</div>
        ) : (
          <div id="challengeParticipants" className="leaderboard-content-area">
            {participants.length === 0 ? (
              <p className="leaderboard-empty">No participants yet. Share the code to invite others!</p>
            ) : (
              participants.map((participant, index) => {
                const statusIcon = participant.isComplete ? '✅' : '⏳';

                return (
                  <div key={participant.userId} className={`leaderboard-entry ${index < 3 ? 'top-3' : ''}`}>
                    <div className={`leaderboard-rank ${index < 3 ? 'top-3' : ''}`}>
                      {index === 0 && '🥇'}
                      {index === 1 && '🥈'}
                      {index === 2 && '🥉'}
                      {index > 2 && `#${index + 1}`}
                    </div>
                    <div className="leaderboard-name">
                      {participant.userDisplayName || participant.displayName || participant.username || 'Anonymous'}
                    </div>
                    <div className="leaderboard-score">
                      {participant.finalScore !== null && participant.finalScore !== undefined
                        ? participant.finalScore.toLocaleString()
                        : '-'}
                    </div>
                    <div className="leaderboard-date" style={{
                      minWidth: '40px',
                      textAlign: 'center',
                      fontSize: '18px'
                    }}>
                      {statusIcon}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        )}

        <div className="modal-buttons">
          <button onClick={loadLeaderboard} className="submit-button">
            🔄 Refresh
          </button>
          <button onClick={onClose} className="close-button">
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
