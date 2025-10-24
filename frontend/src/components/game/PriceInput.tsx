import { useState, useEffect } from 'react';

interface PriceInputProps {
  onSubmit: (price: number) => void;
  disabled?: boolean;
  resetTrigger?: unknown; // Any value that changes to trigger reset
}

// Convert price to slider position (0-100)
const priceToSlider = (price: number): number => {
  if (price <= 100000) {
    // First half: £0-£100k maps to 0-50
    return (price / 100000) * 50;
  } else {
    // Second half: £100k-£500k maps to 50-100
    const priceAbove100k = Math.min(price - 100000, 400000);
    return 50 + (priceAbove100k / 400000) * 50;
  }
};

// Convert slider position (0-100) to price
const sliderToPrice = (sliderValue: number): number => {
  if (sliderValue <= 50) {
    // First half: 0-50 maps to £0-£100k
    return (sliderValue / 50) * 100000;
  } else {
    // Second half: 50-100 maps to £100k-£500k
    const extraValue = (sliderValue - 50) / 50;
    return 100000 + extraValue * 400000;
  }
};

export const PriceInput = ({ onSubmit, disabled = false, resetTrigger }: PriceInputProps) => {
  const [inputValue, setInputValue] = useState('');
  const [sliderValue, setSliderValue] = useState(50);
  const [overlayText, setOverlayText] = useState('Enter price');

  useEffect(() => {
    // Reset when resetTrigger changes (new car loaded)
    setInputValue('');
    setSliderValue(50);
    setOverlayText('Enter price');
  }, [resetTrigger]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setInputValue(value);

    if (value && !isNaN(Number(value))) {
      const numValue = parseInt(value);
      const sliderVal = Math.min(Math.max(numValue, 0), 500000);
      setSliderValue(priceToSlider(sliderVal));
      setOverlayText(numValue.toLocaleString());
    } else {
      setOverlayText('Enter price');
    }
  };

  const handleSliderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const sliderVal = parseFloat(e.target.value);
    setSliderValue(sliderVal);

    const price = Math.round(sliderToPrice(sliderVal));
    setInputValue(price.toString());
    setOverlayText(price.toLocaleString());
  };

  const handleSubmit = () => {
    const guessValue = parseInt(inputValue);

    if (!guessValue || guessValue <= 0) {
      // Shake the input
      const input = document.getElementById('priceGuess');
      if (input) {
        input.style.animation = 'shake 0.5s';
        setTimeout(() => {
          input.style.animation = '';
        }, 500);
      }
      return;
    }

    onSubmit(guessValue);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSubmit();
    }
  };

  return (
    <div className="guess-section">
      <h3>What's Your Price Guess?</h3>
      <div className="price-input-container">
        <span className="currency">£</span>
        <div className="price-input-wrapper">
          <input
            type="number"
            id="priceGuess"
            className="price-input"
            inputMode="numeric"
            min="0"
            max="10000000"
            step="1"
            value={inputValue}
            onChange={handleInputChange}
            onKeyPress={handleKeyPress}
            disabled={disabled}
          />
          <div id="priceOverlay" className="price-overlay" style={{
            color: inputValue ? '#fff' : 'rgba(255, 255, 255, 0.5)',
            fontStyle: inputValue ? 'normal' : 'italic'
          }}>
            {overlayText}
          </div>
        </div>
      </div>
      <div className="price-slider-container">
        <input
          type="range"
          id="priceSlider"
          className="price-slider"
          min="0"
          max="100"
          step="1"
          value={sliderValue}
          onChange={handleSliderChange}
          disabled={disabled}
        />
        <div className="slider-labels">
          <span>£0</span>
          <span className="mid-label">£100k</span>
          <span>£500k</span>
        </div>
        <div className="slider-help">
          <small>Left half: £0-£100k • Right half: £100k-£500k</small>
        </div>
      </div>
      <button
        id="submitGuess"
        className="submit-button"
        onClick={handleSubmit}
        disabled={disabled}
      >
        Submit Guess
      </button>
    </div>
  );
};
