package internal

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/polygonscan"
)

const (
	// rewardsDistributor is the Binaryx claim contract on Polygon.
	// CoreManager posts rent here; holders pull USDT with claim.
	rewardsDistributor = "0x326fdcca0f5b9f5905f6bfdb914b1f6eddc45061"
	// paidRentTopic is PaidRent(address,address,uint256,uint128,uint128,uint256).
	// The log names the property token and the rent window.
	paidRentTopic = "0x015e7e41d528ca1976bac28c93fbbef520250f9e4837b66a42f879ec545ca9f4"
)

type rentalProfitMonth struct {
	Month time.Time
	USDT  *big.Int
}

type rentSlice struct {
	Token  string
	Amount *big.Int
	Start  time.Time
	End    time.Time
}

// FetchRentalProfit returns PaidRent slices from the claim contract, keyed by
// property token. Each slice is booked to the UTC month its window starts.
func FetchRentalProfit(client *polygonscan.Client, scanPause time.Duration) (map[string][]rentalProfitMonth, error) {
	logs, err := client.FetchLogs(rewardsDistributor, paidRentTopic, scanPause)
	if err != nil {
		return nil, fmt.Errorf("fetch rental profit: %w", err)
	}
	byToken := make(map[string][]rentSlice)
	for _, l := range logs {
		slice, ok := parsePaidRent(l)
		if !ok {
			continue
		}
		byToken[slice.Token] = append(byToken[slice.Token], slice)
	}
	out := make(map[string][]rentalProfitMonth, len(byToken))
	for token, slices := range byToken {
		out[token] = rentMonths(slices)
	}
	return out, nil
}

func parsePaidRent(l polygonscan.EventLog) (rentSlice, bool) {
	if len(l.Topics) < 3 {
		return rentSlice{}, false
	}
	topic := strings.TrimPrefix(l.Topics[2], "0x")
	if len(topic) < 40 {
		return rentSlice{}, false
	}
	words := extractWords(l.Data)
	if len(words) < 3 {
		return rentSlice{}, false
	}
	amount := wordInt(words[0])
	start := wordInt(words[1])
	end := wordInt(words[2])
	if amount == nil || start == nil || end == nil || amount.Sign() <= 0 {
		return rentSlice{}, false
	}
	if !start.IsInt64() || !end.IsInt64() || end.Cmp(start) <= 0 {
		return rentSlice{}, false
	}
	return rentSlice{
		Token:  wordAddress(topic),
		Amount: amount,
		Start:  time.Unix(start.Int64(), 0).UTC(),
		End:    time.Unix(end.Int64(), 0).UTC(),
	}, true
}

func rentMonths(slices []rentSlice) []rentalProfitMonth {
	byMonth := make(map[time.Time]*big.Int)
	var first, last time.Time
	for _, slice := range slices {
		if slice.Amount == nil || slice.Amount.Sign() <= 0 || !slice.End.After(slice.Start) {
			continue
		}
		month := monthStart(slice.Start)
		if cur := byMonth[month]; cur == nil {
			byMonth[month] = new(big.Int).Set(slice.Amount)
		} else {
			cur.Add(cur, slice.Amount)
		}
		if first.IsZero() || month.Before(first) {
			first = month
		}
		if last.IsZero() || month.After(last) {
			last = month
		}
	}
	if first.IsZero() {
		return nil
	}
	var out []rentalProfitMonth
	for month := first; !month.After(last); month = month.AddDate(0, 1, 0) {
		usdt := byMonth[month]
		if usdt == nil {
			usdt = big.NewInt(0)
		}
		out = append(out, rentalProfitMonth{Month: month, USDT: usdt})
	}
	return out
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// netRentalYield is (annualized rent profit / Binaryx price) × 100.
// The window is the last 12 months, or 6, 3, or 1 when history is shorter.
// A shorter window scales to a year: 6 months × 2, 3 × 4, 1 × 12.
func netRentalYield(months []rentalProfitMonth, binaryxPrice uint) (percent float64, basis int, ok bool) {
	monthsCount := len(months)
	switch {
	case monthsCount >= 12:
		basis = 12
	case monthsCount >= 6:
		basis = 6
	case monthsCount >= 3:
		basis = 3
	case monthsCount >= 1:
		basis = 1
	}
	if basis == 0 || binaryxPrice == 0 {
		return 0, 0, false
	}
	sum := big.NewInt(0)
	for _, month := range months[len(months)-basis:] {
		if month.USDT != nil {
			sum.Add(sum, month.USDT)
		}
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(polygonscan.USDTDecimal)), nil)
	annual := new(big.Rat).SetInt(sum)
	annual.Mul(annual, big.NewRat(12, int64(basis)))
	price := new(big.Int).Mul(scale, new(big.Int).SetUint64(uint64(binaryxPrice)))
	annual.Quo(annual, new(big.Rat).SetInt(price))
	annual.Mul(annual, big.NewRat(100, 1))
	f, _ := annual.Float64()
	return math.Round(f*100) / 100, basis, true
}
