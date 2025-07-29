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
        
        // Reset guess inputs
        document.getElementById('priceGuess').value = '';
        document.getElementById('priceSlider').value = 50000;
        
    } catch (error) {
        console.error('Error loading car:', error);
        alert('Failed to load car listing. Please try again.');
    }
}

// Display car information
function displayCar(car) {
    // Set car image
    const carImage = document.getElementById('carImage');
    carImage.src = (car.images && car.images.length > 0) ? car.images[0] : 'https://via.placeholder.com/600x400?text=No+Image';
    
    // Set car title
    document.getElementById('carTitle').textContent = `${car.year || 'Unknown'} ${car.make || 'Unknown'} ${car.model || 'Unknown'}`;
    
    // Set car details
    document.getElementById('carYear').textContent = car.year || 'Unknown';
    document.getElementById('carEngine').textContent = car.engine || 'Unknown';
    document.getElementById('carMileage').textContent = car.mileage ? car.mileage.toLocaleString() + ' miles' : 'Unknown';
    document.getElementById('carFuelType').textContent = car.fuelType || 'Unknown';
    document.getElementById('carGearbox').textContent = car.gearbox || 'Unknown';
    document.getElementById('carBodyType').textContent = car.bodyType || 'Unknown';
}

// Sync price input and slider
document.getElementById('priceGuess').addEventListener('input', function(e) {
    const value = parseInt(e.target.value) || 0;
    document.getElementById('priceSlider').value = Math.min(Math.max(value, 5000), 100000);
});

document.getElementById('priceSlider').addEventListener('input', function(e) {
    document.getElementById('priceGuess').value = e.target.value;
});

// Submit guess
async function submitGuess() {
    const guessValue = parseInt(document.getElementById('priceGuess').value);
    
    if (!guessValue || guessValue <= 0) {
        alert('Please enter a valid price guess');
        return;
    }
    
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
    
    // Allow Enter key to submit guess
    const priceGuess = document.getElementById('priceGuess');
    priceGuess.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            submitGuess();
        }
    });
});