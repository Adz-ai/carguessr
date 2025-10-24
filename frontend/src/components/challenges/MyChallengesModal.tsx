import { useEffect, useState } from 'react';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';
import type { Challenge } from '../../types';

interface MyChallengesModalProps {
  onClose: () => void;
  onResumeChallenge: (challengeCode: string) => void;
  onShowLeaderboard: (challengeCode: string) => void;
  onShowCreateModal: () => void;
  onShowJoinModal: () => void;
}

export const MyChallengesModal = ({
  onClose,
  onResumeChallenge,
  onShowLeaderboard,
  onShowCreateModal,
  onShowJoinModal
}: MyChallengesModalProps) => {
  const [activeTab, setActiveTab] = useState<'created' | 'participating'>('created');
  const [createdChallenges, setCreatedChallenges] = useState<Challenge[]>([]);
  const [participatingChallenges, setParticipatingChallenges] = useState<Challenge[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    loadChallenges();
  }, []);

  const loadChallenges = async () => {
    setIsLoading(true);

    try {
      const data = await apiClient.getMyChallenges();
      setCreatedChallenges(data.created || []);
      setParticipatingChallenges(data.participating || []);
    } catch (error: any) {
      showToast(error.message || 'Failed to load challenges', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const copyCode = (code: string) => {
    navigator.clipboard.writeText(code).then(() => {
      showToast('Challenge code copied!', 'success');
    }).catch(() => {
      showToast('Failed to copy code', 'error');
    });
  };

  const shareChallenge = (code: string, title: string) => {
    const shareText = `Join my CarGuessr challenge "${title}"! Use code: ${code}\n\nPlay at: ${window.location.href}`;

    if (navigator.share) {
      navigator.share({
        title: `CarGuessr Challenge: ${title}`,
        text: shareText
      }).catch(() => {
        copyCode(code);
      });
    } else {
      navigator.clipboard.writeText(shareText).then(() => {
        showToast('Challenge details copied to clipboard!', 'success');
      });
    }
  };

  const renderCreatedChallenges = () => {
    if (createdChallenges.length === 0) {
      return (
        <div className="empty-state">
          You haven't created any challenges yet.<br />
          <button onClick={() => { onClose(); onShowCreateModal(); }} className="create-challenge-btn">
            Create Your First Challenge
          </button>
        </div>
      );
    }

    return createdChallenges.map((challenge) => {
      const createdDate = new Date(challenge.createdAt).toLocaleDateString();
      const expiresDate = new Date(challenge.expiresAt).toLocaleDateString();
      const isExpired = new Date() > new Date(challenge.expiresAt);
      const statusIcon = isExpired ? '⏰' : challenge.isActive ? '🟢' : '🔴';
      const statusText = isExpired ? 'Expired' : challenge.isActive ? 'Active' : 'Inactive';

      return (
        <div key={challenge.id} className={`challenge-item ${isExpired ? 'expired' : ''}`}>
          <div className="challenge-header">
            <h4>{challenge.title}</h4>
            <span className="challenge-status">{statusIcon} {statusText}</span>
          </div>
          <div className="challenge-details">
            <span className="challenge-code">Code: <strong>{challenge.challengeCode}</strong></span>
            <span className="challenge-difficulty">
              {challenge.difficulty === 'easy' ? '🟢 Easy' : '🔴 Hard'}
            </span>
            <span className="challenge-participants">
              {challenge.participantCount || 0}/{challenge.maxParticipants} players
            </span>
          </div>
          <div className="challenge-dates">
            <small>Created: {createdDate} | Expires: {expiresDate}</small>
          </div>
          <div className="challenge-actions">
            <button onClick={() => copyCode(challenge.challengeCode)} className="action-btn copy-btn">
              📋 Copy Code
            </button>
            <button onClick={() => onShowLeaderboard(challenge.challengeCode)} className="action-btn leaderboard-btn">
              🏆 Leaderboard
            </button>
            {!isExpired && challenge.isActive && challenge.isComplete !== true && (
              <>
                <button onClick={() => onResumeChallenge(challenge.challengeCode)} className="action-btn resume-btn">
                  ▶️ Play
                </button>
                <button onClick={() => shareChallenge(challenge.challengeCode, challenge.title)} className="action-btn share-btn">
                  📤 Share
                </button>
              </>
            )}
            {challenge.isComplete && <span className="completion-status">✅ Completed</span>}
          </div>
        </div>
      );
    });
  };

  const renderParticipatingChallenges = () => {
    if (participatingChallenges.length === 0) {
      return (
        <div className="empty-state">
          You haven't joined any challenges yet.<br />
          <button onClick={() => { onClose(); onShowJoinModal(); }} className="join-challenge-btn">
            Join a Challenge
          </button>
        </div>
      );
    }

    return participatingChallenges.map((challenge) => {
      const joinedDate = challenge.joinedAt ? new Date(challenge.joinedAt).toLocaleDateString() : 'Unknown';
      const expiresDate = challenge.expiresAt ? new Date(challenge.expiresAt).toLocaleDateString() : 'Unknown';
      const isExpired = challenge.expiresAt ? new Date() > new Date(challenge.expiresAt) : false;
      const statusIcon = isExpired ? '⏰' : challenge.isActive ? '🟢' : '🔴';
      const statusText = isExpired ? 'Expired' : challenge.isActive ? 'Active' : 'Inactive';

      return (
        <div key={challenge.id} className={`challenge-item ${isExpired ? 'expired' : ''}`}>
          <div className="challenge-header">
            <h4>{challenge.title}</h4>
            <span className="challenge-status">{statusIcon} {statusText}</span>
          </div>
          <div className="challenge-details">
            <span className="challenge-creator">By: {challenge.creatorDisplayName}</span>
            <span className="challenge-code">Code: <strong>{challenge.challengeCode}</strong></span>
            <span className="challenge-difficulty">
              {challenge.difficulty === 'easy' ? '🟢 Easy' : '🔴 Hard'}
            </span>
          </div>
          <div className="challenge-dates">
            <small>Joined: {joinedDate} | Expires: {expiresDate}</small>
          </div>
          <div className="challenge-actions">
            <button onClick={() => onShowLeaderboard(challenge.challengeCode)} className="action-btn leaderboard-btn">
              🏆 Leaderboard
            </button>
            {!isExpired && challenge.isActive && challenge.isComplete !== true && (
              <button onClick={() => onResumeChallenge(challenge.challengeCode)} className="action-btn resume-btn">
                ▶️ Resume
              </button>
            )}
            {challenge.isComplete && <span className="completion-status">✅ Completed</span>}
          </div>
        </div>
      );
    });
  };

  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content challenges-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>My Challenges</h2>

        <div className="challenges-tabs">
          <button
            className={`tab-button ${activeTab === 'created' ? 'active' : ''}`}
            onClick={() => setActiveTab('created')}
          >
            Created by Me
          </button>
          <button
            className={`tab-button ${activeTab === 'participating' ? 'active' : ''}`}
            onClick={() => setActiveTab('participating')}
          >
            I'm Participating
          </button>
        </div>

        {isLoading ? (
          <div className="loading-state">Loading challenges...</div>
        ) : (
          <>
            <div id="createdChallenges" className="challenge-list" style={{ display: activeTab === 'created' ? 'block' : 'none' }}>
              {renderCreatedChallenges()}
            </div>
            <div id="participatingChallenges" className="challenge-list" style={{ display: activeTab === 'participating' ? 'block' : 'none' }}>
              {renderParticipatingChallenges()}
            </div>
          </>
        )}
      </div>
    </div>
  );
};
