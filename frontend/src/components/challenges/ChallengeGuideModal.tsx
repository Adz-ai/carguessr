interface ChallengeGuideModalProps {
  onClose: () => void;
}

export const ChallengeGuideModal = ({ onClose }: ChallengeGuideModalProps) => {
  return (
    <div className="modal" style={{ display: 'flex' }} onClick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}>
      <div className="modal-content guide-modal">
        <button className="modal-close" onClick={onClose}>&times;</button>
        <h2>How Friend Challenges Work</h2>

        <div className="guide-section">
          <h3>🎯 Overview</h3>
          <p>
            Friend Challenges let you compete with friends on the exact same set of cars! Create a challenge,
            share the code, and see who can guess prices the most accurately.
          </p>
        </div>

        <div className="guide-section">
          <h3>📝 Creating a Challenge</h3>
          <ol>
            <li>Click "Create Friend Challenge"</li>
            <li>Give your challenge a fun name</li>
            <li>Choose difficulty (Easy or Hard mode)</li>
            <li>Set max participants (2-100 players)</li>
            <li>Share the generated code with friends!</li>
          </ol>
        </div>

        <div className="guide-section">
          <h3>🎮 Joining a Challenge</h3>
          <ol>
            <li>Get a challenge code from a friend</li>
            <li>Click "Join with Code"</li>
            <li>Enter the 6-character code</li>
            <li>Start guessing! You'll see the exact same cars</li>
          </ol>
        </div>

        <div className="guide-section">
          <h3>🏆 Scoring</h3>
          <p>
            Challenges use GeoGuessr-style scoring. Each car is worth up to 5,000 points based on accuracy:
          </p>
          <ul>
            <li><strong>Perfect guess:</strong> 5,000 points</li>
            <li><strong>Within 1%:</strong> ~4,900 points</li>
            <li><strong>Within 5%:</strong> ~4,000 points</li>
            <li><strong>Within 10%:</strong> ~2,500 points</li>
            <li><strong>Further away:</strong> Fewer points</li>
          </ul>
          <p>Total 10 cars = maximum 50,000 points possible!</p>
        </div>

        <div className="guide-section">
          <h3>⏰ Important Notes</h3>
          <ul>
            <li>Challenges expire after 7 days</li>
            <li>You can only complete each challenge once</li>
            <li>All participants see the same cars in the same order</li>
            <li>Your score is final once you complete all 10 cars</li>
            <li>Check the leaderboard anytime to see rankings</li>
          </ul>
        </div>

        <div className="modal-buttons">
          <button onClick={onClose} className="close-button">Got It!</button>
        </div>
      </div>
    </div>
  );
};
