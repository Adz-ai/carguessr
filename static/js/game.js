// Game state
let currentGame = {
    mode: null,
    difficulty: 'easy', // Default to easy mode
    currentListing: null,
    score: 0,
    sessionId: generateSessionId(),
    challengeSession: null,
    challengeGuesses: [],
    pendingLeaderboardData: null, // Store data for leaderboard submission
    leaderboardShownAfterSubmission: false // Track if leaderboard was shown after score submission
};

// Leaderboard state
let leaderboardState = {
    currentGameMode: 'challenge',
    currentDifficulty: 'easy'
};

// Generate a session ID for tracking scores
function generateSessionId() {
    const letters = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let result = '';
    for (let i = 0; i < 16; i++) {
        result += letters.charAt(Math.floor(Math.random() * letters.length));
    }
    return result;
}

// LocalStorage functions for difficulty preference
function saveDifficultyPreference(difficulty) {
    try {
        localStorage.setItem('carguessr_difficulty', difficulty);
    } catch (e) {
        console.log('Unable to save difficulty preference:', e);
    }
}

function loadDifficultyPreference() {
    try {
        return localStorage.getItem('carguessr_difficulty') || 'easy';
    } catch (e) {
        console.log('Unable to load difficulty preference:', e);
        return 'easy';
    }
}

// Select difficulty mode
function selectDifficulty(difficulty) {
    currentGame.difficulty = difficulty;
    leaderboardState.currentDifficulty = difficulty; // Update leaderboard state too
    
    // Save preference to localStorage
    saveDifficultyPreference(difficulty);
    
    // Update button states
    document.getElementById('easyButton').classList.remove('active');
    document.getElementById('hardButton').classList.remove('active');
    
    if (difficulty === 'easy') {
        document.getElementById('easyButton').classList.add('active');
        document.getElementById('difficultyDescription').innerHTML = 
            '<strong>Easy Mode:</strong> Modern used cars from Lookers dealership. Realistic pricing from a major UK car dealer - perfect for beginners!';
    } else {
        document.getElementById('hardButton').classList.add('active');
        document.getElementById('difficultyDescription').innerHTML = 
            '<strong>Hard Mode:</strong> Classic and exotic cars from Bonhams auction house. Higher value cars with unique characteristics - the ultimate challenge!';
    }
}

// Start the game with selected mode
function startGame(mode) {
    currentGame.mode = mode;
    currentGame.score = 0;
    
    // Hide mode selection, show game area
    document.getElementById('modeSelection').style.display = 'none';
    document.getElementById('gameArea').style.display = 'block';
    document.getElementById('scoreDisplay').style.display = 'inline-block';
    
    // Update score label based on mode
    const scoreLabel = document.getElementById('scoreLabel');
    if (mode === 'zero') {
        scoreLabel.textContent = 'Total Difference: £';
    } else if (mode === 'streak') {
        scoreLabel.textContent = 'Streak: ';
    } else if (mode === 'challenge') {
        scoreLabel.textContent = 'Score: ';
    }
    
    // Data source info removed - leaderboard button now takes that space
    
    // Handle challenge mode differently
    if (mode === 'challenge') {
        startChallengeMode();
    } else {
        // Load first car for other modes
        loadNextCar();
    }
}

// Load a random car listing
async function loadNextCar() {
    try {
        // Include difficulty parameter in the request
        const difficultyParam = currentGame.difficulty ? `?difficulty=${currentGame.difficulty}` : '';
        
        // Prepare headers with session ID for car history tracking
        const headers = {};
        if (currentGame.sessionId) {
            headers['X-Session-ID'] = currentGame.sessionId;
        }
        
        let response = await fetch(`/api/random-enhanced-listing${difficultyParam}`, {
            headers: headers
        });
        
        if (!response.ok) {
            console.log('Enhanced listing not available, falling back to standard');
            response = await fetch('/api/random-listing', {
                headers: headers
            });
            if (!response.ok) throw new Error('Failed to load listing');
        }
        
        const listing = await response.json();
        currentGame.currentListing = listing;
        displayCar(listing);
        
        // Reset guess inputs to be synchronized
        const priceInput = document.getElementById('priceGuess');
        const priceOverlay = document.getElementById('priceOverlay');
        
        priceInput.value = '';
        document.getElementById('priceSlider').value = 50; // Set to £50k (middle of first range)
        
        // Set placeholder styling
        priceOverlay.textContent = 'Enter price';
        priceOverlay.style.color = 'rgba(255, 255, 255, 0.5)';
        priceOverlay.style.fontStyle = 'italic';
        
    } catch (error) {
        console.error('Error loading car:', error);
        alert('Failed to load car listing. Please try again.');
    }
}

