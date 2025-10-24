// Authentication and Friend Challenge JavaScript

// Global variables
let currentUser = null;
let currentChallengeCode = null;

// Initialize authentication state on page load
document.addEventListener('DOMContentLoaded', async () => {
    // FORCE correct initial state - ensure modals are closed
    // This fixes the bug where challengeGuideModal gets stuck open
    const challengeGuideModal = document.getElementById('challengeGuideModal');
    if (challengeGuideModal) {
        challengeGuideModal.style.display = 'none';
    }

    // Ensure mode selection is visible on fresh page load
    const modeSelection = document.getElementById('modeSelection');
    if (modeSelection) {
        modeSelection.style.display = 'block';
    }

    // Ensure game area is hidden initially
    const gameArea = document.getElementById('gameArea');
    if (gameArea) {
        gameArea.style.display = 'none';
    }

    // Close all other modals to ensure clean state
    document.querySelectorAll('.modal').forEach(modal => {
        if (modal.id !== 'challengeGuideModal') {
            modal.style.display = 'none';
        }
    });

    await checkAuthStatus();
});

// Check if user is authenticated
async function checkAuthStatus() {
    const token = localStorage.getItem('sessionToken');
    if (!token) {
        showUnauthenticatedUI();
        return;
    }

    try {
        const response = await fetch('/api/auth/profile', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.ok) {
            const data = await response.json();
            currentUser = data.user;
            currentUser.leaderboardStats = data.leaderboardStats;
            showAuthenticatedUI();
        } else {
            // Token invalid, clear it
            localStorage.removeItem('sessionToken');
            showUnauthenticatedUI();
        }
    } catch (error) {
        console.error('Auth check failed:', error);
        showUnauthenticatedUI();
    }
}

// Show UI for authenticated users
function showAuthenticatedUI() {
    document.getElementById('loginBtn').style.display = 'none';
    document.getElementById('userMenu').style.display = 'block';
    document.getElementById('userDisplayName').textContent = currentUser.displayName || currentUser.username;
    
    // Show friend challenge section, hide signup promotion
    document.getElementById('friendChallengeSection').style.display = 'block';
    document.getElementById('signupPromotionSection').style.display = 'none';
    document.getElementById('challengesBtn').style.display = 'inline-block';
}

// Show UI for unauthenticated users
function showUnauthenticatedUI() {
    document.getElementById('loginBtn').style.display = 'inline-block';
    document.getElementById('userMenu').style.display = 'none';
    document.getElementById('friendChallengeSection').style.display = 'none';
    document.getElementById('signupPromotionSection').style.display = 'block';
    document.getElementById('challengesBtn').style.display = 'none';
}

// Toggle user dropdown menu
function toggleUserDropdown() {
    const dropdown = document.getElementById('userDropdown');
    dropdown.style.display = dropdown.style.display === 'none' ? 'block' : 'none';
}

// Close dropdowns when clicking outside
document.addEventListener('click', (e) => {
    if (!e.target.closest('.user-menu')) {
        document.getElementById('userDropdown').style.display = 'none';
    }
});

// Modal functions
function showLoginModal() {
    closeAllModals();
    document.getElementById('loginModal').style.display = 'flex';
}

function showRegisterModal() {
    closeAllModals();
    document.getElementById('registerModal').style.display = 'flex';
}

function switchToRegister() {
    document.getElementById('loginModal').style.display = 'none';
    document.getElementById('registerModal').style.display = 'flex';
}

function switchToLogin() {
    document.getElementById('registerModal').style.display = 'none';
    document.getElementById('loginModal').style.display = 'flex';
}

function closeModal(modalId) {
    document.getElementById(modalId).style.display = 'none';
}

function closeAllModals() {
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => modal.style.display = 'none');
}

