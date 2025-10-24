import { useState, useEffect } from 'react';
import { useAuth } from './hooks/useAuth';
import { TopNav } from './components/common/TopNav';
import { ModeSelection } from './components/game/ModeSelection';
import { GameArea } from './components/game/GameArea';
import { LeaderboardModal } from './components/leaderboard/LeaderboardModal';
import { NameInputModal } from './components/leaderboard/NameInputModal';
import { AuthModals } from './components/auth/AuthModals';
import { ChallengeModals } from './components/challenges/ChallengeModals';
import { Footer } from './components/common/Footer';
import { useGameStore } from './stores/gameStore';

function App() {
  useAuth(); // Initialize auth

  const mode = useGameStore((state) => state.mode);
  const setPendingLeaderboardData = useGameStore((state) => state.setPendingLeaderboardData);
  const resetGame = useGameStore((state) => state.resetGame);

  const [showLeaderboard, setShowLeaderboard] = useState(false);
  const [showNameInput, setShowNameInput] = useState(false);

  // Listen for showNameInputModal event
  useEffect(() => {
    const handleShowNameInput = () => {
      setShowNameInput(true);
    };

    window.addEventListener('showNameInputModal', handleShowNameInput);
    return () => window.removeEventListener('showNameInputModal', handleShowNameInput);
  }, []);

  return (
    <div className="app">
      <div className="container">
        {/* Europe Warning Banner */}
        <div id="europeWarning" className="europe-warning" style={{ display: 'none' }}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          <span>Easy mode images may not load outside Europe due to regional restrictions. Try Hard mode for full functionality worldwide.</span>
          <button onClick={() => {
            const warning = document.getElementById('europeWarning');
            if (warning) warning.style.display = 'none';
          }} className="close-warning">×</button>
        </div>

        <TopNav onOpenLeaderboard={() => setShowLeaderboard(true)} />

        <header>
          <h1>CarGuessr</h1>
        </header>

        {!mode ? <ModeSelection /> : <GameArea />}
      </div>

      <Footer />

      {/* Modals */}
      <AuthModals />
      <ChallengeModals />
      {showLeaderboard && (
        <LeaderboardModal onClose={() => setShowLeaderboard(false)} />
      )}
      {showNameInput && (
        <NameInputModal
          onClose={() => {
            setShowNameInput(false);
            setPendingLeaderboardData(null);
          }}
          onSuccess={() => {
            setShowNameInput(false);
            setPendingLeaderboardData(null);
            resetGame(); // Return to homepage after successful submission
          }}
        />
      )}
    </div>
  );
}

export default App;