// Display car information
function displayCar(car) {
    // Animate car info area
    const carDisplay = document.querySelector('.car-display');
    carDisplay.style.opacity = '0';
    carDisplay.style.transform = 'translateY(20px)';
    
    setTimeout(() => {
        // Set up image gallery
        setupImageGallery(car.images || []);
        
        // Set car title - use cleaned title for Easy mode if available
        let displayTitle;
        if (!car.auctionDetails && car.fullTitle && car.fullTitle.includes(' - ')) {
            // Easy mode: extract main part before hyphen
            const parts = car.fullTitle.split(' - ');
            displayTitle = parts[0];
        } else {
            // Hard mode or fallback: use standard format
            displayTitle = `${car.year || 'Unknown'} ${car.make || 'Unknown'} ${car.model || 'Unknown'}`;
        }
        document.getElementById('carTitle').textContent = displayTitle;
        
        // Handle trim display for Easy mode
        const trimRow = document.getElementById('trimRow');
        const trimElement = document.getElementById('carTrim');
        if (!car.auctionDetails && car.trim && car.trim !== '') {
            trimElement.textContent = car.trim;
            trimRow.style.display = 'flex';
        } else {
            trimRow.style.display = 'none';
        }
        
        // Set car details with staggered animation
        const details = [
            { id: 'carYear', value: car.year || 'Unknown' },
            { id: 'carEngine', value: car.engine || 'Unknown' },
            { id: 'carMileage', value: car.mileageFormatted || (car.mileage ? car.mileage.toLocaleString() + ' miles' : 'Unknown') },
            { id: 'carTrim', value: car.trim || 'Unknown' },
            { id: 'carOwners', value: car.owners || 'Unknown' },
            { id: 'carFuelType', value: car.fuelType || 'Unknown' },
            { id: 'carBodyType', value: car.bodyType || 'Unknown' },
            { id: 'carGearbox', value: car.gearbox || 'Unknown' },
            { id: 'carDoors', value: car.doors || 'Unknown' },
            { id: 'carBodyColour', value: car.bodyColour || car.exteriorColor || 'Unknown' },
            { id: 'carLocation', value: car.location || 'Unknown' }
        ];

        // Handle car type specific fields
        if (car.auctionDetails) {
            // Hard Mode (Bonhams) - Hide Easy mode fields, show Hard mode fields
            document.getElementById('ownersRow').style.display = 'none';
            document.getElementById('bodyTypeRow').style.display = 'none';
            document.getElementById('doorsRow').style.display = 'none';
            
            // Show sale date if available
            const saleDateRow = document.getElementById('saleDateRow');
            const saleDateElement = document.getElementById('carSaleDate');
            if (car.saleDate) {
                saleDateElement.textContent = car.saleDate;
                saleDateRow.style.display = 'flex';
            } else {
                saleDateRow.style.display = 'none';
            }

            // Show location (always visible for both modes)
            const locationRow = document.getElementById('locationRow');
            const locationElement = document.getElementById('carLocation');
            if (car.location) {
                locationElement.textContent = car.location;
                locationRow.style.display = 'flex';
            } else {
                locationRow.style.display = 'none';
            }

            // Show interior color if available
            const interiorColorRow = document.getElementById('interiorColorRow');
            const interiorColorElement = document.getElementById('carInteriorColor');
            if (car.interiorColor) {
                interiorColorElement.textContent = car.interiorColor;
                interiorColorRow.style.display = 'flex';
            } else {
                interiorColorRow.style.display = 'none';
            }

            // Show steering if available
            const steeringRow = document.getElementById('steeringRow');
            const steeringElement = document.getElementById('carSteering');
            if (car.steering) {
                steeringElement.textContent = car.steering;
                steeringRow.style.display = 'flex';
            } else {
                steeringRow.style.display = 'none';
            }

            // Show auction details section
            const auctionDetailsSection = document.getElementById('auctionDetailsSection');
            auctionDetailsSection.style.display = 'block';

            // Display key facts
            const keyFactsSection = document.getElementById('keyFactsSection');
            const keyFactsList = document.getElementById('keyFactsList');
            if (car.keyFacts && car.keyFacts.length > 0) {
                keyFactsList.innerHTML = '';
                car.keyFacts.forEach(fact => {
                    const li = document.createElement('li');
                    li.textContent = fact;
                    keyFactsList.appendChild(li);
                });
                keyFactsSection.style.display = 'block';
            } else {
                keyFactsSection.style.display = 'none';
            }
        } else {
            // Easy Mode (Lookers) - Show Easy mode fields, hide Hard mode fields
            // Show Easy mode specific fields if data is available
            const ownersRow = document.getElementById('ownersRow');
            if (car.owners && car.owners !== 'Unknown') {
                ownersRow.style.display = 'flex';
            } else {
                ownersRow.style.display = 'none';
            }
            
            const bodyTypeRow = document.getElementById('bodyTypeRow');
            if (car.bodyType && car.bodyType !== 'Unknown') {
                bodyTypeRow.style.display = 'flex';
            } else {
                bodyTypeRow.style.display = 'none';
            }
            
            const doorsRow = document.getElementById('doorsRow');
            if (car.doors && car.doors !== 'Unknown') {
                doorsRow.style.display = 'flex';
            } else {
                doorsRow.style.display = 'none';
            }
            
            // Show location if available
            const locationRow = document.getElementById('locationRow');
            if (car.location && car.location !== 'Unknown') {
                locationRow.style.display = 'flex';
            } else {
                locationRow.style.display = 'none';
            }
            
            // Hide Hard mode only sections
            document.getElementById('saleDateRow').style.display = 'none';
            document.getElementById('interiorColorRow').style.display = 'none';
            document.getElementById('steeringRow').style.display = 'none';
            document.getElementById('auctionDetailsSection').style.display = 'none';
        }
        
        details.forEach((detail, index) => {
            const element = document.getElementById(detail.id);
            element.style.opacity = '0';
            setTimeout(() => {
                element.textContent = detail.value;
                element.style.transition = 'opacity 0.3s ease';
                element.style.opacity = '1';
            }, 50 * index);
        });
        
        // Animate car display back in
        carDisplay.style.transition = 'all 0.5s ease';
        carDisplay.style.opacity = '1';
        carDisplay.style.transform = 'translateY(0)';
    }, 100);
}

// Handle image loading errors
function handleImageError(imgElement, originalUrl) {
    console.log('Image failed to load:', originalUrl);
    
    // Check if this is Easy mode (current game exists and difficulty is easy)
    if (currentGame && currentGame.difficulty === 'easy') {
        // Show Europe warning if not already shown
        const warning = document.getElementById('europeWarning');
        if (warning && warning.style.display === 'none') {
            warning.style.display = 'flex';
        }
    }
    
    // Set placeholder image
    imgElement.src = 'https://www.travelodge.co.uk/nw/assets/img/photo/image-unavailable.png';
    imgElement.style.filter = 'grayscale(1)';
}