// Authentication handlers
async function handleLogin(event) {
    event.preventDefault();
    
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    
    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        });
        
        const data = await response.json();
        
        if (data.success) {
            localStorage.setItem('sessionToken', data.sessionToken);
            currentUser = data.user;
            closeModal('loginModal');
            showAuthenticatedUI();
            showToast('Login successful!', 'success');
        } else {
            showToast(data.message || 'Login failed', 'error');
        }
    } catch (error) {
        console.error('Login error:', error);
        showToast('Login failed. Please try again.', 'error');
    }
}

async function handleRegister(event) {
    event.preventDefault();
    
    const username = document.getElementById('registerUsername').value;
    const displayName = document.getElementById('registerDisplayName').value;
    const password = document.getElementById('registerPassword').value;
    const securityQuestion = document.getElementById('registerSecurityQuestion').value;
    const securityAnswer = document.getElementById('registerSecurityAnswer').value;
    
    try {
        const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, displayName, password, securityQuestion, securityAnswer })
        });
        
        const data = await response.json();
        
        if (data.success) {
            localStorage.setItem('sessionToken', data.sessionToken);
            currentUser = data.user;
            closeModal('registerModal');
            showAuthenticatedUI();
            showToast('Account created successfully!', 'success');
        } else {
            showToast(data.message || 'Registration failed', 'error');
        }
    } catch (error) {
        console.error('Registration error:', error);
        showToast('Registration failed. Please try again.', 'error');
    }
}


async function logout() {
    try {
        const token = localStorage.getItem('sessionToken');
        if (token) {
            await fetch('/api/auth/logout', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
        }
    } catch (error) {
        console.error('Logout error:', error);
    }
    
    localStorage.removeItem('sessionToken');
    currentUser = null;
    showUnauthenticatedUI();
    location.reload();
}

// Profile functions
async function showProfile() {
    if (!currentUser) {
        showLoginModal();
        return;
    }
    
    closeAllModals();
    document.getElementById('profileModal').style.display = 'flex';
    
    // Populate profile information
    document.getElementById('profileDisplayName').textContent = currentUser.displayName || currentUser.username;
    document.getElementById('profileUsername').textContent = currentUser.username;
    document.getElementById('profileAccountType').textContent = 'Registered Account';
    
    const memberSince = new Date(currentUser.createdAt);
    document.getElementById('profileMemberSince').textContent = memberSince.toLocaleDateString();
    
    // Load user statistics
    await loadUserStats();
}

// Friend Challenge functions
function showCreateChallengeModal() {
    if (!currentUser) {
        showLoginModal();
        return;
    }
    closeAllModals();
    document.getElementById('createChallengeModal').style.display = 'flex';
}

function showJoinChallengeModal() {
    if (!currentUser) {
        showLoginModal();
        return;
    }
    closeAllModals();
    document.getElementById('joinChallengeModal').style.display = 'flex';
}

async function handleCreateChallenge(event) {
    event.preventDefault();
    
    const title = document.getElementById('challengeTitle').value;
    const difficulty = document.getElementById('challengeDifficulty').value;
    const maxParticipants = parseInt(document.getElementById('maxParticipants').value);
    
    try {
        const token = localStorage.getItem('sessionToken');
        const response = await fetch('/api/friends/challenges', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ title, difficulty, maxParticipants })
        });
        
        const data = await response.json();
        
        if (data.success) {
            currentChallengeCode = data.challengeCode;
            document.getElementById('createdChallengeCode').textContent = data.challengeCode;
            document.getElementById('challengeShareMessage').textContent = data.shareMessage;
            
            // Store the creator's session ID for immediate play
            if (data.sessionId) {
                localStorage.setItem('creatorSessionId', data.sessionId);
            }
            
            closeModal('createChallengeModal');
            document.getElementById('challengeCreatedModal').style.display = 'flex';
        } else {
            showToast(data.message || 'Failed to create challenge', 'error');
        }
    } catch (error) {
        console.error('Create challenge error:', error);
        showToast('Failed to create challenge', 'error');
    }
}

