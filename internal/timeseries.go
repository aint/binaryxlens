package internal

import (
	"math/big"
	"time"
	"slices"
)

type DailyPoint struct {
	Day        time.Time
	Value      *big.Int
	CumValue   *big.Int
	CumPercent float64
}

type WeeklyPoint struct {
	Week     time.Time // Monday 00:00 UTC
	Value    *big.Int
	CumValue *big.Int
}

func (p *Property) buildInitialSaleDailySeries() {
	var start, end time.Time
	timelineMap := make(map[time.Time]*big.Int)
	for _, t := range p.transfers {
		day := t.Time.Truncate(24 * time.Hour)
		if start.IsZero() || day.Before(start) {
			start = day
		}
		if day.After(end) {
			end = day
		}

		if p.isInitialSale(t.From) {
			cur := timelineMap[day]
			if cur == nil {
				cur = big.NewInt(0)
			}
			timelineMap[day] = new(big.Int).Add(cur, t.Value)
		}
	}

	cumValue := big.NewInt(0)
	dailyPoints := make([]DailyPoint, 0, len(timelineMap))
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		value, ok := timelineMap[d]
		if !ok {
			value = big.NewInt(0)
		}
		cumValue = new(big.Int).Add(cumValue, value)
		pct, _ := new(big.Rat).Mul(
			new(big.Rat).SetFrac(cumValue, p.TotalSupplyRaw),
			big.NewRat(100, 1),
		).Float64()
		dailyPoints = append(dailyPoints, DailyPoint{Day: d, Value: value, CumValue: cumValue, CumPercent: pct})
		if cumValue.Cmp(p.TotalSupplyRaw) == 0 {
			break
		}
	}

	p.InitialSaleDailyPoints = dailyPoints
}

func (p *Property) buildP2PSaleWeeklySeries() {
	timelineMap := make(map[time.Time]*big.Int)
	for _, t := range p.transfers {
		if !p.isP2PTransfer(t.From, t.To) {
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
		return
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
	}

	p.P2PSaleWeeklyPoints = series
}

func startOfWeekUTC(t time.Time) time.Time {
	t = t.UTC().Truncate(24 * time.Hour)
	offset := (int(t.Weekday()) + 6) % 7 // Monday = week start
	return t.AddDate(0, 0, -offset)
}
