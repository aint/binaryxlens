package internal

import (
	"math/big"
	"slices"
	"time"
)

type WeeklyPoint struct {
	Week     time.Time // Monday 00:00 UTC
	Value    *big.Int
	CumValue *big.Int
}

func (p *Property) buildInitialSaleWeeklySeries() {
	p.InitialSaleWeeklyPoints = p.buildWeeklySeries(func(t transfer) bool {
		return p.isInitialSale(t.From)
	}, true)
}

func (p *Property) buildP2PSaleWeeklySeries() {
	p.P2PSaleWeeklyPoints = p.buildWeeklySeries(func(t transfer) bool {
		return p.isP2PTransfer(t.From, t.To)
	}, false)
}

// buildWeeklySeries buckets matching transfers by Monday UTC. When stopAtSupply
// is set, the series ends on the week cumulative volume reaches total supply.
func (p *Property) buildWeeklySeries(match func(transfer) bool, stopAtSupply bool) []WeeklyPoint {
	timelineMap := make(map[time.Time]*big.Int)
	for _, t := range p.transfers {
		if !match(t) {
			continue
		}

		week := startOfWeekUTC(t.Time)
		cur := timelineMap[week]
		if cur == nil {
			cur = big.NewInt(0)
		}
		timelineMap[week] = new(big.Int).Add(cur, t.Value)
	}

	if len(timelineMap) == 0 {
		return nil
	}

	weeks := make([]time.Time, 0, len(timelineMap))
	for week := range timelineMap {
		weeks = append(weeks, week)
	}
	slices.SortFunc(weeks, func(a, b time.Time) int { return a.Compare(b) })

	cumValue := big.NewInt(0)
	series := make([]WeeklyPoint, 0, len(weeks))
	for _, week := range weeks {
		value := timelineMap[week]
		cumValue = new(big.Int).Add(cumValue, value)
		series = append(series, WeeklyPoint{
			Week:     week,
			Value:    value,
			CumValue: new(big.Int).Set(cumValue),
		})
		if stopAtSupply && cumValue.Cmp(p.TotalSupplyRaw) >= 0 {
			break
		}
	}

	return series
}

func startOfWeekUTC(t time.Time) time.Time {
	t = t.UTC().Truncate(24 * time.Hour)
	offset := (int(t.Weekday()) + 6) % 7 // Monday = week start
	return t.AddDate(0, 0, -offset)
}