// Set up image gallery with multiple photos
function setupImageGallery(images) {
    const mainImage = document.getElementById('mainCarImage');
    const thumbnailStrip = document.getElementById('thumbnailStrip');
    
    // Clear existing thumbnails
    thumbnailStrip.innerHTML = '';
    
    if (images.length === 0) {
        mainImage.src = 'https://www.travelodge.co.uk/nw/assets/img/photo/image-unavailable.png';
        return;
    }
    
    // Set main image with fade effect
    mainImage.style.opacity = '0';
    setTimeout(() => {
        mainImage.src = images[0];
        mainImage.onerror = () => handleImageError(mainImage, images[0]);
        mainImage.style.opacity = '1';
    }, 200);
    
    // Create thumbnails if more than one image
    if (images.length > 1) {
        images.forEach((imageUrl, index) => {
            const thumbnail = document.createElement('img');
            thumbnail.src = imageUrl;
            thumbnail.onerror = () => handleImageError(thumbnail, imageUrl);
            thumbnail.className = 'thumbnail' + (index === 0 ? ' active' : '');
            thumbnail.onclick = () => switchMainImage(imageUrl, thumbnail);
            
            // Add staggered animation
            thumbnail.style.opacity = '0';
            thumbnail.style.transform = 'translateY(20px)';
            thumbnailStrip.appendChild(thumbnail);
            
            setTimeout(() => {
                thumbnail.style.transition = 'all 0.3s ease';
                thumbnail.style.opacity = '';
                thumbnail.style.transform = '';
            }, 50 * index);
        });
        
        // Show thumbnail strip
        thumbnailStrip.style.display = 'flex';
    } else {
        // Hide thumbnail strip if only one image
        thumbnailStrip.style.display = 'none';
    }
}

// Switch main image when thumbnail clicked
function switchMainImage(imageUrl, clickedThumbnail) {
    const mainImage = document.getElementById('mainCarImage');
    
    // Fade out, change, fade in
    mainImage.style.opacity = '0';
    setTimeout(() => {
        mainImage.src = imageUrl;
        mainImage.onerror = () => handleImageError(mainImage, imageUrl);
        mainImage.style.opacity = '1';
    }, 200);
    
    // Update active thumbnail with smooth transition
    document.querySelectorAll('.thumbnail').forEach(thumb => {
        thumb.classList.remove('active');
        thumb.style.transform = '';
    });
    clickedThumbnail.classList.add('active');
    clickedThumbnail.style.transform = 'scale(1.1)';
}

// Convert price to slider position (0-100)
function priceToSlider(price) {
    if (price <= 100000) {
        // First half: £0-£100k maps to 0-50
        return (price / 100000) * 50;
    } else {
        // Second half: £100k-£500k maps to 50-100
        const priceAbove100k = Math.min(price - 100000, 400000);
        return 50 + (priceAbove100k / 400000) * 50;
    }
}

// Convert slider position (0-100) to price
function sliderToPrice(sliderValue) {
    if (sliderValue <= 50) {
        // First half: 0-50 maps to £0-£100k
        return (sliderValue / 50) * 100000;
    } else {
        // Second half: 50-100 maps to £100k-£500k
        const extraValue = (sliderValue - 50) / 50;
        return 100000 + (extraValue * 400000);
    }
}

// Sync price input and slider with beautiful overlay formatting
function syncInputToSlider() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    const overlay = document.getElementById('priceOverlay');
    
    let value = input.value;
    if (!isNaN(value) && value !== '') {
        const numValue = parseInt(value);
        
        // Allow higher values in number input, but clamp slider to £500k max
        const sliderValue = Math.min(Math.max(numValue, 0), 500000);
        
        // Update slider using non-linear mapping
        slider.value = priceToSlider(sliderValue);
        
        // Show formatted price in overlay
        overlay.textContent = numValue.toLocaleString();
        overlay.style.color = '#fff';
        overlay.style.fontStyle = 'normal';
    } else {
        overlay.textContent = 'Enter price';
        overlay.style.color = 'rgba(255, 255, 255, 0.5)';
        overlay.style.fontStyle = 'italic';
    }
}

function syncSliderToInput() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    const overlay = document.getElementById('priceOverlay');
    
    const sliderValue = parseFloat(slider.value);
    const price = Math.round(sliderToPrice(sliderValue));
    input.value = price;
    
    // Show formatted price in overlay
    overlay.textContent = price.toLocaleString();
    overlay.style.color = '#fff';
    overlay.style.fontStyle = 'normal';
}


// Submit guess
async function submitGuess() {
    const guessInput = document.getElementById('priceGuess').value; // Number input already gives us clean number
    const guessValue = parseInt(guessInput);
    const submitButton = document.querySelector('.submit-button');
    
    if (!guessValue || guessValue <= 0) {
        // Shake the input instead of alert
        const input = document.getElementById('priceGuess');
        input.style.animation = 'shake 0.5s';
        setTimeout(() => input.style.animation = '', 500);
        return;
    }
    
    // Handle challenge mode differently
    if (currentGame.mode === 'challenge') {
        await submitChallengeGuess(guessValue);
        return;
    }
    
    // Add loading state
    submitButton.innerHTML = '<span class="loading"></span>';
    submitButton.disabled = true;
    
    try {
        const response = await fetch('/api/check-guess', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': currentGame.sessionId
            },
            body: JSON.stringify({
                listingId: currentGame.currentListing.id,
                guessedPrice: guessValue,
                gameMode: currentGame.mode,
                difficulty: currentGame.difficulty
            })
        });
        
        if (!response.ok) throw new Error('Failed to submit guess');
        
        const result = await response.json();
        displayResult(result);
        
    } catch (error) {
        console.error('Error submitting guess:', error);
        alert('Failed to submit guess. Please try again.');
    } finally {
        // Reset button
        submitButton.innerHTML = 'Submit Guess';
        submitButton.disabled = false;
    }
}

