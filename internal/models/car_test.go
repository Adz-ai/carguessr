package models

import (
	"reflect"
	"testing"
)

func TestBonhamsCarConversions(t *testing.T) {
	car := &BonhamsCar{
		ID:             "1",
		Make:           "Ferrari",
		Model:          "Testarossa",
		Year:           1987,
		Price:          250000,
		Images:         []string{"image1"},
		OriginalURL:    "http://example.com",
		Mileage:        "41,565 Miles",
		Engine:         "5000cc",
		Gearbox:        "manual",
		ExteriorColor:  "Red",
		InteriorColor:  "Black",
		Steering:       "Left",
		FuelType:       "Petrol",
		KeyFacts:       []string{"iconic"},
		MileageNumeric: 41565,
		Description:    "Legendary",
		SaleDate:       "2025-01-10",
		Location:       "Maranello",
	}

	standard := car.ToStandardCar()
	if standard == nil {
		t.Fatal("expected non-nil standard car")
	}
	if standard.Make != car.Make || standard.Engine != car.Engine || standard.BodyColour != car.ExteriorColor {
		t.Fatalf("unexpected standard car conversion: %+v", standard)
	}

	enhanced := car.ToEnhancedCar()
	if enhanced == nil {
		t.Fatal("expected non-nil enhanced car")
	}

	if enhanced.AuctionDetails != true {
		t.Fatalf("expected AuctionDetails true, got %+v", enhanced)
	}

	if !reflect.DeepEqual(enhanced.KeyFacts, car.KeyFacts) {
		t.Fatalf("expected key facts %v, got %v", car.KeyFacts, enhanced.KeyFacts)
	}
	if enhanced.ExteriorColor != car.ExteriorColor || enhanced.Steering != car.Steering {
		t.Fatalf("unexpected enhanced conversion: %+v", enhanced)
	}
}

func TestLookersCarToEnhancedCar(t *testing.T) {
	car := &LookersCar{
		ID:       "2",
		Make:     "Land Rover",
		Model:    "Range Rover",
		Title:    "LAND ROVER RANGE ROVER ESTATE - 2025 3.0 P460E Autobiography 4Dr Auto",
		Location: "London",
		BodyType: "SUV",
		Characteristics: map[string]string{
			"Mileage":      "75,851 miles",
			"Registration": "XX21 XXX",
			"Owners":       "1",
			"Fuel Type":    "Hybrid",
			"Engine Size":  "3.0L",
			"Transmission": "Automatic",
			"Doors":        "5",
			"Color":        "Black",
			"Year":         "2024",
		},
	}

	enhanced := car.ToEnhancedCar()
	if enhanced.AuctionDetails {
		t.Fatalf("expected AuctionDetails false, got %+v", enhanced)
	}

	if enhanced.Trim != "3.0 P460E Autobiography 4Dr Auto" {
		t.Fatalf("unexpected trim extracted: %q", enhanced.Trim)
	}

	if enhanced.Mileage != 75851 {
		t.Fatalf("unexpected mileage: %d", enhanced.Mileage)
	}

	if enhanced.Year != 2024 {
		t.Fatalf("expected year 2024, got %d", enhanced.Year)
	}

	if enhanced.FullTitle != car.Title {
		t.Fatalf("expected full title preserved")
	}
}
