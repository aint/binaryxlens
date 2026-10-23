package internal

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
		{-0.3, 90},
		{-0.2, 80},
		{-0.1, 60},
		{0, 50},
		{0.1, 40},
		{0.2, 20},
		{0.3, 10},
		{0.35, 0},
	}
	yieldScoreKnots = []scoreKnot{
		{0, 0},
		{3, 20},
		{5, 40},
		{8, 60},
		{12, 80},
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
	yieldAnchorPct   = 8.0
	minYieldMonths   = 3
	fullYieldMonths  = 12
	priceYieldWeight = 0.40
	ageWeight        = 0.20
	fiftyFiftyWeight = 0.50
	leaseYears       = 30
)

// score is the rounded mean of each property that has a score.
func (pr *Project) score() int {
	total, n := 0, 0
	for _, property := range pr.Properties {
		score := property.score()
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
func (p *Property) score() int {
	price, okPrice := p.priceScore()
	yield, okYield := p.yieldScore()
	if !okPrice || !okYield {
		return -1
	}
	var raw float64
	if age, okAge := p.ageScore(time.Now().UTC()); okAge {
		raw = priceYieldWeight*price + priceYieldWeight*yield + ageWeight*age
	} else {
		raw = fiftyFiftyWeight*price + fiftyFiftyWeight*yield
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
	if p.BinaryxRaisedValue == 0 || p.AreaM2 == 0 {
		return 0, false
	}
	mid := float64(p.MarketLowValue+p.MarketHighValue) / 2
	if mid == 0 {
		return 0, false
	}
	delta := (float64(p.BinaryxRaisedValue)/float64(p.AreaM2) - mid) / mid
	return interpolatePart(delta, priceScoreKnots), true
}

func (p *Property) yieldScore() (float64, bool) {
	percent, months, ok := trailingNetRentalYield(p.RentalProfitMonths, p.BinaryxRaisedValue, fullYieldMonths, true)
	if !ok || months < minYieldMonths {
		return 0, false
	}
	weight := float64(months) / fullYieldMonths
	if weight > 1 {
		weight = 1
	}
	used := yieldAnchorPct + (percent-yieldAnchorPct)*weight
	return interpolatePart(used, yieldScoreKnots), true
}

func (p *Property) ageScore(now time.Time) (float64, bool) {
	if p.RentalStartActual == nil {
		return 0, false
	}
	quarters := yearQuarterAt(now).quartersSince(*p.RentalStartActual)
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

func scoreTier(score int) string {
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
