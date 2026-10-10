package internal

import (
	"fmt"
	"math/big"
	"time"
)

type ETA struct {
	Time   time.Time
	Days   int64
	Rate   string
	Window string
}

var trailingWindows = map[string]int{
	"last1": 1,
	"last4": 4,
	"all":   -1,
}

// MovingAverageETA calculates up to three point estimates (1-week, 4-week, lifetime trailing average of weekly Δ).
// When the property is fully sold, ETAs are replaced with the sale period and average buy rate.
func (p *Property) calculateMovingAverageETA() error {
	if p.RemainingRaw.Sign() <= 0 {
		return p.calculateCompletedSaleDates()
	}

	if len(p.InitialSaleWeeklyPoints) < 1 {
		return fmt.Errorf("not enough data to calculate ETA")
	}

	// todo: map is random order
	trailingWindows = map[string]int{
		"last1": min(1, len(p.InitialSaleWeeklyPoints)),
		"last4": min(4, len(p.InitialSaleWeeklyPoints)),
		"all":   len(p.InitialSaleWeeklyPoints),
	}

	etas := make([]ETA, 0, len(trailingWindows))
	for name, weeks := range trailingWindows {
		eta, d, rate, err := p.etaFromTrailingWindow(weeks)
		if err != nil {
			return err
		}
		etas = append(etas, ETA{Time: eta, Days: d, Rate: rate, Window: name})
	}

	p.ETAs = etas

	return nil
}

func (p *Property) etaFromTrailingWindow(w int) (time.Time, int64, string, error) {
	sum := big.NewInt(0)
	from := len(p.InitialSaleWeeklyPoints) - w
	for j := from; j < len(p.InitialSaleWeeklyPoints); j++ {
		sum.Add(sum, p.InitialSaleWeeklyPoints[j].Value)
	}
	// sum / w = avg weekly Δ in the window
	avgRat := new(big.Rat).SetFrac(new(big.Int).Set(sum), big.NewInt(int64(w)))
	if avgRat.Sign() == 0 {
		return time.Time{}, 0, "", fmt.Errorf("average weekly Δ is zero over last %d weeks", w)
	}

	// daysRat = remaining / (weeklyRate / 7)
	remRat := new(big.Rat).SetInt(p.RemainingRaw)
	dailyRat := new(big.Rat).Quo(avgRat, big.NewRat(7, 1))
	daysRat := new(big.Rat).Quo(remRat, dailyRat)
	if daysRat.Sign() < 0 {
		return time.Time{}, 0, "", fmt.Errorf("negative days (unexpected)")
	}

	daysInt := ceilRatToInt64(daysRat)
	if daysInt < 0 {
		return time.Time{}, 0, "", fmt.Errorf("day count out of int64 range")
	}

	lastWeek := p.InitialSaleWeeklyPoints[len(p.InitialSaleWeeklyPoints)-1].Week
	eta := lastWeek.AddDate(0, 0, int(daysInt))
	rate := FormatBigRat(avgRat, p.Decimal, 1)
	return eta, daysInt, rate, nil
}

func (p *Property) calculateCompletedSaleDates() error {
	pts := p.InitialSaleWeeklyPoints
	if len(pts) == 0 {
		return fmt.Errorf("no weekly points for completed sale")
	}

	start := pts[0].Week
	end := pts[len(pts)-1].Week
	days := inclusiveUTCDays(start, end.AddDate(0, 0, 6))
	if days <= 0 {
		return fmt.Errorf("invalid sale period: %s – %s", start.Format(timeDateOnly), end.Format(timeDateOnly))
	}

	avgRat := new(big.Rat).SetFrac(new(big.Int).Set(p.TotalSupplyRaw), big.NewInt(int64(len(pts))))
	p.ETAs = []ETA{{
		Days:   days,
		Rate:   FormatBigRat(avgRat, p.Decimal, 1),
		Window: fmt.Sprintf("%s – %s", start.Format(timeDateOnly), end.Format(timeDateOnly)),
	}}

	return nil
}

func inclusiveUTCDays(start, end time.Time) int64 {
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)
	if end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Hours()/24) + 1
}

// ceilRatToInt64 returns ⌈x⌉ for x ≥ 0; for huge values beyond int64, returns -1.
func ceilRatToInt64(x *big.Rat) int64 {
	if x.Sign() <= 0 {
		return 0
	}
	num := new(big.Int).Set(x.Num())
	den := new(big.Int).Set(x.Denom())
	if den.Sign() == 0 {
		return -1
	}
	// ceil(num/den) = (num + den - 1) / den  for num ≥ 0, den > 0
	ceilNum := new(big.Int).Add(num, new(big.Int).Sub(den, big.NewInt(1)))
	q := new(big.Int).Quo(ceilNum, den)
	if !q.IsInt64() {
		return -1
	}
	return q.Int64()
}
