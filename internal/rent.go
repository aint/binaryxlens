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
	// legacyRewardsDistributor emitted the same PaidRent event through 2024.
	legacyRewardsDistributor = "0x0cd8f3fe85780e620f80cdb884807174d9a1d805"
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

// FetchRentalProfit returns PaidRent slices from both distributors, keyed by
// property token. Each slice is booked to the UTC month its window starts.
func FetchRentalProfit(client *polygonscan.Client, scanPause time.Duration) (map[string][]rentalProfitMonth, error) {
	byToken := make(map[string][]rentSlice)
	for _, distributor := range []string{legacyRewardsDistributor, rewardsDistributor} {
		logs, err := client.FetchLogs(distributor, paidRentTopic, scanPause)
		if err != nil {
			return nil, fmt.Errorf("fetch rental profit: %w", err)
		}
		for _, l := range logs {
			slice, ok := parsePaidRent(l)
			if !ok {
				continue
			}
			byToken[slice.Token] = append(byToken[slice.Token], slice)
		}
	}
	through := monthStart(time.Now().UTC())
	out := make(map[string][]rentalProfitMonth, len(byToken))
	for token, slices := range byToken {
		out[token] = rentMonths(slices, through)
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

func sumRentUSDT(months []rentalProfitMonth) *big.Int {
	sum := big.NewInt(0)
	for _, month := range months {
		if month.USDT != nil {
			sum.Add(sum, month.USDT)
		}
	}
	return sum
}

func rentMonths(slices []rentSlice, through time.Time) []rentalProfitMonth {
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
	// Extend through the current month with zero rent so yields count missed months.
	end := last
	if t := monthStart(through); t.After(end) {
		end = t
	}
	var out []rentalProfitMonth
	for month := first; !month.After(end); month = month.AddDate(0, 1, 0) {
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

// trailingNetRentalYield annualizes the last n months. When allowShort is set
// and the history is shorter than n, the whole history is used and basis is that length.
func trailingNetRentalYield(months []rentalProfitMonth, value uint, n int, allowShort bool) (percent float64, basis int, ok bool) {
	if n <= 0 || len(months) == 0 || (!allowShort && len(months) < n) {
		return 0, 0, false
	}
	window := months
	if len(months) > n {
		window = months[len(months)-n:]
	}
	return netRentalYield(window, value)
}

func allTimeNetRentalYield(months []rentalProfitMonth, value uint) (percent float64, ok bool) {
	percent, _, ok = netRentalYield(months, value)
	return percent, ok
}

// netRentalYield is (annualized net rent profit / current value) × 100.
// The total in window is scaled to a year by 12 / len(window).
func netRentalYield(window []rentalProfitMonth, value uint) (percent float64, basis int, ok bool) {
	basis = len(window)
	if basis == 0 || value == 0 {
		return 0, 0, false
	}
	sum := big.NewInt(0)
	for _, month := range window {
		if month.USDT != nil {
			sum.Add(sum, month.USDT)
		}
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(polygonscan.USDTDecimal)), nil)
	annual := new(big.Rat).SetInt(sum)
	annual.Mul(annual, big.NewRat(12, int64(basis)))
	price := new(big.Int).Mul(scale, new(big.Int).SetUint64(uint64(value)))
	annual.Quo(annual, new(big.Rat).SetInt(price))
	annual.Mul(annual, big.NewRat(100, 1))
	f, _ := annual.Float64()
	return math.Round(f*100) / 100, basis, true
}
