// Game state
let currentGame = {
    mode: null,
    currentListing: null,
    score: 0,
    sessionId: generateSessionId()
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
    } else {
        scoreLabel.textContent = 'Streak: ';
    }
    
    // Load data source info
    loadDataSourceInfo();
    
    // Load first car
    loadNextCar();
}

// Load a random car listing
async function loadNextCar() {
    try {
        const response = await fetch('/api/random-listing');
        if (!response.ok) throw new Error('Failed to load listing');
        
        const listing = await response.json();
        currentGame.currentListing = listing;
        displayCar(listing);
        
        // Reset guess inputs to be synchronized
        document.getElementById('priceGuess').value = '';
        document.getElementById('priceSlider').value = 50000; // Set to a reasonable middle value
        
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
            { id: 'carMileage', value: car.mileage ? car.mileage.toLocaleString() + ' miles' : 'Unknown' },
            { id: 'carFuelType', value: car.fuelType || 'Unknown' },
            { id: 'carGearbox', value: car.gearbox || 'Unknown' },
            { id: 'carBodyType', value: car.bodyType || 'Unknown' },
            { id: 'carDoors', value: car.doors || 'Unknown' },
            { id: 'carSeats', value: car.seats || 'Unknown' }
        ];
        
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
    
    // Set main image with fade effect
    mainImage.style.opacity = '0';
    setTimeout(() => {
        mainImage.src = images[0];
        mainImage.style.opacity = '1';
    }, 200);
    
    // Create thumbnails if more than one image
    if (images.length > 1) {
        images.forEach((imageUrl, index) => {
            const thumbnail = document.createElement('img');
            thumbnail.src = imageUrl;
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

// Sync price input and slider with improved formatting
function syncInputToSlider() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    
    let value = input.value.replace(/,/g, ''); // Remove existing commas
    if (!isNaN(value) && value !== '') {
        const numValue = parseInt(value);
        const clampedValue = Math.min(Math.max(numValue, 0), 1000000);
        
        // Update slider
        slider.value = clampedValue;
        
        // Format input with commas (but don't trigger another event)
        input.value = numValue.toLocaleString();
    }
}

function syncSliderToInput() {
    const input = document.getElementById('priceGuess');
    const slider = document.getElementById('priceSlider');
    
    const value = parseInt(slider.value);
    input.value = value.toLocaleString();
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
    if (result.originalUrl && result.originalUrl.includes('motors')) {
        originalLinkDiv.style.display = 'block';
        originalLinkDiv.innerHTML = `<a href="${result.originalUrl}" target="_blank" class="original-link">View Original Listing on Motors.co.uk</a>`;
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
    loadNextCar();
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
        
        sourceInfo.textContent = `Data source: ${sourceName} (${data.total_listings} listings)`;
    } catch (error) {
        console.error('Error loading data source info:', error);
    }
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