async function handleJoinChallenge(event) {
    event.preventDefault();
    
    const code = document.getElementById('challengeCode').value.toUpperCase();
    
    try {
        const token = localStorage.getItem('sessionToken');
        const response = await fetch(`/api/friends/challenges/${code}/join`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        
        if (data.success) {
            closeAllModals();
            showToast(data.message, 'success');
            // Start the challenge with the session ID
            startChallengeWithSession(data.sessionId, data.challenge.difficulty);
        } else {
            showToast(data.message || 'Failed to join challenge', 'error');
        }
    } catch (error) {
        console.error('Join challenge error:', error);
        showToast('Failed to join challenge', 'error');
    }
}

function copyChallengCode() {
    const code = document.getElementById('createdChallengeCode').textContent;
    navigator.clipboard.writeText(code).then(() => {
        showToast('Challenge code copied!', 'success');
    }).catch(() => {
        showToast('Failed to copy code', 'error');
    });
}

async function startChallengeFromCode() {
    if (currentChallengeCode) {
        try {
            // Check if we have a creator session ID stored
            const creatorSessionId = localStorage.getItem('creatorSessionId');
            
            if (creatorSessionId) {
                // Creator starting their own challenge - use stored session ID
                const difficulty = document.getElementById('challengeDifficulty').value;
                closeAllModals();
                startChallengeWithSession(creatorSessionId, difficulty);
                localStorage.removeItem('creatorSessionId'); // Clean up
                return;
            }
            
            // Not the creator - join the challenge normally
            const token = localStorage.getItem('sessionToken');
            const response = await fetch(`/api/friends/challenges/${currentChallengeCode}/join`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            
            const data = await response.json();
            
            if (data.success) {
                closeAllModals();
                // Start the challenge with the session ID
                startChallengeWithSession(data.sessionId, data.challenge.difficulty);
            } else {
                showToast(data.message || 'Failed to start challenge', 'error');
            }
        } catch (error) {
            console.error('Error starting challenge:', error);
            showToast('Failed to start challenge', 'error');
        }
    }
}

// Helper function to start challenge with specific session
function startChallengeWithSession(sessionId, difficulty) {
    // Store session ID for challenge mode
    localStorage.setItem('challengeSessionId', sessionId);
    localStorage.setItem('selectedDifficulty', difficulty);
    
    // Set up game state before starting
    if (window.currentGame) {
        window.currentGame.mode = 'challenge';
        window.currentGame.difficulty = difficulty;
        window.currentGame.sessionId = sessionId; // Use the session from backend
    }
    
    // Hide mode selection and show game area
    document.getElementById('modeSelection').style.display = 'none';
    document.getElementById('gameArea').style.display = 'block';
    document.getElementById('scoreDisplay').style.display = 'inline-block';
    
    // Set score label for challenge mode
    const scoreLabel = document.getElementById('scoreLabel');
    if (scoreLabel) {
        scoreLabel.textContent = 'Score: ';
    }
    
    // Initialize challenge mode with the specific session
    if (window.initializeChallenge) {
        window.initializeChallenge(sessionId);
    }
}

// My Challenges functions
function showMyChallenges() {
    openChallenges();
}

async function openChallenges() {
    if (!currentUser) {
        showLoginModal();
        return;
    }
    
    closeAllModals();
    document.getElementById('myChallengesModal').style.display = 'flex';
    
    // Load challenges
    await loadMyChallenges();
}

async function loadMyChallenges() {
    try {
        const token = localStorage.getItem('sessionToken');
        const response = await fetch('/api/friends/challenges/my-challenges', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        
        if (data.success) {
            displayCreatedChallenges(data.created);
            displayParticipatingChallenges(data.participating);
        } else {
            showToast(data.message || 'Failed to load challenges', 'error');
        }
    } catch (error) {
        console.error('Failed to load challenges:', error);
        showToast('Failed to load challenges', 'error');
        
        // Show error state in both tabs
        document.getElementById('createdChallenges').innerHTML = '<div class="error-state">Failed to load challenges</div>';
        document.getElementById('participatingChallenges').innerHTML = '<div class="error-state">Failed to load challenges</div>';
    }
}

// Challenge display functions
function displayCreatedChallenges(challenges) {
    const container = document.getElementById('createdChallenges');
    
    if (!challenges || challenges.length === 0) {
        container.innerHTML = '<div class="empty-state">You haven\'t created any challenges yet.<br><button onclick="showCreateChallengeModal(); closeModal(\'myChallengesModal\');" class="create-challenge-btn">Create Your First Challenge</button></div>';
        return;
    }
    
    console.log('Displaying created challenges:', challenges); // Debug log
    
    const html = challenges.map(challenge => {
        const createdDate = new Date(challenge.createdAt).toLocaleDateString();
        const expiresDate = new Date(challenge.expiresAt).toLocaleDateString();
        const isExpired = new Date() > new Date(challenge.expiresAt);
        const statusIcon = isExpired ? '⏰' : challenge.isActive ? '🟢' : '🔴';
        const statusText = isExpired ? 'Expired' : challenge.isActive ? 'Active' : 'Inactive';
        
        return `
            <div class="challenge-item ${isExpired ? 'expired' : ''}">
                <div class="challenge-header">
                    <h4>${challenge.title}</h4>
                    <span class="challenge-status">${statusIcon} ${statusText}</span>
                </div>
                <div class="challenge-details">
                    <span class="challenge-code">Code: <strong>${challenge.challengeCode}</strong></span>
                    <span class="challenge-difficulty">${challenge.difficulty === 'easy' ? '🟢 Easy' : '🔴 Hard'}</span>
                    <span class="challenge-participants">${challenge.participantCount || 0}/${challenge.maxParticipants} players</span>
                </div>
                <div class="challenge-dates">
                    <small>Created: ${createdDate} | Expires: ${expiresDate}</small>
                </div>
                <div class="challenge-actions">
                    <button onclick="copyToClipboard('${challenge.challengeCode}')" class="action-btn copy-btn">📋 Copy Code</button>
                    <button onclick="showChallengeLeaderboard('${challenge.challengeCode}')" class="action-btn leaderboard-btn">🏆 Leaderboard</button>
                    ${!isExpired && challenge.isActive && challenge.isComplete !== true ? `<button onclick="resumeChallenge('${challenge.challengeCode}')" class="action-btn resume-btn">▶️ Play</button>` : ''}
                    ${!isExpired && challenge.isActive && challenge.isComplete !== true ? `<button onclick="shareChallenge('${challenge.challengeCode}', '${challenge.title}')" class="action-btn share-btn">📤 Share</button>` : ''}
                    ${challenge.isComplete === true ? `<span class="completion-status">✅ Completed</span>` : ''}
                </div>
            </div>
        `;
    }).join('');
    
    container.innerHTML = html;
}

function displayParticipatingChallenges(challenges) {
    const container = document.getElementById('participatingChallenges');
    
    if (!challenges || challenges.length === 0) {
        container.innerHTML = '<div class="empty-state">You haven\'t joined any challenges yet.<br><button onclick="showJoinChallengeModal(); closeModal(\'myChallengesModal\');" class="join-challenge-btn">Join a Challenge</button></div>';
        return;
    }
    
    console.log('Displaying participating challenges:', challenges); // Debug log
    
    const html = challenges.map(challenge => {
        // Handle date parsing more safely
        const joinedDate = challenge.joinedAt ? new Date(challenge.joinedAt).toLocaleDateString() : 'Unknown';
        const expiresDate = challenge.expiresAt ? new Date(challenge.expiresAt).toLocaleDateString() : 'Unknown';
        const isExpired = challenge.expiresAt ? new Date() > new Date(challenge.expiresAt) : false;
        const statusIcon = isExpired ? '⏰' : challenge.isActive ? '🟢' : '🔴';
        const statusText = isExpired ? 'Expired' : challenge.isActive ? 'Active' : 'Inactive';
        
        return `
            <div class="challenge-item ${isExpired ? 'expired' : ''}">
                <div class="challenge-header">
                    <h4>${challenge.title}</h4>
                    <span class="challenge-status">${statusIcon} ${statusText}</span>
                </div>
                <div class="challenge-details">
                    <span class="challenge-creator">By: ${challenge.creatorDisplayName}</span>
                    <span class="challenge-code">Code: <strong>${challenge.challengeCode}</strong></span>
                    <span class="challenge-difficulty">${challenge.difficulty === 'easy' ? '🟢 Easy' : '🔴 Hard'}</span>
                </div>
                <div class="challenge-dates">
                    <small>Joined: ${joinedDate} | Expires: ${expiresDate}</small>
                </div>
                <div class="challenge-actions">
                    <button onclick="showChallengeLeaderboard('${challenge.challengeCode}')" class="action-btn leaderboard-btn">🏆 Leaderboard</button>
                    ${!isExpired && challenge.isActive && challenge.isComplete !== true ? `<button onclick="resumeChallenge('${challenge.challengeCode}')" class="action-btn resume-btn">▶️ Resume</button>` : ''}
                    ${challenge.isComplete === true ? `<span class="completion-status">✅ Completed</span>` : ''}
                </div>
            </div>
        `;
    }).join('');
    
    container.innerHTML = html;
}

// Tab switching for My Challenges
function switchChallengeTab(tab) {
    // Update tab buttons
    document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');
    
    // Show/hide appropriate content
    if (tab === 'created') {
        document.getElementById('createdChallenges').style.display = 'block';
        document.getElementById('participatingChallenges').style.display = 'none';
    } else {
        document.getElementById('createdChallenges').style.display = 'none';
        document.getElementById('participatingChallenges').style.display = 'block';
    }
}

// Helper functions for challenge actions
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('Challenge code copied!', 'success');
    }).catch(() => {
        showToast('Failed to copy code', 'error');
    });
}

