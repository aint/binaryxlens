package internal

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
)

type DailyPoint struct {
	Day        time.Time
	Value      *big.Int
	CumValue   *big.Int
	CumPercent float64
}

func (p *Property) buildDailySeries() error {
	var start, end time.Time
	timelineMap := make(map[time.Time]*big.Int)
	for _, tx := range p.txs {
		ts, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			return fmt.Errorf("parse timestamp %q: %w", tx.TimeStamp, err)
		}
		day := time.Unix(ts, 0).UTC().Truncate(24 * time.Hour)
		if start.IsZero() || day.Before(start) {
			start = day
		}
		if day.After(end) {
			end = day
		}

		value, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return fmt.Errorf("parse value %q: %w", tx.Value, err)
		}

		if p.isInitialSale(tx.From) {
			cur := timelineMap[day]
			if cur == nil {
				cur = big.NewInt(0)
			}
			timelineMap[day] = new(big.Int).Add(cur, value)
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

	p.DailyPoints = dailyPoints

	return nil
}
