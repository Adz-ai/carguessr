import { useEffect, useState } from 'react';
import { useGameStore } from '../../stores/gameStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';
import { CarDisplay } from './CarDisplay';
import { PriceInput } from './PriceInput';
import { ResultModal } from './ResultModal';
import { GameOverModal } from './GameOverModal';
import { ChallengeCompleteModal } from './ChallengeCompleteModal';
import type { GuessResult } from '../../types';

export const GameArea = () => {
  const {
    mode,
    difficulty,
    currentListing,
    score,
    sessionId,
    challengeSession,
    challengeGuesses,
    challengeCode,
    setCurrentListing,
    setScore,
    setChallengeSession,
    addChallengeGuess,
    setPendingLeaderboardData
  } = useGameStore();

  const [showResult, setShowResult] = useState(false);
  const [currentResult, setCurrentResult] = useState<GuessResult | null>(null);
  const [showGameOver, setShowGameOver] = useState(false);
  const [showChallengeComplete, setShowChallengeComplete] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Load first car on mount
  useEffect(() => {
    if (!mode) return;

    setIsLoading(true);
    if (mode === 'challenge') {
      // Only start a new challenge if we don't already have a session (friend challenge)
      if (!challengeSession) {
        loadChallengeMode();
      } else {
        // Friend challenge - session already loaded
        setIsLoading(false);
      }
    } else {
      loadNextCar();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode]); // Don't add challengeSession to dependencies! It would cause re-initialization

  const loadChallengeMode = async () => {
    try {
      const session = await apiClient.startChallenge(difficulty);

      // Ensure guesses is always an array
      if (!session.guesses) {
        session.guesses = [];
      }

      setChallengeSession(session);
      if (session.cars && session.cars.length > 0) {
        setCurrentListing(session.cars[0]);
      }
    } catch (error) {
      console.error('Error starting challenge:', error);
      showToast('Failed to start challenge mode', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const loadNextCar = async () => {
    try {
      const car = await apiClient.getRandomCar(difficulty, sessionId);
      setCurrentListing(car);
    } catch (error) {
      console.error('Error loading car:', error);
      showToast('Failed to load car listing', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const loadNextChallengeCar = () => {
    if (!challengeSession) return;

    if (challengeSession.currentCar < challengeSession.cars.length) {
      const nextCar = challengeSession.cars[challengeSession.currentCar];
      setCurrentListing(nextCar);
    }
  };

  const handleGuessSubmit = async (guessedPrice: number) => {
    if (!currentListing) return;

    setIsSubmitting(true);

    try {
      if (mode === 'challenge') {
        await handleChallengeGuess(guessedPrice);
      } else {
        await handleRegularGuess(guessedPrice);
      }
    } catch (error) {
      console.error('Error submitting guess:', error);
      showToast('Failed to submit guess', 'error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChallengeGuess = async (guessedPrice: number) => {
    if (!challengeSession) return;

    const result = await apiClient.submitChallengeGuess(challengeSession.sessionId, guessedPrice);

    addChallengeGuess(result);
    setScore(result.totalScore || 0);

    // Update challenge session
    const updatedSession = {
      ...challengeSession,
      currentCar: challengeSession.currentCar + 1,
      totalScore: result.totalScore || 0,
      guesses: [...(challengeSession.guesses || []), result],
      isComplete: result.isLastCar || result.sessionComplete || false
    };
    setChallengeSession(updatedSession);

    setCurrentResult(result);
    setShowResult(true);
  };

  const handleRegularGuess = async (guessedPrice: number) => {
    if (!currentListing) return;

    const result = await apiClient.submitGuess({
      listingId: currentListing.id,
      guessedPrice,
      gameMode: mode!,
      difficulty,
      sessionId
    });

    setScore(result.score);
    setCurrentResult(result);
    setShowResult(true);

    if (result.gameOver) {
      // Will show game over modal after result modal
    }
  };

  const handleNextRound = () => {
    setShowResult(false);

    if (currentResult?.gameOver) {
      // Show game over modal
      setShowGameOver(true);
      return;
    }

    if (mode === 'challenge') {
      if (challengeSession?.isComplete) {
        setShowChallengeComplete(true);
      } else {
        loadNextChallengeCar();
        scrollToTop();
      }
    } else {
      loadNextCar();
      scrollToTop();
    }
  };

  const handleEndGame = () => {
    window.location.reload();
  };

  const handleSubmitScore = () => {
    if (mode === 'challenge' && challengeSession) {
      setPendingLeaderboardData({
        gameMode: mode,
        score: challengeSession.totalScore,
        sessionId: challengeSession.sessionId,
        difficulty: challengeSession.difficulty
      });
    } else {
      setPendingLeaderboardData({
        gameMode: mode!,
        score,
        sessionId,
        difficulty
      });
    }

    // Trigger name input modal
    window.dispatchEvent(new CustomEvent('showNameInputModal'));

    // Close current modals
    setShowGameOver(false);
    setShowChallengeComplete(false);
  };

  const handleViewChallengeLeaderboard = () => {
    // Close challenge complete modal first
    setShowChallengeComplete(false);

    // Small delay to let the modal close before opening leaderboard
    setTimeout(() => {
      // Trigger challenge leaderboard modal with current challenge code
      window.dispatchEvent(new CustomEvent('showChallengeLeaderboard', {
        detail: { challengeCode }
      }));
    }, 100);
  };

  const scrollToTop = () => {
    setTimeout(() => {
      const carImage = document.getElementById('mainCarImage');
      if (carImage) {
        carImage.scrollIntoView({
          behavior: 'smooth',
          block: 'start',
          inline: 'nearest'
        });
      }
    }, 300);
  };

  if (isLoading || !currentListing) {
    return (
      <div className="game-area" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '400px',
        fontSize: '1.5rem',
        color: '#fff'
      }}>
        <div className="loading-state">
          <div className="loading"></div>
          <p style={{ marginTop: '20px' }}>Loading car listing...</p>
        </div>
      </div>
    );
  }

  const currentCar = challengeSession ? challengeSession.currentCar : 0;
  const totalCars = challengeSession ? challengeSession.cars.length : 10;
  const displayCurrentCar = Math.min(currentCar + 1, totalCars);

  return (
    <>
      <div id="gameArea" className="game-area">
        {/* Challenge Progress */}
        {mode === 'challenge' && challengeSession && !challengeSession.isComplete && (
          <div className="car-info">
            <div className="challenge-progress" id="challengeProgress">
              <h4>Challenge Progress</h4>
              <p>Car {displayCurrentCar} of {totalCars}</p>
              <div className="challenge-progress-bar">
                <div
                  className="challenge-progress-fill"
                  style={{ width: `${(displayCurrentCar / totalCars) * 100}%` }}
                ></div>
              </div>
              <p>Current Score: <span id="challengeCurrentScore">{challengeSession.totalScore.toLocaleString()}</span> points</p>
            </div>
          </div>
        )}

        <CarDisplay car={currentListing} />
        <PriceInput
          onSubmit={handleGuessSubmit}
          disabled={isSubmitting || (mode === 'challenge' && challengeSession?.isComplete === true)}
          resetTrigger={currentListing}
        />
      </div>

      {/* Modals */}
      {showResult && currentResult && (
        <ResultModal
          result={currentResult}
          onNext={handleNextRound}
          onEnd={handleEndGame}
          onSubmitScore={() => {
            setShowResult(false);
            handleSubmitScore();
          }}
          isChallenge={mode === 'challenge'}
          currentCar={currentCar}
          totalCars={totalCars}
          gameMode={mode || undefined}
          currentScore={score}
        />
      )}

      {showGameOver && (
        <GameOverModal
          score={score}
          mode={mode!}
          onSubmitScore={handleSubmitScore}
          onPlayAgain={handleEndGame}
        />
      )}

      {showChallengeComplete && challengeSession && (
        <ChallengeCompleteModal
          totalScore={challengeSession.totalScore}
          guesses={challengeGuesses}
          cars={challengeSession.cars}
          challengeCode={challengeCode}
          onSubmitScore={handleSubmitScore}
          onViewLeaderboard={challengeCode ? handleViewChallengeLeaderboard : undefined}
          onPlayAgain={handleEndGame}
        />
      )}
    </>
  );
};
