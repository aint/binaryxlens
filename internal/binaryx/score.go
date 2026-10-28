package binaryx

import (
	"math"
	"time"
)

type scoreKnot struct {
	x, s float64
}

var (
	priceScoreKnots = []scoreKnot{
		{-0.35, 100},
		{-0.2, 80},
		{0, 40},
		{0.2, 20},
		{0.35, 0},
	}
	yieldScoreKnots = []scoreKnot{
		{0, 0},
		{3, 20},
		{6, 60},
		{10, 90},
		{16, 100},
	}
	ageScoreKnots = []scoreKnot{
		{8, 100},
		{15, 80},
		{22, 40},
		{27, 20},
		{30, 0},
	}
)

const (
	yieldTrailingMonths = 3
	ageWeight           = 0.15
	priceWeight         = 0.35
	yieldWeight         = 0.50
	leaseYears          = 30
)

// score is the rounded mean of each property that has a score.
func (pr *Project) Score() int {
	total, n := 0, 0
	for _, property := range pr.Properties {
		score := property.Score()
		if score == -1 {
			continue
		}
		total += score
		n++
	}
	if n == 0 {
		return -1
	}
	return (total + n/2) / n
}

// non-rental properties return 0 instead of -1

// score is the 1–100 rental property score. It is present only when
// both the price part and the yield part can be computed.
func (p *Property) Score() int {
	price, okPrice := p.priceScore()
	yield, okYield := p.yieldScore()
	if !okPrice || !okYield {
		return -1
	}
	var raw float64
	if age, okAge := p.ageScore(time.Now().UTC()); okAge {
		raw = priceWeight*price + yieldWeight*yield + ageWeight*age
	} else {
		raw = priceWeight*price + yieldWeight*yield
	}
	n := int(math.Round(raw))
	if n < 1 {
		return 1
	}
	if n > 100 {
		return 100
	}
	return n
}

func (p *Property) priceScore() (float64, bool) {
	if p.CurrentValue.Price == 0 || p.AreaM2 == 0 {
		return 0, false
	}
	mid := float64(p.MarketLowValue+p.MarketHighValue) / 2
	if mid == 0 {
		return 0, false
	}
	delta := (float64(p.CurrentValue.Price)/float64(p.AreaM2) - mid) / mid
	return interpolatePart(delta, priceScoreKnots), true
}

func (p *Property) yieldScore() (float64, bool) {
	percent, months, ok := TrailingNetRentalYield(p.RentalProfitMonths, p.CurrentValue.Price, yieldTrailingMonths, true)
	if !ok || months < yieldTrailingMonths {
		return 0, false
	}
	return interpolatePart(percent, yieldScoreKnots), true
}

// Add unit tests
// case age is 0 (just built)

func (p *Property) ageScore(now time.Time) (float64, bool) {
	if p.RentalStartActual == nil {
		return 0, false
	}
	quarters := NewYearQuarter(now).QuartersSince(*p.RentalStartActual)
	if quarters < 0 {
		quarters = 0
	}
	years := float64(quarters) / 4
	if years > leaseYears {
		years = leaseYears
	}
	return interpolatePart(years, ageScoreKnots), true
}

// interpolatePart maps x onto 0–100 through knots. Below the first x the
// part is the first score; above the last x it is the last score.
func interpolatePart(x float64, knots []scoreKnot) float64 {
	if len(knots) == 0 {
		return 0
	}
	if x <= knots[0].x {
		return knots[0].s
	}
	last := knots[len(knots)-1]
	if x >= last.x {
		return last.s
	}
	for i := 1; i < len(knots); i++ {
		if x <= knots[i].x {
			x0, s0 := knots[i-1].x, knots[i-1].s
			x1, s1 := knots[i].x, knots[i].s
			return s0 + (s1-s0)*(x-x0)/(x1-x0)
		}
	}
	return last.s
}

func ScoreTier(score int) string {
	switch {
	case score >= 70:
		return "high"
	case score >= 40:
		return "mid"
	case score >= 1:
		return "low"
	default:
		return ""
	}
}
