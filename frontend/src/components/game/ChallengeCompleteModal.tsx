import type { GuessResult, CarListing } from '../../types';

interface ChallengeCompleteModalProps {
  totalScore: number;
  guesses: GuessResult[];
  cars: CarListing[];
  challengeCode?: string | null; // For friend challenges
  onSubmitScore: () => void;
  onViewLeaderboard?: () => void; // For friend challenges
  onPlayAgain: () => void;
}

export const ChallengeCompleteModal = ({
  totalScore,
  guesses,
  cars,
  challengeCode,
  onSubmitScore,
  onViewLeaderboard,
  onPlayAgain
}: ChallengeCompleteModalProps) => {
  const isFriendChallenge = !!challengeCode;
  const showSubmitButton = !isFriendChallenge && totalScore > 0;

  return (
    <div className="modal" style={{ display: 'flex' }}>
      <div className="modal-content">
        <h2>🏆 Challenge Complete!</h2>
        <p className="final-score" style={{ marginBottom: '20px' }}>Final Score: {totalScore.toLocaleString()} points</p>

        <div className="challenge-breakdown" style={{ marginBottom: '20px' }}>
          <h3 style={{ marginBottom: '12px' }}>Score Breakdown:</h3>
          <div className="challenge-results">
            {guesses.length === 0 ? (
              <div className="no-results">No guesses recorded</div>
            ) : (
              guesses.map((guess, index) => {
                const car = cars[index];
                if (!car) return null;

                const accuracy = (100 - guess.percentage).toFixed(1);

                return (
                  <div key={index} className="challenge-result-item">
                    <div className="challenge-result-car">
                      {car.year || ''} {car.make || ''} {car.model || 'Unknown Car'}
                    </div>
                    <div className="challenge-result-accuracy">{accuracy}% accurate</div>
                    <div className="challenge-result-points">{(guess.points || 0).toLocaleString()} pts</div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        <div className="modal-buttons">
          {isFriendChallenge ? (
            <>
              {onViewLeaderboard && (
                <button onClick={onViewLeaderboard} className="submit-score-button">
                  View Leaderboard
                </button>
              )}
              <button onClick={onPlayAgain} className="play-again-button">
                Back to Challenges
              </button>
            </>
          ) : (
            <>
              {showSubmitButton && (
                <button onClick={onSubmitScore} className="submit-score-button">
                  Submit to Leaderboard
                </button>
              )}
              <button onClick={onPlayAgain} className="play-again-button">
                New Challenge
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};
