package binaryx

import (
	"math/big"
	"testing"
	"time"
)

func scoredProperty() *Property {
	return &Property{
		AreaM2:          1,
		CurrentValue:    Value{Price: 1_000_000},
		MarketLowValue:  1_000_000,
		MarketHighValue: 1_000_000,
		RentalProfitMonths: []RentalProfitMonth{
			{Month: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(75_000)},
			{Month: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(75_000)},
			{Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(75_000)},
		},
	}
}

func TestPropertyScore(t *testing.T) {
	t.Run("scores a property with price and yield data", func(t *testing.T) {
		if got := scoredProperty().Score(); got != 55 {
			t.Fatalf("Score() = %d, want 55", got)
		}
	})

	t.Run("requires price and yield data", func(t *testing.T) {
		p := scoredProperty()
		p.CurrentValue.Price = 0
		if got := p.Score(); got != -1 {
			t.Fatalf("Score() without price = %d, want -1", got)
		}

		p = scoredProperty()
		p.RentalProfitMonths = nil
		if got := p.Score(); got != -1 {
			t.Fatalf("Score() without yield history = %d, want -1", got)
		}
	})
}

func TestProjectScore(t *testing.T) {
	t.Run("returns -1 when no property can be scored", func(t *testing.T) {
		if got := (&Project{}).Score(); got != -1 {
			t.Fatalf("Score() = %d, want -1", got)
		}
	})

	t.Run("averages only scored properties and rounds", func(t *testing.T) {
		p1, p2 := scoredProperty(), scoredProperty()
		p2.CurrentValue.Price = 0
		project := &Project{Properties: []*Property{p1, p2}}
		if got := project.Score(); got != 55 {
			t.Fatalf("Score() = %d, want 55", got)
		}
	})
}

func TestScoreTier(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{"missing", -1, ""},
		{"zero", 0, ""},
		{"low minimum", 1, "low"},
		{"low maximum", 39, "low"},
		{"mid minimum", 40, "mid"},
		{"mid maximum", 69, "mid"},
		{"high minimum", 70, "high"},
		{"high maximum", 100, "high"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScoreTier(tt.score); got != tt.want {
				t.Errorf("ScoreTier(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}
