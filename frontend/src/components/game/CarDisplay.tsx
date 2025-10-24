import { useState, useEffect } from 'react';
import type { CarListing } from '../../types';

interface CarDisplayProps {
  car: CarListing;
}

export const CarDisplay = ({ car }: CarDisplayProps) => {
  const [mainImage, setMainImage] = useState(car.images[0] || '');
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    setMainImage(car.images[0] || '');
    setActiveIndex(0);
  }, [car]);

  const handleImageError = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const imgSrc = e.currentTarget.src;

    // Only set placeholder if not already a placeholder
    if (!imgSrc.includes('image-unavailable.png')) {
      // Show Europe warning if Easy mode
      const warning = document.getElementById('europeWarning');
      if (warning && warning.style.display === 'none') {
        warning.style.display = 'flex';
      }

      // Set placeholder image
      e.currentTarget.src = 'https://www.travelodge.co.uk/nw/assets/img/photo/image-unavailable.png';
      e.currentTarget.style.filter = 'grayscale(1)';
      e.currentTarget.onerror = null; // Prevent infinite loop if placeholder fails
    }
  };

  const switchImage = (imageUrl: string, index: number) => {
    setMainImage(imageUrl);
    setActiveIndex(index);
  };

  // Get display title
  const getDisplayTitle = () => {
    if (!car.auctionDetails && car.fullTitle && car.fullTitle.includes(' - ')) {
      const parts = car.fullTitle.split(' - ');
      return parts[0];
    }
    return `${car.year || 'Unknown'} ${car.make || 'Unknown'} ${car.model || 'Unknown'}`;
  };

  return (
    <div className="car-display">
      <div className="car-images">
        <div className="image-gallery" id="imageGallery">
          <img
            id="mainCarImage"
            src={mainImage || undefined}
            alt="Car Image"
            className="main-car-image"
            onError={handleImageError}
          />
          {car.images.length > 1 && (
            <div className="thumbnail-strip" id="thumbnailStrip">
              {car.images.map((imageUrl, index) => (
                <img
                  key={index}
                  src={imageUrl || undefined}
                  alt={`Car thumbnail ${index + 1}`}
                  className={`thumbnail ${index === activeIndex ? 'active' : ''}`}
                  onClick={() => switchImage(imageUrl, index)}
                  onError={handleImageError}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="car-info">
        <h2 id="carTitle">{getDisplayTitle()}</h2>

        {!car.auctionDetails && car.trim && (
          <div className="detail-row" id="trimRow">
            <span className="detail-label">Trim:</span>
            <span id="carTrim" className="detail-value">{car.trim}</span>
          </div>
        )}

        <div className="car-details">
          <div className="detail-row">
            <span className="detail-label">Year:</span>
            <span id="carYear" className="detail-value">{car.year || 'Unknown'}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Engine Size:</span>
            <span id="carEngine" className="detail-value">{car.engine || 'Unknown'}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Mileage:</span>
            <span id="carMileage" className="detail-value">
              {car.mileageFormatted || (car.mileage ? car.mileage.toLocaleString() + ' miles' : 'Unknown')}
            </span>
          </div>

          {!car.auctionDetails && car.owners && (
            <div className="detail-row" id="ownersRow">
              <span className="detail-label">Owners:</span>
              <span id="carOwners" className="detail-value">{car.owners}</span>
            </div>
          )}

          <div className="detail-row">
            <span className="detail-label">Fuel Type:</span>
            <span id="carFuelType" className="detail-value">{car.fuelType || 'Unknown'}</span>
          </div>

          {!car.auctionDetails && car.bodyType && (
            <div className="detail-row" id="bodyTypeRow">
              <span className="detail-label">Body Type:</span>
              <span id="carBodyType" className="detail-value">{car.bodyType}</span>
            </div>
          )}

          <div className="detail-row">
            <span className="detail-label">Gearbox:</span>
            <span id="carGearbox" className="detail-value">{car.gearbox || 'Unknown'}</span>
          </div>

          {!car.auctionDetails && car.doors && (
            <div className="detail-row" id="doorsRow">
              <span className="detail-label">Doors:</span>
              <span id="carDoors" className="detail-value">{car.doors}</span>
            </div>
          )}

          <div className="detail-row">
            <span className="detail-label">Exterior Color:</span>
            <span id="carBodyColour" className="detail-value">
              {car.bodyColour || car.exteriorColor || 'Unknown'}
            </span>
          </div>

          {car.location && (
            <div className="detail-row" id="locationRow">
              <span className="detail-label">Location:</span>
              <span id="carLocation" className="detail-value">{car.location}</span>
            </div>
          )}

          {/* Hard Mode Only Fields */}
          {car.auctionDetails && (
            <>
              {car.saleDate && (
                <div className="detail-row" id="saleDateRow">
                  <span className="detail-label">Sale Date:</span>
                  <span id="carSaleDate" className="detail-value">{car.saleDate}</span>
                </div>
              )}
              {car.interiorColor && (
                <div className="detail-row" id="interiorColorRow">
                  <span className="detail-label">Interior Color:</span>
                  <span id="carInteriorColor" className="detail-value">{car.interiorColor}</span>
                </div>
              )}
              {car.steering && (
                <div className="detail-row" id="steeringRow">
                  <span className="detail-label">Steering:</span>
                  <span id="carSteering" className="detail-value">{car.steering}</span>
                </div>
              )}
            </>
          )}
        </div>

        {/* Auction Details Section */}
        {car.auctionDetails && car.keyFacts && car.keyFacts.length > 0 && (
          <div id="auctionDetailsSection" className="auction-details">
            <h3>Auction Details</h3>
            <div id="keyFactsSection" className="key-facts">
              <h4>Key Facts</h4>
              <ul id="keyFactsList" className="key-facts-list">
                {car.keyFacts.map((fact, index) => (
                  <li key={index}>{fact}</li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
