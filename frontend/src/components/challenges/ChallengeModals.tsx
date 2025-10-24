import { useEffect, useState } from 'react';
import { useGameStore } from '../../stores/gameStore';
import { apiClient } from '../../api/client';
import { showToast } from '../../utils/toast';
import { CreateChallengeModal } from './CreateChallengeModal';
import { ChallengeCreatedModal } from './ChallengeCreatedModal';
import { JoinChallengeModal } from './JoinChallengeModal';
import { MyChallengesModal } from './MyChallengesModal';
import { ChallengeLeaderboardModal } from './ChallengeLeaderboardModal';
import { ChallengeGuideModal } from './ChallengeGuideModal';
import type { Difficulty } from '../../types';

type ChallengeModalType = 'create' | 'created' | 'join' | 'myChallenges' | 'leaderboard' | 'guide' | null;

export const ChallengeModals = () => {
  const [activeModal, setActiveModal] = useState<ChallengeModalType>(null);
  const [createdChallengeCode, setCreatedChallengeCode] = useState('');
  const [createdSessionId, setCreatedSessionId] = useState('');
  const [selectedChallengeCode, setSelectedChallengeCode] = useState('');

  const {
    setMode,
    setDifficulty,
    setChallengeSession,
    setCurrentListing,
    setChallengeCode
  } = useGameStore();

  useEffect(() => {
    const handleShowCreate = () => setActiveModal('create');
    const handleShowJoin = () => setActiveModal('join');
    const handleShowMyChallenges = () => setActiveModal('myChallenges');
    const handleShowGuide = () => setActiveModal('guide');
    const handleShowLeaderboardEvent = (e: Event) => {
      const customEvent = e as CustomEvent;
      const code = customEvent.detail?.challengeCode;
      if (code) {
        setSelectedChallengeCode(code);
        setActiveModal('leaderboard');
      }
    };

    window.addEventListener('showCreateChallengeModal', handleShowCreate);
    window.addEventListener('showJoinChallengeModal', handleShowJoin);
    window.addEventListener('showMyChallenges', handleShowMyChallenges);
    window.addEventListener('showChallengeGuide', handleShowGuide);
    window.addEventListener('showChallengeLeaderboard', handleShowLeaderboardEvent);

    return () => {
      window.removeEventListener('showCreateChallengeModal', handleShowCreate);
      window.removeEventListener('showJoinChallengeModal', handleShowJoin);
      window.removeEventListener('showMyChallenges', handleShowMyChallenges);
      window.removeEventListener('showChallengeGuide', handleShowGuide);
      window.removeEventListener('showChallengeLeaderboard', handleShowLeaderboardEvent);
    };
  }, []);

  const closeModal = () => {
    setActiveModal(null);
    setCreatedChallengeCode('');
    setCreatedSessionId('');
    setSelectedChallengeCode('');
  };

  const handleCreateSuccess = (challengeCode: string, sessionId: string) => {
    setCreatedChallengeCode(challengeCode);
    setCreatedSessionId(sessionId);
    setActiveModal('created');
  };

  const startChallengeWithSession = async (sessionId: string, difficulty: Difficulty, challengeCode?: string) => {
    try {
      // Get the challenge session from backend
      const session = await apiClient.getChallengeSession(sessionId);

      // Ensure guesses is always an array
      if (!session.guesses) {
        session.guesses = [];
      }

      // Set up game state
      setMode('challenge');
      setDifficulty(difficulty);
      setChallengeSession(session);
      setChallengeCode(challengeCode || null); // Store challenge code for friend challenges

      if (session.cars && session.cars.length > 0) {
        setCurrentListing(session.cars[session.currentCar || 0]);
      }

      closeModal();
    } catch (error) {
      console.error('Error starting challenge:', error);
      showToast('Failed to start challenge', 'error');
    }
  };

  const handleJoinSuccess = (sessionId: string, difficulty: string, challengeCode: string) => {
    closeModal();
    startChallengeWithSession(sessionId, difficulty as Difficulty, challengeCode);
  };

  const handleResumeChallenge = async (challengeCode: string) => {
    closeModal();

    try {
      // Check participation status
      const participationData = await apiClient.getChallengeParticipation(challengeCode);

      if (participationData.success && participationData.session) {
        const session = participationData.session;

        if (session.isComplete) {
          showToast('You have already completed this challenge!', 'info');
          return;
        }

        // Resume existing session
        startChallengeWithSession(session.sessionId, participationData.challenge.difficulty as Difficulty, challengeCode);
        return;
      }

      // Fallback: Try to join the challenge
      const joinData = await apiClient.joinChallenge(challengeCode);
      startChallengeWithSession(joinData.sessionId, joinData.challenge.difficulty as Difficulty, challengeCode);
    } catch (error) {
      console.error('Error resuming challenge:', error);
      const message = error instanceof Error ? error.message : 'Failed to resume challenge';
      showToast(message, 'error');
    }
  };

  const handleShowLeaderboard = (challengeCode: string) => {
    setSelectedChallengeCode(challengeCode);
    setActiveModal('leaderboard');
  };

  return (
    <>
      {activeModal === 'create' && (
        <CreateChallengeModal
          onClose={closeModal}
          onSuccess={handleCreateSuccess}
        />
      )}

      {activeModal === 'created' && createdChallengeCode && (
        <ChallengeCreatedModal
          challengeCode={createdChallengeCode}
          sessionId={createdSessionId}
          onClose={closeModal}
          onStartChallenge={(sessionId) => {
            const difficulty = useGameStore.getState().difficulty;
            startChallengeWithSession(sessionId, difficulty, createdChallengeCode);
          }}
        />
      )}

      {activeModal === 'join' && (
        <JoinChallengeModal
          onClose={closeModal}
          onSuccess={handleJoinSuccess}
        />
      )}

      {activeModal === 'myChallenges' && (
        <MyChallengesModal
          onClose={closeModal}
          onResumeChallenge={handleResumeChallenge}
          onShowLeaderboard={handleShowLeaderboard}
          onShowCreateModal={() => setActiveModal('create')}
          onShowJoinModal={() => setActiveModal('join')}
        />
      )}

      {activeModal === 'leaderboard' && selectedChallengeCode && (
        <ChallengeLeaderboardModal
          challengeCode={selectedChallengeCode}
          onClose={() => {
            setSelectedChallengeCode('');
            setActiveModal(null);
            // Reload page to reset game state and return to homepage
            window.location.reload();
          }}
        />
      )}

      {activeModal === 'guide' && (
        <ChallengeGuideModal onClose={closeModal} />
      )}
    </>
  );
};