// Display result
function displayResult(result) {
    // Update score
    currentGame.score = result.score;
    document.getElementById('scoreValue').textContent = 
        currentGame.mode === 'zero' ? result.score.toLocaleString() : result.score;
    
    // Set result modal content
    document.getElementById('resultTitle').textContent = result.correct ? 'Good Guess!' : 'Game Over!';
    document.getElementById('actualPrice').textContent = '£' + result.actualPrice.toLocaleString();
    document.getElementById('yourGuess').textContent = '£' + result.guessedPrice.toLocaleString();
    document.getElementById('priceDifference').textContent = '£' + result.difference.toLocaleString();
    document.getElementById('accuracy').textContent = (100 - result.percentage).toFixed(1) + '% accurate';
    document.getElementById('resultMessage').textContent = result.message;
    
    // Show original listing link if available
    const originalLinkDiv = document.getElementById('originalLink');
    if (result.originalUrl) {
        originalLinkDiv.style.display = 'block';
        let linkText = 'View Original Listing';
        if (result.originalUrl.includes('bonhams')) {
            linkText = 'View Original Auction Listing on Bonhams';
        } else if (result.originalUrl.includes('lookers')) {
            linkText = 'View Original Listing on Lookers';
        }
        originalLinkDiv.innerHTML = `<a href="${result.originalUrl}" target="_blank" class="original-link">${linkText}</a>`;
    } else {
        originalLinkDiv.style.display = 'none';
    }
    
    // Show appropriate modal
    if (result.gameOver) {
        document.getElementById('finalScore').textContent = 
            currentGame.mode === 'zero' 
                ? `Total Difference: £${result.score.toLocaleString()}`
                : `Final Streak: ${result.score}`;
        
        // Hide submit button if streak score is 0
        const submitButton = document.getElementById('submitStreakScore');
        if (currentGame.mode === 'streak' && result.score === 0) {
            submitButton.style.display = 'none';
        } else {
            submitButton.style.display = '';
        }
        
        lockBodyScroll();
        document.getElementById('gameOverModal').style.display = 'flex';
    } else {
        lockBodyScroll();
        document.getElementById('resultModal').style.display = 'flex';
    }
}

// Next round
function nextRound() {
    document.getElementById('resultModal').style.display = 'none';
    unlockBodyScroll();
    
    if (currentGame.mode === 'challenge') {
        // Check if challenge is complete (10 cars)
        const currentCarNum = currentGame.challengeSession ? currentGame.challengeSession.currentCar : 0;
        const totalGuesses = currentGame.challengeGuesses ? currentGame.challengeGuesses.length : 0;
        
        console.log('Next round analysis:', {
            currentCarNum,
            totalGuesses,
            sessionComplete: currentGame.challengeSession ? currentGame.challengeSession.isComplete : 'unknown',
            sessionData: currentGame.challengeSession
        });
        
        // Only consider complete if backend explicitly says so
        const isComplete = (currentGame.challengeSession && currentGame.challengeSession.isComplete === true);
        
        if (isComplete) {
            // Challenge complete - show final results
            console.log('Challenge complete! Showing results...');
            displayChallengeResults();
        } else {
            // Load next challenge car
            console.log('Loading next challenge car...');
            loadChallengeAuto();
            updateChallengeProgress();
            // Auto-scroll to car image after loading next car
            scrollToCarImage();
        }
    } else {
        // Regular game modes
        loadNextCar();
        // Auto-scroll to car image after loading next car
        scrollToCarImage();
    }
}

// Auto-scroll to car image (especially useful on mobile)
function scrollToCarImage() {
    // Add a small delay to ensure the new car has loaded
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
}

// End game
function endGame() {
    location.reload();
}

// Close modals when clicking outside
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        // Special handling for leaderboard modal to ensure proper redirect
        if (event.target.id === 'leaderboardModal') {
            closeLeaderboard();
        } else {
            event.target.style.display = 'none';
        }
    }
}

// Data source info removed - functionality preserved in backend for admin use

