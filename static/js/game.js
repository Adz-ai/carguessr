// Game state
let currentGame = {
    mode: null,
    currentListing: null,
    score: 0,
    sessionId: generateSessionId(),
    challengeSession: null,
    challengeGuesses: []
};

// Generate a session ID for tracking scores
function generateSessionId() {
    return Math.random().toString(36).substr(2, 9);
}

// Start the game with selected mode
function startGame(mode) {
    currentGame.mode = mode;
    currentGame.score = 0;
    
    // Hide mode selection, show game area
    document.getElementById('modeSelection').style.display = 'none';
    document.getElementById('gameArea').style.display = 'block';
    document.getElementById('scoreDisplay').style.display = 'block';
    
    // Update score label based on mode
    const scoreLabel = document.getElementById('scoreLabel');
    if (mode === 'zero') {
        scoreLabel.textContent = 'Total Difference: £';
    } else if (mode === 'streak') {
        scoreLabel.textContent = 'Streak: ';
    } else if (mode === 'challenge') {
        scoreLabel.textContent = 'Score: ';
    }
    
    // Load data source info
    loadDataSourceInfo();
    
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
        // Try enhanced listing first, fallback to standard
        let response = await fetch('/api/random-enhanced-listing');
        if (!response.ok) {
            console.log('Enhanced listing not available, falling back to standard');
            response = await fetch('/api/random-listing');
            if (!response.ok) throw new Error('Failed to load listing');
        }
        
        const listing = await response.json();
        currentGame.currentListing = listing;
        displayCar(listing);
        
        // Reset guess inputs to be synchronized
        document.getElementById('priceGuess').value = '';
        document.getElementById('priceSlider').value = 25; // Set to £50k (middle of first range)
        
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
        
        // Set car title
        document.getElementById('carTitle').textContent = `${car.year || 'Unknown'} ${car.make || 'Unknown'} ${car.model || 'Unknown'}`;
        
        // Set car details with staggered animation
        const details = [
            { id: 'carYear', value: car.year || 'Unknown' },
            { id: 'carEngine', value: car.engine || 'Unknown' },
            { id: 'carMileage', value: car.mileageFormatted || (car.mileage ? car.mileage.toLocaleString() + ' miles' : 'Unknown') },
            { id: 'carFuelType', value: car.fuelType || 'Unknown' },
            { id: 'carGearbox', value: car.gearbox || 'Unknown' },
            { id: 'carBodyColour', value: car.bodyColour || car.exteriorColor || 'Unknown' }
        ];

        // Handle enhanced Bonhams characteristics
        if (car.auctionDetails) {
            // Show sale date if available
            const saleDateRow = document.getElementById('saleDateRow');
            const saleDateElement = document.getElementById('carSaleDate');
            if (car.saleDate) {
                saleDateElement.textContent = car.saleDate;
                saleDateRow.style.display = 'flex';
            } else {
                saleDateRow.style.display = 'none';
            }

            // Show location if available
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
            // Hide enhanced sections for standard cars
            document.getElementById('saleDateRow').style.display = 'none';
            document.getElementById('locationRow').style.display = 'none';
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

// Enhance image URL for better quality
function enhanceImageUrl(url) {
    // For imgix URLs, add quality and size parameters
    if (url.includes('imgix')) {
        const separator = url.includes('?') ? '&' : '?';
        return `${url}${separator}w=800&h=600&fit=crop&auto=format,enhance&q=85`;
    }
    
    // For amazonaws URLs, try to get larger versions
    if (url.includes('amazonaws')) {
        // Remove any existing size constraints and add high quality params
        let cleanUrl = url.replace(/\/\d+x\d+\//, '/').replace(/[?&]w=\d+/, '').replace(/[?&]h=\d+/, '');
        const separator = cleanUrl.includes('?') ? '&' : '?';
        return `${cleanUrl}${separator}w=800&h=600&q=85`;
    }
    
    // For other URLs, return as-is but try to remove small size constraints
    return url.replace(/\/thumb\//, '/').replace(/\/small\//, '/').replace(/\/preview\//, '/');
}

// Set up image gallery with multiple photos
function setupImageGallery(images) {
    const mainImage = document.getElementById('mainCarImage');
    const thumbnailStrip = document.getElementById('thumbnailStrip');
    
    // Clear existing thumbnails
    thumbnailStrip.innerHTML = '';
    
    if (images.length === 0) {
        mainImage.src = 'https://via.placeholder.com/600x400?text=No+Image';
        return;
    }
    
    // Enhance all image URLs for better quality
    const enhancedImages = images.map(url => enhanceImageUrl(url));
    
    // Preload all images to prevent blurry loading
    const imagePromises = enhancedImages.map(url => {
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = () => resolve({ url, img });
            img.onerror = () => reject(new Error(`Failed to load ${url}`));
            img.src = url;
        });
    });
    
    // Set main image with proper loading
    mainImage.style.opacity = '0';
    const firstImage = new Image();
    firstImage.onload = () => {
        mainImage.src = enhancedImages[0];
        mainImage.style.opacity = '1';
    };
    firstImage.onerror = () => {
        // Fallback to original URL if enhanced version fails
        const fallbackImage = new Image();
        fallbackImage.onload = () => {
            mainImage.src = images[0];
            mainImage.style.opacity = '1';
        };
        fallbackImage.onerror = () => {
            mainImage.src = 'https://via.placeholder.com/600x400?text=Image+Not+Found';
            mainImage.style.opacity = '1';
        };
        fallbackImage.src = images[0];
    };
    firstImage.src = enhancedImages[0];
    
    // Create thumbnails if more than one image
    if (images.length > 1) {
        images.forEach((imageUrl, index) => {
            const enhancedUrl = enhancedImages[index];
            const thumbnail = document.createElement('img');
            
            // Add loading placeholder
            thumbnail.style.backgroundColor = 'rgba(255, 255, 255, 0.1)';
            thumbnail.className = 'thumbnail' + (index === 0 ? ' active' : '');
            
            // Ensure image loads properly before showing
            thumbnail.onload = () => {
                thumbnail.style.backgroundColor = '';
                // Add staggered animation after image loads
                setTimeout(() => {
                    thumbnail.style.transition = 'all 0.3s ease';
                    thumbnail.style.opacity = index === 0 ? '1' : '';
                    thumbnail.style.transform = '';
                }, 50 * index);
            };
            
            thumbnail.onerror = () => {
                // Fallback to original URL if enhanced version fails
                const fallbackThumb = new Image();
                fallbackThumb.onload = () => {
                    thumbnail.src = imageUrl;
                    thumbnail.style.backgroundColor = '';
                };
                fallbackThumb.onerror = () => {
                    thumbnail.src = 'https://via.placeholder.com/90x70?text=Error';
                    thumbnail.style.backgroundColor = '';
                };
                fallbackThumb.src = imageUrl;
            };
            
            thumbnail.onclick = () => switchMainImage(enhancedUrl, thumbnail, imageUrl);
            
            // Set initial animation state
            thumbnail.style.opacity = '0';
            thumbnail.style.transform = 'translateY(20px)';
            
            // Set the enhanced source after setting up event handlers
            thumbnail.src = enhancedUrl;
            
            thumbnailStrip.appendChild(thumbnail);
        });
        
        // Show thumbnail strip
        thumbnailStrip.style.display = 'flex';
    } else {
        // Hide thumbnail strip if only one image
        thumbnailStrip.style.display = 'none';
    }
}

// Switch main image when thumbnail clicked
function switchMainImage(enhancedUrl, clickedThumbnail, originalUrl = null) {
    const mainImage = document.getElementById('mainCarImage');
    
    // Add loading state
    mainImage.style.opacity = '0.3';
    mainImage.style.filter = 'blur(2px)';
    
    // Preload the enhanced image to ensure it's crisp
    const newImage = new Image();
    newImage.onload = () => {
        // Enhanced image is fully loaded, now switch
        mainImage.src = enhancedUrl;
        mainImage.style.opacity = '1';
        mainImage.style.filter = 'none';
        mainImage.style.transition = 'all 0.3s ease';
    };
    
    newImage.onerror = () => {
        // Enhanced URL failed, try original URL if provided
        if (originalUrl && originalUrl !== enhancedUrl) {
            const fallbackImage = new Image();
            fallbackImage.onload = () => {
                mainImage.src = originalUrl;
                mainImage.style.opacity = '1';
                mainImage.style.filter = 'none';
                mainImage.style.transition = 'all 0.3s ease';
            };
            fallbackImage.onerror = () => {
                // Both URLs failed, show error placeholder
                mainImage.src = 'https://via.placeholder.com/600x400?text=Image+Error';
                mainImage.style.opacity = '1';
                mainImage.style.filter = 'none';
            };
            fallbackImage.src = originalUrl;
        } else {
            // No fallback available, show error
            mainImage.src = 'https://via.placeholder.com/600x400?text=Image+Error';
            mainImage.style.opacity = '1';
            mainImage.style.filter = 'none';
        }
    };
    
    // Start loading the enhanced image
    newImage.src = enhancedUrl;
    
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

// Sync price input and slider with improved formatting
function syncInputToSlider() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    
    let value = input.value.replace(/,/g, ''); // Remove existing commas
    if (!isNaN(value) && value !== '') {
        const numValue = parseInt(value);
        
        // Allow higher values in text input, but clamp slider to £500k max
        const sliderValue = Math.min(Math.max(numValue, 0), 500000);
        
        // Update slider using non-linear mapping
        slider.value = priceToSlider(sliderValue);
        
        // Format input with commas (but don't trigger another event)
        input.value = numValue.toLocaleString();
    }
}

function syncSliderToInput() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    
    const sliderValue = parseFloat(slider.value);
    const price = Math.round(sliderToPrice(sliderValue));
    input.value = price.toLocaleString();
}


// Submit guess
async function submitGuess() {
    const guessInput = document.getElementById('priceGuess').value.replace(/,/g, ''); // Remove commas
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
                gameMode: currentGame.mode
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
        if (result.originalUrl.includes('motors')) {
            linkText = 'View Original Listing on Motors.co.uk';
        } else if (result.originalUrl.includes('collectingcars')) {
            linkText = 'View Original Listing on Collecting Cars';
        } else if (result.originalUrl.includes('bonhams')) {
            linkText = 'View Original Auction Listing on Bonhams';
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
        document.getElementById('gameOverModal').style.display = 'flex';
    } else {
        document.getElementById('resultModal').style.display = 'flex';
    }
}

// Next round
function nextRound() {
    document.getElementById('resultModal').style.display = 'none';
    
    if (currentGame.mode === 'challenge') {
        if (currentGame.challengeSession.currentCar >= currentGame.challengeSession.cars.length) {
            // Challenge complete - show final results
            displayChallengeResults();
        } else {
            // Load next challenge car
            loadChallengeAuto();
            updateChallengeProgress();
        }
    } else {
        // Regular game modes
        loadNextCar();
    }
}

// End game
function endGame() {
    location.reload();
}

// Close modals when clicking outside
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        event.target.style.display = 'none';
    }
}