function shareChallenge(code, title) {
    const shareText = `Join my CarGuessr challenge "${title}"! Use code: ${code}`;
    
    if (navigator.share) {
        navigator.share({
            title: `CarGuessr Challenge: ${title}`,
            text: shareText,
            url: window.location.href
        });
    } else {
        copyToClipboard(`${shareText}\n\nPlay at: ${window.location.href}`);
        showToast('Challenge details copied to clipboard!', 'success');
    }
}

async function resumeChallenge(code) {
    try {
        // First check user's participation status
        const token = localStorage.getItem('sessionToken');
        const participationResponse = await fetch(`/api/friends/challenges/${code}/participation`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (participationResponse.ok) {
            const participationData = await participationResponse.json();
            if (participationData.success && participationData.session) {
                // User already has a session - check if it's complete
                const session = participationData.session;
                if (session.isComplete) {
                    showToast('You have already completed this challenge!', 'info');
                    return;
                }
                
                // Resume existing session
                closeAllModals();
                startChallengeWithSession(session.sessionId, participationData.challenge.difficulty);
                return;
            }
        }
        
        // Fallback: Try to join the challenge (shouldn't happen if we reach here)
        const response = await fetch(`/api/friends/challenges/${code}/join`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        
        if (data.success) {
            closeAllModals();
            startChallengeWithSession(data.sessionId, data.challenge.difficulty);
        } else {
            if (data.message && data.message.includes('already participating')) {
                showToast('You have already completed this challenge!', 'info');
            } else {
                showToast(data.message || 'Failed to resume challenge', 'error');
            }
        }
    } catch (error) {
        console.error('Error resuming challenge:', error);
        showToast('Failed to resume challenge', 'error');
    }
}

// User statistics loading
async function loadUserStats() {
    try {
        // Fetch fresh profile data to ensure leaderboard stats are up-to-date
        const token = localStorage.getItem('sessionToken');
        const profileResponse = await fetch('/api/auth/profile', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (profileResponse.ok) {
            const profileData = await profileResponse.json();
            // Update currentUser with fresh data
            currentUser = profileData.user;
            currentUser.leaderboardStats = profileData.leaderboardStats;
        }
        
        // Display basic stats from the user object
        document.getElementById('totalGamesPlayed').textContent = currentUser.totalGamesPlayed || 0;
        // Display favorite difficulty with proper capitalization
        const favDifficulty = currentUser.favoriteDifficulty;
        let displayText = 'None yet';
        if (favDifficulty === 'easy') {
            displayText = 'Easy';
        } else if (favDifficulty === 'hard') {
            displayText = 'Hard';
        }
        document.getElementById('favoriteDifficulty').textContent = displayText;
        
        // Display leaderboard ranks (both registered and overall)
        const leaderboardStats = currentUser.leaderboardStats || {};
        document.getElementById('challengeEasyRegisteredRank').textContent = leaderboardStats.challenge_easy_registered_rank || 'N/A';
        document.getElementById('challengeEasyOverallRank').textContent = leaderboardStats.challenge_easy_overall_rank || 'N/A';
        document.getElementById('challengeHardRegisteredRank').textContent = leaderboardStats.challenge_hard_registered_rank || 'N/A';
        document.getElementById('challengeHardOverallRank').textContent = leaderboardStats.challenge_hard_overall_rank || 'N/A';
        
        // Fetch challenges data
        const response = await fetch('/api/friends/challenges/my-challenges', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        const data = await response.json();
        if (data.success) {
            document.getElementById('challengesCreated').textContent = data.created?.length || 0;
            document.getElementById('challengesJoined').textContent = data.participating?.length || 0;
        }
    } catch (error) {
        console.error('Failed to load user stats:', error);
        // Don't show error toast for stats as it's not critical
    }
}

// Challenge leaderboard functions
async function showChallengeLeaderboard(code) {
    currentChallengeCode = code;
    document.getElementById('challengeCodeDisplay').textContent = code;
    
    // Close My Challenges modal if it's open to prevent layering issues
    closeModal('myChallengesModal');
    
    document.getElementById('challengeLeaderboardModal').style.display = 'flex';
    await refreshChallengeLeaderboard();
}

async function refreshChallengeLeaderboard() {
    if (!currentChallengeCode) return;
    
    try {
        const response = await fetch(`/api/friends/challenges/${currentChallengeCode}/leaderboard`);
        const data = await response.json();
        
        if (data.success) {
            displayChallengeParticipants(data.participants, data.challenge);
        }
    } catch (error) {
        console.error('Failed to load leaderboard:', error);
        showToast('Failed to load leaderboard', 'error');
    }
}

function displayChallengeParticipants(participants, challenge) {
    const container = document.getElementById('challengeParticipants');
    container.innerHTML = '';
    
    if (participants.length === 0) {
        container.innerHTML = '<p class="no-participants">No participants yet</p>';
        return;
    }
    
    const html = participants.map((p, index) => {
        const rankEmoji = index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : `${index + 1}.`;
        const statusIcon = p.isComplete ? '✅' : '⏳';
        
        return `
            <div class="participant-row ${index < 3 ? 'top-three' : ''}">
                <span class="rank">${rankEmoji}</span>
                <span class="name">${p.userDisplayName || p.displayName || p.username || 'Anonymous'}</span>
                <span class="score">${p.finalScore !== null ? p.finalScore : '-'}</span>
                <span class="status">${statusIcon}</span>
            </div>
        `;
    }).join('');
    
    container.innerHTML = html;
    document.getElementById('challengeLeaderboardTitle').textContent = `🏆 ${challenge.title}`;
}

// Toast notification system
function showToast(message, type = 'info') {
    // Remove existing toast
    const existingToast = document.querySelector('.toast');
    if (existingToast) {
        existingToast.remove();
    }
    
    // Create new toast
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    
    // Add to body
    document.body.appendChild(toast);
    
    // Trigger animation
    setTimeout(() => toast.classList.add('show'), 10);
    
    // Remove after 3 seconds
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Add toast styles dynamically
const toastStyles = `
    .toast {
        position: fixed;
        bottom: 20px;
        right: 20px;
        padding: 12px 24px;
        background: #333;
        color: white;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        opacity: 0;
        transform: translateY(20px);
        transition: all 0.3s ease;
        z-index: 3000;
        max-width: 300px;
    }
    
    .toast.show {
        opacity: 1;
        transform: translateY(0);
    }
    
    .toast-success {
        background: #4CAF50;
    }
    
    .toast-error {
        background: #f44336;
    }
    
    .toast-info {
        background: #2196F3;
    }
    
    @media (max-width: 768px) {
        .toast {
            bottom: 60px;
            right: 10px;
            left: 10px;
            max-width: none;
        }
    }
`;

// Add styles to document
const styleSheet = document.createElement('style');
styleSheet.textContent = toastStyles;
document.head.appendChild(styleSheet);

// Export functions for use in game.js
// Password reset functions
function showPasswordReset() {
    closeAllModals();
    document.getElementById('passwordResetModal').style.display = 'flex';
    // Reset form state
    document.getElementById('securityQuestionSection').style.display = 'none';
    document.getElementById('verifyUserBtn').style.display = 'inline-block';
    document.getElementById('resetPasswordBtn').style.display = 'none';
    document.getElementById('passwordResetForm').reset();
}

async function verifyUserForReset() {
    const username = document.getElementById('resetUsername').value;
    const displayName = document.getElementById('resetDisplayName').value;
    
    if (!username || !displayName) {
        showToast('Please enter both username and display name', 'error');
        return;
    }
    
    try {
        const response = await fetch('/api/auth/security-question', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, displayName })
        });
        
        const data = await response.json();
        
        if (data.success) {
            // Show the actual security question the user set
            document.getElementById('securityQuestionText').textContent = data.securityQuestion;
            document.getElementById('securityQuestionSection').style.display = 'block';
            document.getElementById('verifyUserBtn').style.display = 'none';
            document.getElementById('resetPasswordBtn').style.display = 'inline-block';
        } else {
            showToast(data.message || 'User not found with those credentials', 'error');
        }
    } catch (error) {
        console.error('Verification error:', error);
        showToast('Failed to verify user. Please try again.', 'error');
    }
}

async function handlePasswordReset(event) {
    event.preventDefault();
    
    const username = document.getElementById('resetUsername').value;
    const displayName = document.getElementById('resetDisplayName').value;
    const securityAnswer = document.getElementById('resetSecurityAnswer').value;
    const newPassword = document.getElementById('resetNewPassword').value;
    
    try {
        const response = await fetch('/api/auth/reset-password', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, displayName, securityAnswer, newPassword })
        });
        
        const data = await response.json();
        
        if (data.success) {
            showToast('Password reset successfully! You can now login.', 'success');
            closeModal('passwordResetModal');
            showLoginModal();
        } else {
            showToast(data.message || 'Password reset failed', 'error');
        }
    } catch (error) {
        console.error('Password reset error:', error);
        showToast('Password reset failed. Please try again.', 'error');
    }
}

// Challenge guide function
function showChallengeGuide() {
    closeAllModals();
    document.getElementById('challengeGuideModal').style.display = 'flex';
}

// Home button function - ensures clean state on navigation to home
function goHome() {
    // Close ALL modals explicitly
    closeAllModals();

    // Explicitly hide the challenge guide modal to prevent it from persisting
    const challengeGuideModal = document.getElementById('challengeGuideModal');
    if (challengeGuideModal) {
        challengeGuideModal.style.display = 'none';
    }

    // Hide game area if showing
    const gameArea = document.getElementById('gameArea');
    if (gameArea) {
        gameArea.style.display = 'none';
    }

    // Show mode selection
    const modeSelection = document.getElementById('modeSelection');
    if (modeSelection) {
        modeSelection.style.display = 'block';
    }

    // Hard reload with cache-busting to ensure fresh state
    // Use window.location.href instead of reload() to force a clean navigation
    window.location.href = window.location.origin + window.location.pathname + '?t=' + Date.now();
}

window.authFunctions = {
    checkAuthStatus,
    showChallengeLeaderboard,
    currentUser: () => currentUser
};