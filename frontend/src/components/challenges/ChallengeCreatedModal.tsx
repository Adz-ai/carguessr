import { showToast } from '../../utils/toast';

interface ChallengeCreatedModalProps {
  challengeCode: string;
  sessionId: string;
  onClose: () => void;
  onStartChallenge: (sessionId: string) => void;
}

export const ChallengeCreatedModal = ({
  challengeCode,
  sessionId,
  onClose,
  onStartChallenge
}: ChallengeCreatedModalProps) => {

  const copyCode = () => {
    navigator.clipboard.writeText(challengeCode).then(() => {
      showToast('Challenge code copied!', 'success');
    }).catch(() => {
      showToast('Failed to copy code', 'error');
    });
  };

  return (
    <div className="modal" style={{ display: 'flex' }}>
      <div className="modal-content challenge-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>Challenge Created! 🎉</h2>
        <div className="challenge-code-display">
          <p>Share this code with your friends:</p>
          <div className="code-box">
            <span id="createdChallengeCode">{challengeCode}</span>
            <button onClick={copyCode} className="copy-btn">📋 Copy</button>
          </div>
        </div>
        <p className="challenge-info" id="challengeShareMessage">
          Share this code with friends to let them join. The challenge will be available for 7 days.
        </p>
        <button className="challenge-start-btn" onClick={() => onStartChallenge(sessionId)}>
          Start Challenge
        </button>
      </div>
    </div>
  );
};