// Load data source information
async function loadDataSourceInfo() {
    try {
        const response = await fetch('/api/data-source');
        if (!response.ok) throw new Error('Failed to load data source info');
        
        const data = await response.json();
        const sourceInfo = document.getElementById('dataSourceInfo');
        
        let sourceName = data.data_source;
        if (sourceName === 'carwow') sourceName = 'CarWow';
        else if (sourceName === 'autotrader') sourceName = 'AutoTrader';
        else if (sourceName === 'mixed') sourceName = 'CarWow + AutoTrader';
        else if (sourceName === 'uk_realistic') sourceName = 'UK Realistic Data';
        else if (sourceName === 'bonhams_auctions') sourceName = 'Bonhams Car Auctions';
        else if (sourceName === 'collecting_cars_live') sourceName = 'Collecting Cars';
        
        sourceInfo.textContent = `Data source: ${sourceName} (${data.total_listings} listings)`;
    } catch (error) {
        console.error('Error loading data source info:', error);
    }
}

// Challenge Mode Functions
async function startChallengeMode() {
    try {
        // Start a new challenge session
        const response = await fetch('/api/challenge/start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) throw new Error('Failed to start challenge');
        
        const challengeSession = await response.json();
        currentGame.challengeSession = challengeSession;
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

function loadChallengeAuto() {
    const session = currentGame.challengeSession;
    if (session.currentCar >= session.cars.length) {
        return;
    }
    
    const currentCar = session.cars[session.currentCar];
    currentGame.currentListing = currentCar;
    displayCar(currentCar);
    
    // Reset guess inputs
    document.getElementById('priceGuess').value = '';
    document.getElementById('priceSlider').value = 25; // Set to £50k (middle of first range)
}

async function submitChallengeGuess(guessValue) {
    const submitButton = document.querySelector('.submit-button');
    submitButton.innerHTML = '<span class="loading"></span>';
    submitButton.disabled = true;
    
    try {
        const response = await fetch(`/api/challenge/${currentGame.challengeSession.sessionId}/guess`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                guessedPrice: guessValue
            })
        });
        
        if (!response.ok) throw new Error('Failed to submit challenge guess');
        
        const result = await response.json();
        currentGame.challengeGuesses.push(result);
        
        // Update challenge session
        currentGame.challengeSession.currentCar++;
        currentGame.challengeSession.totalScore = result.totalScore;
        currentGame.challengeSession.guesses.push(result);
        
        // Update score display
        document.getElementById('scoreValue').textContent = result.totalScore;
        
        if (result.sessionComplete) {
            // Show challenge complete modal
            displayChallengeResults();
        } else {
            // Show result and move to next car
            displayChallengeResult(result);
        }
        
    } catch (error) {
        console.error('Error submitting challenge guess:', error);
        alert('Failed to submit guess. Please try again.');
    } finally {
        submitButton.innerHTML = 'Submit Guess';
        submitButton.disabled = false;
    }
}

