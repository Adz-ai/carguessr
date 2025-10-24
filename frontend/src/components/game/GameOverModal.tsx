import type { GameMode } from '../../types';

interface GameOverModalProps {
  score: number;
  mode: GameMode;
  onSubmitScore: () => void;
  onPlayAgain: () => void;
}

export const GameOverModal = ({ score, mode, onSubmitScore, onPlayAgain }: GameOverModalProps) => {
  const scoreText = mode === 'zero'
    ? `Total Difference: £${score.toLocaleString()}`
    : `Final Streak: ${score}`;

  const showSubmitButton = !(mode === 'streak' && score === 0);

  return (
    <div className="modal" style={{ display: 'flex' }}>
      <div className="modal-content">
        <h2>Game Over!</h2>
        <p className="final-score">{scoreText}</p>
        <div className="modal-buttons">
          {showSubmitButton && (
            <button onClick={onSubmitScore} className="submit-score-button" id="submitStreakScore">
              Submit to Leaderboard
            </button>
          )}
          <button onClick={onPlayAgain} className="play-again-button">
            Play Again
          </button>
        </div>
      </div>
    </div>
  );
};