// Challenge Mode Functions
async function startChallengeMode() {
    try {
        // Include difficulty parameter in the request
        const difficultyParam = currentGame.difficulty ? `?difficulty=${currentGame.difficulty}` : '';
        const response = await fetch(`/api/challenge/start${difficultyParam}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) throw new Error('Failed to start challenge');
        
        const challengeSession = await response.json();
        currentGame.challengeSession = challengeSession;
        currentGame.sessionId = challengeSession.sessionId; // Use backend session ID
        currentGame.challengeGuesses = [];
        
        // Load first car
        loadChallengeAuto();
        
        // Add challenge progress display
        addChallengeProgress();
        
    } catch (error) {
        console.error('Error starting challenge:', error);
        alert('Failed to start challenge mode. Please try again.');
        location.reload();
    }
}

// Initialize challenge with existing session (for friend challenges)
async function initializeChallenge(sessionId) {
    try {
        const response = await fetch(`/api/challenge/${sessionId}`);
        
        if (!response.ok) throw new Error('Failed to load challenge session');
        
        const challengeSession = await response.json();
        
        // Set up complete game state - IMPORTANT: Use the session ID from the backend
        currentGame.mode = 'challenge';
        currentGame.difficulty = challengeSession.difficulty;
        currentGame.score = challengeSession.totalScore || 0;
        currentGame.sessionId = challengeSession.sessionId; // Use backend session ID
        currentGame.challengeSession = challengeSession;
        currentGame.challengeGuesses = challengeSession.guesses || [];
        
        // Update UI elements
        document.getElementById('scoreDisplay').style.display = 'inline-block';
        const scoreLabel = document.getElementById('scoreLabel');
        scoreLabel.textContent = 'Challenge Score: ';
        document.getElementById('scoreValue').textContent = currentGame.score;
        
        // Load current car
        loadChallengeAuto();
        
        // Add challenge progress display
        addChallengeProgress();
        
    } catch (error) {
        console.error('Error initializing challenge:', error);
        alert('Failed to load challenge. Please try again.');
        location.reload();
    }
}

function addChallengeProgress() {
    // Add challenge progress bar after car info
    const carInfo = document.querySelector('.car-info');
    const progressDiv = document.createElement('div');
    progressDiv.className = 'challenge-progress';
    progressDiv.id = 'challengeProgress';
    
    const session = currentGame.challengeSession;
    const currentCarNum = session.currentCar + 1;
    
    progressDiv.innerHTML = `
        <h4>Challenge Progress</h4>
        <p>Car ${currentCarNum} of 10</p>
        <div class="challenge-progress-bar">
            <div class="challenge-progress-fill" style="width: ${(currentCarNum - 1) / 10 * 100}%"></div>
        </div>
        <p>Current Score: <span id="challengeCurrentScore">${session.totalScore}</span> points</p>
    `;
    
    carInfo.appendChild(progressDiv);
}

function updateChallengeProgress() {
    const session = currentGame.challengeSession;
    const currentCarNum = session.currentCar + 1;
    const progressDiv = document.getElementById('challengeProgress');
    
    if (progressDiv) {
        progressDiv.innerHTML = `
            <h4>Challenge Progress</h4>
            <p>Car ${currentCarNum} of 10</p>
            <div class="challenge-progress-bar">
                <div class="challenge-progress-fill" style="width: ${(currentCarNum - 1) / 10 * 100}%"></div>
            </div>
            <p>Current Score: <span id="challengeCurrentScore">${session.totalScore}</span> points</p>
        `;
    }
}

async function loadChallengeAuto() {
    let session = currentGame.challengeSession;
    
    // If session is missing or incomplete, fetch it from backend
    if (!session || !session.cars || session.cars.length === 0) {
        console.log('Challenge session missing or incomplete, fetching from backend...');
        
        let sessionId = currentGame.sessionId;
        if (!sessionId && session) {
            sessionId = session.sessionId;
        }
        
        if (!sessionId) {
            console.error('No session ID available to fetch challenge data');
            alert('Challenge session error. Please refresh and try again.');
            return;
        }
        
        try {
            const response = await fetch(`/api/challenge/${sessionId}`);
            if (!response.ok) {
                throw new Error('Failed to fetch challenge session');
            }
            
            const fetchedSession = await response.json();
            currentGame.challengeSession = fetchedSession;
            session = fetchedSession;
            console.log('Fetched challenge session:', session);
        } catch (error) {
            console.error('Error fetching challenge session:', error);
            alert('Failed to load challenge data. Please refresh and try again.');
            return;
        }
    }
    
    if (session.currentCar >= session.cars.length) {
        console.log('All cars completed');
        return;
    }
    
    const currentCar = session.cars[session.currentCar];
    currentGame.currentListing = currentCar;
    displayCar(currentCar);
    
    // Reset guess inputs
    const priceInput = document.getElementById('priceGuess');
    const priceOverlay = document.getElementById('priceOverlay');
    
    priceInput.value = '';
    document.getElementById('priceSlider').value = 25; // Set to £50k (middle of first range)
    
    // Set placeholder styling
    priceOverlay.textContent = 'Enter price';
    priceOverlay.style.color = 'rgba(255, 255, 255, 0.5)';
    priceOverlay.style.fontStyle = 'italic';
}

async function submitChallengeGuess(guessValue) {
    const submitButton = document.querySelector('.submit-button');
    submitButton.innerHTML = '<span class="loading"></span>';
    submitButton.disabled = true;
    
    try {
        // Initialize currentGame.challengeSession at the very beginning if needed
        if (!currentGame.challengeSession) {
            currentGame.challengeSession = {
                sessionId: null,
                currentCar: 0,
                totalScore: 0,
                guesses: []
            };
        }
        
        // Get session ID with fallback logic
        let sessionId = currentGame.sessionId;
        if (!sessionId && currentGame.challengeSession) {
            sessionId = currentGame.challengeSession.sessionId;
        }
        
        if (!sessionId) {
            console.error('No session ID available for challenge guess');
            throw new Error('Session not properly initialized');
        }
        
        console.log('Submitting challenge guess with session ID:', sessionId);
        
        const response = await fetch(`/api/challenge/${sessionId}/guess`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                guessedPrice: guessValue
            })
        });
        
        console.log('Challenge guess response status:', response.status);
        
        if (!response.ok) {
            const errorText = await response.text();
            console.error('Challenge guess failed:', errorText);
            throw new Error(`Server error: ${response.status}`);
        }
        
        const result = await response.json();
        console.log('Challenge guess result:', result);
        console.log('currentGame.challengeSession before initialization:', currentGame.challengeSession);
        
        // Initialize challenge session BEFORE accessing it
        if (!currentGame.challengeSession) {
            currentGame.challengeSession = {
                sessionId: sessionId,
                currentCar: 0,
                totalScore: 0,
                guesses: [],
                isComplete: false
            };
        }
        
        // Ensure all required properties exist
        if (!currentGame.challengeSession.guesses) {
            currentGame.challengeSession.guesses = [];
        }
        if (typeof currentGame.challengeSession.currentCar !== 'number') {
            currentGame.challengeSession.currentCar = 0;
        }
        if (typeof currentGame.challengeSession.totalScore !== 'number') {
            currentGame.challengeSession.totalScore = 0;
        }
        
        // Initialize challengeGuesses array if it doesn't exist
        if (!currentGame.challengeGuesses) {
            currentGame.challengeGuesses = [];
        }
        
        currentGame.challengeGuesses.push(result);
        
        // Update challenge session safely
        currentGame.challengeSession.currentCar++;
        currentGame.challengeSession.totalScore = result.totalScore;
        currentGame.challengeSession.guesses.push(result);
        
        // CRITICAL: Update completion status ONLY when backend confirms completion
        if (result.sessionComplete === true) {
            currentGame.challengeSession.isComplete = true;
        }
        if (result.isLastCar === true) {
            currentGame.challengeSession.isComplete = true;
        }
        
        console.log('Updated challenge session:', {
            currentCar: currentGame.challengeSession.currentCar,
            totalScore: currentGame.challengeSession.totalScore,
            isComplete: currentGame.challengeSession.isComplete,
            sessionComplete: result.sessionComplete,
            isLastCar: result.isLastCar
        });
        
        // Update score display
        document.getElementById('scoreValue').textContent = result.totalScore;
        
        // Always show the individual result first
        displayChallengeResult(result);
        
    } catch (error) {
        console.error('Challenge guess submission error:', error);
        alert(`Failed to submit guess: ${error.message}`);
    } finally {
        submitButton.innerHTML = 'Submit Guess';
        submitButton.disabled = false;
    }
}

function displayChallengeResult(result) {
    // Update result modal content for challenge mode
    const currentCar = currentGame.challengeSession ? currentGame.challengeSession.currentCar : result.nextCarNumber || 1;
    document.getElementById('resultTitle').textContent = `Car ${currentCar}/10 Complete!`;
    document.getElementById('actualPrice').textContent = '£' + result.actualPrice.toLocaleString();
    document.getElementById('yourGuess').textContent = '£' + result.guessedPrice.toLocaleString();
    document.getElementById('priceDifference').textContent = '£' + result.difference.toLocaleString();
    document.getElementById('accuracy').textContent = (100 - result.percentage).toFixed(1) + '% accurate';
    document.getElementById('resultMessage').innerHTML = `
        <strong>${result.points} points!</strong><br>
        ${result.message}
    `;
    
    // Show original listing link
    const originalLinkDiv = document.getElementById('originalLink');
    if (result.originalUrl) {
        originalLinkDiv.style.display = 'block';
        let linkText = 'View Original Listing';
        if (result.originalUrl.includes('bonhams')) {
            linkText = 'View Original Auction Listing on Bonhams';
        } else if (result.originalUrl.includes('lookers')) {
            linkText = 'View Original Listing on Lookers';
        }
        originalLinkDiv.innerHTML = `<a href="${result.originalUrl}" target="_blank" class="original-link">${linkText}</a>`;
    } else {
        originalLinkDiv.style.display = 'none';
    }
    
    // Show modal with modified buttons
    const nextButton = document.querySelector('.next-button');
    
    // Only use backend-provided completion flags
    const isLastCar = result.isLastCar === true || result.sessionComplete === true;
                     
    nextButton.textContent = isLastCar ? 'View Results' : 'Next Car';
    
    console.log('Challenge result analysis:', {
        isLastCar: result.isLastCar,
        sessionComplete: result.sessionComplete,
        currentCar: currentGame.challengeSession ? currentGame.challengeSession.currentCar : 'unknown',
        nextCarNumber: result.nextCarNumber,
        finalDecision: isLastCar
    });
    
    lockBodyScroll();
    document.getElementById('resultModal').style.display = 'flex';
}

function displayChallengeResults() {
    const session = currentGame.challengeSession;
    
    if (!session) {
        console.error('No challenge session available for results display');
        alert('Error: Challenge session not found. Please try again.');
        location.reload();
        return;
    }
    
    console.log('Displaying challenge results for session:', session);
    
    // Set final score
    const finalScore = session.totalScore || 0;
    document.getElementById('challengeFinalScore').textContent = `Final Score: ${finalScore.toLocaleString()} points`;
    
    // Build results breakdown
    const resultsDiv = document.getElementById('challengeResults');
    resultsDiv.innerHTML = '';
    
    const guesses = session.guesses || currentGame.challengeGuesses || [];
    const cars = session.cars || [];
    
    console.log('Results breakdown:', { guessesCount: guesses.length, carsCount: cars.length });
    
    if (guesses.length === 0) {
        resultsDiv.innerHTML = '<div class="no-results">No guesses recorded</div>';
    } else {
        guesses.forEach((guess, index) => {
            const car = cars[index];
            if (!car) {
                console.warn(`No car data for guess ${index + 1}`);
                return;
            }
            
            const resultItem = document.createElement('div');
            resultItem.className = 'challenge-result-item';
            
            const accuracy = (100 - guess.percentage).toFixed(1);
            
            resultItem.innerHTML = `
                <div class="challenge-result-car">${car.year || ''} ${car.make || ''} ${car.model || 'Unknown Car'}</div>
                <div class="challenge-result-accuracy">${accuracy}% accurate</div>
                <div class="challenge-result-points">${guess.points || 0} pts</div>
            `;
            
            resultsDiv.appendChild(resultItem);
        });
    }
    
    // Hide submit button if score is 0
    const submitButton = document.querySelector('#challengeCompleteModal .submit-score-button');
    if (session.totalScore === 0) {
        submitButton.style.display = 'none';
    } else {
        submitButton.style.display = '';
    }
    
    // Show challenge complete modal
    document.getElementById('challengeCompleteModal').style.display = 'flex';
}

// Initialize the app
document.addEventListener('DOMContentLoaded', () => {
    
    // Load saved difficulty preference and apply it
    const savedDifficulty = loadDifficultyPreference();
    currentGame.difficulty = savedDifficulty;
    leaderboardState.currentDifficulty = savedDifficulty;
    selectDifficulty(savedDifficulty); // This will update the UI to match
    
    // Update leaderboard difficulty tabs to match preference
    document.getElementById('hardDifficultyTab').classList.remove('active');
    document.getElementById('easyDifficultyTab').classList.remove('active');
    document.getElementById(savedDifficulty + 'DifficultyTab').classList.add('active');
    
    // Add shake animation to CSS
    const style = document.createElement('style');
    style.innerHTML = `
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            10%, 30%, 50%, 70%, 90% { transform: translateX(-10px); }
            20%, 40%, 60%, 80% { transform: translateX(10px); }
        }
        
        .main-car-image { transition: opacity 0.3s ease; }
        .detail-value { transition: opacity 0.3s ease; }
    `;
    document.head.appendChild(style);
    
    // Allow Enter key to submit guess and set up slider sync
    const priceGuess = document.getElementById('priceGuess');
    const priceSlider = document.getElementById('priceSlider');
    
    priceGuess.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            submitGuess();
        }
    });
    
    // Set up price input and slider synchronization
    priceGuess.addEventListener('input', syncInputToSlider);
    priceSlider.addEventListener('input', syncSliderToInput);
    
    // Allow Enter key to submit name in leaderboard modal
    const playerNameInput = document.getElementById('playerName');
    if (playerNameInput) {
        playerNameInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                submitToLeaderboard();
            }
        });
    }
    
    // Add smooth scrolling
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({ behavior: 'smooth' });
            }
        });
    });
});

// Leaderboard Functions

// Show name input modal for leaderboard submission
function showNameInputModal(gameMode) {
    // Check if user is logged in via auth.js
    const currentUser = window.authFunctions && window.authFunctions.currentUser();
    
    // Store the leaderboard data
    let score = 0;
    let sessionId = null;
    
    if (gameMode === 'challenge') {
        if (currentGame.challengeSession) {
            score = currentGame.challengeSession.totalScore;
            sessionId = currentGame.challengeSession.sessionId;
        }
    } else if (gameMode === 'streak') {
        score = currentGame.score;
        sessionId = currentGame.sessionId;
    }
    
    currentGame.pendingLeaderboardData = {
        gameMode: gameMode,
        score: score,
        sessionId: sessionId
    };
    
    // If user is logged in, auto-submit to leaderboard with their display name
    if (currentUser) {
        const playerName = currentUser.displayName || currentUser.username;
        console.log('Auto-submitting leaderboard for logged-in user:', playerName);
        autoSubmitForLoggedInUser(playerName);
        return;
    }
    
    // Hide current modal
    document.getElementById('challengeCompleteModal').style.display = 'none';
    document.getElementById('gameOverModal').style.display = 'none';
    
    // Show name input modal (keep body locked since we're showing another modal)
    document.getElementById('nameInputModal').style.display = 'flex';
    
    // Focus on the name input
    setTimeout(() => {
        document.getElementById('playerName').focus();
    }, 100);
}

// Auto-submit leaderboard for logged-in users
async function autoSubmitForLoggedInUser(playerName) {
    if (!currentGame.pendingLeaderboardData) {
        console.error('No score data available for auto-submission');
        return;
    }
    
    try {
        const submissionData = {
            name: playerName,
            score: parseInt(currentGame.pendingLeaderboardData.score) || 0,
            gameMode: currentGame.pendingLeaderboardData.gameMode,
            difficulty: currentGame.difficulty || 'hard',
            sessionId: currentGame.pendingLeaderboardData.sessionId || ''
        };
        
        const response = await fetch('/api/leaderboard/submit', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(submissionData)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to submit score');
        }
        
        const result = await response.json();
        console.log('Score submitted successfully for logged-in user:', result);
        
        // Hide current modals
        document.getElementById('challengeCompleteModal').style.display = 'none';
        document.getElementById('gameOverModal').style.display = 'none';
        
        // Mark that we're showing leaderboard after submission so it redirects on close
        currentGame.leaderboardShownAfterSubmission = true;
        console.log('Setting leaderboardShownAfterSubmission to true for auto-submit');
        
        // Show leaderboard with the appropriate game mode
        const gameMode = currentGame.pendingLeaderboardData.gameMode;
        console.log('Showing leaderboard with game mode:', gameMode);
        showLeaderboard(gameMode);
        
    } catch (error) {
        console.error('Failed to auto-submit score:', error);
        // Fallback to manual submission
        document.getElementById('challengeCompleteModal').style.display = 'none';
        document.getElementById('gameOverModal').style.display = 'none';
        document.getElementById('nameInputModal').style.display = 'flex';
        
        // Pre-populate the name field
        document.getElementById('playerName').value = playerName;
        setTimeout(() => {
            document.getElementById('playerName').focus();
        }, 100);
    }
}

// Show name input modal from game over (streak mode)
function showNameInputModalFromGameOver() {
    showNameInputModal('streak');
}

// Submit score to leaderboard
async function submitToLeaderboard() {
    const nameInput = document.getElementById('playerName');
    const name = nameInput.value.trim();
    
    if (!name) {
        nameInput.style.animation = 'shake 0.5s';
        setTimeout(() => nameInput.style.animation = '', 500);
        return;
    }
    
    if (name.length > 20) {
        alert('Name must be 20 characters or less');
        return;
    }
    
    if (!currentGame.pendingLeaderboardData) {
        alert('No score data available');
        return;
    }
    
    try {
        const submissionData = {
            name: name,
            score: parseInt(currentGame.pendingLeaderboardData.score) || 0, // Ensure score is an integer
            gameMode: currentGame.pendingLeaderboardData.gameMode,
            difficulty: currentGame.difficulty || 'hard', // Include current difficulty
            sessionId: currentGame.pendingLeaderboardData.sessionId || ''
        };
        
        const response = await fetch('/api/leaderboard/submit', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(submissionData)
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to submit score');
        }
        
        const result = await response.json();
        
        // Hide name input modal
        document.getElementById('nameInputModal').style.display = 'none';
        
        // Show success message and leaderboard
        showLeaderboardWithHighlight(result.entry, result.position);
        
    } catch (error) {
        console.error('Error submitting score:', error);
        alert('Failed to submit score: ' + error.message);
    }
}

// Skip leaderboard submission
function skipLeaderboard() {
    document.getElementById('nameInputModal').style.display = 'none';
    currentGame.pendingLeaderboardData = null;
    // Go back to main menu
    location.reload();
}

// Helper function to lock body scroll
function lockBodyScroll() {
    document.body.style.overflow = 'hidden';
    document.body.style.position = 'fixed';
    document.body.style.width = '100%';
}

// Helper function to unlock body scroll
function unlockBodyScroll() {
    document.body.style.overflow = '';
    document.body.style.position = '';
    document.body.style.width = '';
}

// Open leaderboard modal
function openLeaderboard() {
    // Reset flag since this is manual access, not after score submission
    currentGame.leaderboardShownAfterSubmission = false;
    
    // Use current difficulty preference
    leaderboardState.currentDifficulty = currentGame.difficulty;
    
    // Prevent body scroll on mobile
    lockBodyScroll();
    
    document.getElementById('leaderboardModal').style.display = 'flex';
    showLeaderboard('challenge'); // Default to challenge mode
}

// Close leaderboard modal
function closeLeaderboard() {
    document.getElementById('leaderboardModal').style.display = 'none';
    
    // Restore body scroll
    unlockBodyScroll();
    
    // If leaderboard was shown after score submission, reload page to go back to homepage
    console.log('Closing leaderboard, leaderboardShownAfterSubmission:', currentGame.leaderboardShownAfterSubmission);
    if (currentGame.leaderboardShownAfterSubmission) {
        console.log('Reloading page to go back to homepage');
        location.reload();
    }
}

// Show leaderboard for specific game mode
async function showLeaderboard(gameMode) {
    // Update game mode state and tab states
    leaderboardState.currentGameMode = gameMode;
    document.getElementById('challengeTab').classList.remove('active');
    document.getElementById('streakTab').classList.remove('active');
    document.getElementById(gameMode + 'Tab').classList.add('active');
    
    // Update difficulty tab states based on current preference
    document.getElementById('hardDifficultyTab').classList.remove('active');
    document.getElementById('easyDifficultyTab').classList.remove('active');
    document.getElementById(leaderboardState.currentDifficulty + 'DifficultyTab').classList.add('active');
    
    // Load leaderboard with current difficulty
    await loadLeaderboardData();
}

// Toggle difficulty filter for leaderboard
async function toggleDifficultyFilter(difficulty) {
    // Update difficulty state and tab states
    leaderboardState.currentDifficulty = difficulty;
    document.getElementById('hardDifficultyTab').classList.remove('active');
    document.getElementById('easyDifficultyTab').classList.remove('active');
    document.getElementById(difficulty + 'DifficultyTab').classList.add('active');
    
    // Load leaderboard with new difficulty
    await loadLeaderboardData();
}

// Load leaderboard data with current filters
async function loadLeaderboardData() {
    // Show loading
    const contentArea = document.getElementById('leaderboardContent');
    contentArea.innerHTML = '<div class="leaderboard-empty">Loading...</div>';
    
    try {
        const url = `/api/leaderboard?mode=${leaderboardState.currentGameMode}&difficulty=${leaderboardState.currentDifficulty}&limit=10`;
        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to load leaderboard');
        
        const entries = await response.json();
        displayLeaderboardEntries(entries, leaderboardState.currentGameMode, leaderboardState.currentDifficulty);
        
    } catch (error) {
        console.error('Error loading leaderboard:', error);
        contentArea.innerHTML = '<div class="leaderboard-empty">Failed to load leaderboard</div>';
    }
}

// Display leaderboard entries
function displayLeaderboardEntries(entries, gameMode, difficulty) {
    const contentArea = document.getElementById('leaderboardContent');
    
    if (entries.length === 0) {
        const modeLabel = gameMode === 'challenge' ? 'Challenge' : 'Streak';
        const difficultyLabel = difficulty === 'easy' ? 'Easy' : 'Hard';
        contentArea.innerHTML = `
            <div class="leaderboard-empty">
                <p>No ${modeLabel} Mode scores yet for ${difficultyLabel} difficulty!</p>
                <p>Be the first to submit a score!</p>
            </div>
        `;
        return;
    }
    
    let html = '';
    entries.forEach((entry, index) => {
        const rank = index + 1;
        const isTop3 = rank <= 3;
        const scoreText = gameMode === 'challenge' ? `${entry.score.toLocaleString()} pts` : `${entry.score} streak`;
        const date = new Date(entry.date).toLocaleDateString();
        
        html += `
            <div class="leaderboard-entry ${isTop3 ? 'top-3' : ''}">
                <div class="leaderboard-rank ${isTop3 ? 'top-3' : ''}">${getRankDisplay(rank)}</div>
                <div class="leaderboard-name">${escapeHtml(entry.name)}</div>
                <div class="leaderboard-score">${scoreText}</div>
                <div class="leaderboard-date">${date}</div>
            </div>
        `;
    });
    
    contentArea.innerHTML = html;
}

// Show leaderboard with highlighted entry
async function showLeaderboardWithHighlight(entry, position) {
    // Mark that leaderboard is being shown after score submission
    currentGame.leaderboardShownAfterSubmission = true;
    
    document.getElementById('leaderboardModal').style.display = 'flex';
    
    // Show the appropriate tab
    showLeaderboard(entry.gameMode);
    
    // Wait a bit for the leaderboard to load, then highlight the entry
    setTimeout(() => {
        const entries = document.querySelectorAll('.leaderboard-entry');
        if (entries[position - 1]) {
            entries[position - 1].style.background = 'linear-gradient(135deg, rgba(32, 178, 170, 0.3) 0%, rgba(65, 105, 225, 0.3) 100%)';
            entries[position - 1].style.borderColor = '#20b2aa';
            entries[position - 1].scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    }, 500);
}

// Get rank display with emojis for top 3
function getRankDisplay(rank) {
    switch (rank) {
        case 1: return '🥇';
        case 2: return '🥈';
        case 3: return '🥉';
        default: return `#${rank}`;
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}