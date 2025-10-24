import type { GuessResult, GameMode } from '../../types';

interface ResultModalProps {
  result: GuessResult;
  onNext: () => void;
  onEnd: () => void;
  onSubmitScore?: () => void;
  isChallenge?: boolean;
  currentCar?: number;
  totalCars?: number;
  gameMode?: GameMode;
  currentScore?: number;
}

export const ResultModal = ({ result, onNext, onEnd, onSubmitScore, isChallenge, currentCar, totalCars, gameMode, currentScore = 0 }: ResultModalProps) => {
  const accuracy = (100 - result.percentage).toFixed(1);
  const isLastCar = result.isLastCar || result.sessionComplete;
  const isStreakMode = gameMode === 'streak';
  const hasStreakScore = isStreakMode && currentScore > 0;

  const getTitle = () => {
    if (isChallenge && currentCar && totalCars) {
      return `Car ${currentCar}/${totalCars} Complete!`;
    }
    return result.correct ? 'Good Guess!' : 'Game Over!';
  };

  const getLinkText = () => {
    if (result.originalUrl) {
      if (result.originalUrl.includes('bonhams')) {
        return 'View Original Auction Listing on Bonhams';
      } else if (result.originalUrl.includes('lookers')) {
        return 'View Original Listing on Lookers';
      }
      return 'View Original Listing';
    }
    return '';
  };

  return (
    <div className="modal" style={{ display: 'flex' }}>
      <div className="modal-content">
        <h2 id="resultTitle">{getTitle()}</h2>
        <div className="result-details">
          <p>Actual Price: <span className="price-highlight">£{result.actualPrice.toLocaleString()}</span></p>
          <p>Your Guess: <span className="price-highlight">£{result.guessedPrice.toLocaleString()}</span></p>
          <p>Difference: <span className="price-highlight">£{result.difference.toLocaleString()}</span></p>
          <p>Accuracy: <span className="price-highlight">{accuracy}% accurate</span></p>
        </div>
        {isChallenge && result.points !== undefined ? (
          <p className="result-message">
            <strong>{result.points} points!</strong><br />
            {result.message}
          </p>
        ) : (
          <p className="result-message">{result.message}</p>
        )}
        {result.originalUrl && (
          <div className="original-link-container">
            <a href={result.originalUrl} target="_blank" rel="noopener noreferrer" className="original-link">
              {getLinkText()}
            </a>
          </div>
        )}
        <div className="modal-buttons">
          {/* Streak mode: show only one button based on success/failure */}
          {isStreakMode ? (
            <>
              {result.correct ? (
                <button onClick={onNext} className="next-button">Next Car</button>
              ) : hasStreakScore && onSubmitScore ? (
                <>
                  <button onClick={onSubmitScore} className="submit-score-button">Submit to Leaderboard</button>
                  <button onClick={onEnd} className="end-button">New Challenge</button>
                </>
              ) : (
                <button onClick={onEnd} className="end-button">End Game</button>
              )}
            </>
          ) : (
            /* Other modes: normal button logic */
            <>
              <button onClick={onNext} className="next-button">
                {isLastCar ? 'View Results' : 'Next Car'}
              </button>
              {!isLastCar && (
                <button onClick={onEnd} className="end-button">End Game</button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