function displayChallengeResult(result) {
    // Update result modal content for challenge mode
    document.getElementById('resultTitle').textContent = `Car ${currentGame.challengeSession.currentCar}/10 Complete!`;
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
        originalLinkDiv.innerHTML = `<a href="${result.originalUrl}" target="_blank" class="original-link">View Original Auction Listing on Bonhams</a>`;
    } else {
        originalLinkDiv.style.display = 'none';
    }
    
    // Show modal with modified buttons
    const nextButton = document.querySelector('.next-button');
    nextButton.textContent = result.isLastCar ? 'View Results' : 'Next Car';
    
    document.getElementById('resultModal').style.display = 'flex';
}

function displayChallengeResults() {
    const session = currentGame.challengeSession;
    
    // Set final score
    document.getElementById('challengeFinalScore').textContent = `Final Score: ${session.totalScore.toLocaleString()} points`;
    
    // Build results breakdown
    const resultsDiv = document.getElementById('challengeResults');
    resultsDiv.innerHTML = '';
    
    session.guesses.forEach((guess, index) => {
        const car = session.cars[index];
        const resultItem = document.createElement('div');
        resultItem.className = 'challenge-result-item';
        
        const accuracy = (100 - guess.percentage).toFixed(1);
        
        resultItem.innerHTML = `
            <div class="challenge-result-car">${car.year} ${car.make} ${car.model}</div>
            <div class="challenge-result-accuracy">${accuracy}% accurate</div>
            <div class="challenge-result-points">${guess.points} pts</div>
        `;
        
        resultsDiv.appendChild(resultItem);
    });
    
    // Show challenge complete modal
    document.getElementById('challengeCompleteModal').style.display = 'flex';
}

// Initialize the app
document.addEventListener('DOMContentLoaded', () => {
    // Load data source info on page load
    loadDataSourceInfo();
    
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