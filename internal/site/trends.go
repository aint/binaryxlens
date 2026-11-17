package site

import (
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aint/binaryxlens/internal/binaryx"
	"github.com/aint/binaryxlens/internal/polygonscan"
)

const ceoWallet = "0xdf24253c84f16aac030917c2672720b11610e109"

//go:embed trends.html
var trendsTemplateSrc []byte

var trendsPlaceholder = []byte("__TRENDS__")

var trendBuckets = []struct {
	name string
	min  *big.Int
	max  *big.Int
}{
	{"whale", big.NewInt(100_000_000000), nil},
	{"shark", big.NewInt(50_000_000000), big.NewInt(100_000_000000)},
	{"dolphin", big.NewInt(20_000_000000), big.NewInt(50_000_000000)},
	{"fish", big.NewInt(10_000_000000), big.NewInt(20_000_000000)},
	{"crab", big.NewInt(5_000_000000), big.NewInt(10_000_000000)},
	{"shrimp", big.NewInt(0), big.NewInt(5_000_000000)},
}

type trend struct {
	Name       string
	DeltasUSDT []*big.Int
}

func WriteTrends(path string, updated time.Time, projects []*binaryx.Project) error {
	payload := []byte(trendHTML(projects))
	return write(path, updated, trendsTemplateSrc, trendsPlaceholder, payload, headerConfig{activePage: "trends"})
}

func trendHTML(projects []*binaryx.Project) string {
	var b strings.Builder
	for _, t := range trends(globalHolders(projects)) {
		delta := new(big.Int)
		for _, window := range t.DeltasUSDT {
			if window != nil {
				delta.Add(delta, window)
			}
		}
		value, class := "—", "flat"
		if delta.Sign() != 0 {
			sign := "+"
			class = "up"
			if delta.Sign() < 0 {
				sign, class = "-", "down"
			}
			value = sign + `<span class="delta-number">` + binaryx.FormatBigInt(new(big.Int).Abs(delta), polygonscan.USDTDecimal) + `</span>`
		}
		series := append([]*big.Int(nil), t.DeltasUSDT...)
		spark := ""
		if delta.Sign() != 0 {
			spark = sparkline(series)
		}
		fmt.Fprintf(&b, `
				<div class="trend-row">
					<span class="item"><span class="name" data-i18n="trends.type.%s">%s</span></span>
					<span class="trend-metrics">
						%s
						<span class="delta %s">%s</span>
					</span>
				</div>`,
			t.Name,
			t.Name,
			spark,
			class,
			value,
		)
	}
	return b.String()
}

func trends(holders []globalHolder) []trend {
	trends := make([]trend, len(trendBuckets))
	for i, bucket := range trendBuckets {
		trends[i] = trend{
			Name:       bucket.name,
			DeltasUSDT: make([]*big.Int, trendWindowCount),
		}
		for window := range trends[i].DeltasUSDT {
			trends[i].DeltasUSDT[window] = new(big.Int)
		}
	}

	for _, h := range holders {
		if h.Address == ceoWallet {
			trends = append(trends, trend{
				Name:       "ceo",
				DeltasUSDT: activityUSDTSeries(h.DailyActivity),
			})
		}
		for i, bucket := range trendBuckets {
			if h.TotalUSDT.Cmp(bucket.min) >= 0 && (bucket.max == nil || h.TotalUSDT.Cmp(bucket.max) < 0) {
				for window, delta := range activityUSDTSeries(h.DailyActivity) {
					trends[i].DeltasUSDT[window].Add(trends[i].DeltasUSDT[window], delta)
				}
				break
			}
		}
	}

	return trends
}

const (
	trendWindowCount = 3
	trendWindowDays  = 30
)

func activityUSDTSeries(activity binaryx.DailyActivity) []*big.Int {
	series := make([]*big.Int, trendWindowCount)
	for i := range series {
		window := i + 1
		if i == trendWindowCount-1 {
			window = 0
		}
		series[i] = activityUSDTDelta(activity, window)
	}
	return series
}

func activityUSDTDelta(activity binaryx.DailyActivity, window int) *big.Int {
	result := new(big.Int)
	today := binaryx.NewDay(time.Now().UTC())
	start := time.Time(today).AddDate(0, 0, -(window+1)*trendWindowDays)
	end := time.Time(today).AddDate(0, 0, -window*trendWindowDays)
	for day, a := range activity {
		d := time.Time(day)
		if d.Before(start) || !d.Before(end) {
			continue
		}
		result.Add(result, a.UsdtIn)
		result.Add(result, a.UsdtOut)
	}
	return result
}

func sparkline(values []*big.Int) string {
	if len(values) < 2 {
		return ""
	}
	const w, h, pad = 200.0, 24.0, 2.0
	nums := make([]float64, len(values))
	min, max := 0.0, 0.0
	for i, v := range values {
		n := 0.0
		if v != nil {
			n, _ = new(big.Float).SetInt(v).Float64()
		}
		nums[i] = n
		if i == 0 || n < min {
			min = n
		}
		if i == 0 || n > max {
			max = n
		}
	}
	if min == max {
		min, max = min-1, max+1
	}
	innerW, innerH := w-2*pad, h-2*pad
	span := max - min
	var segments strings.Builder
	for i, n := range nums {
		x := pad + float64(i)*innerW/float64(len(nums)-1)
		y := pad + (max-n)*innerH/span
		if i == 0 {
			continue
		}
		color := "#a6a5a5"
		if values[i] != nil && values[i].Sign() > 0 {
			color = "#26954f"
		} else if values[i] != nil && values[i].Sign() < 0 {
			color = "#cf291d"
		}
		previousX := pad + float64(i-1)*innerW/float64(len(nums)-1)
		previousY := pad + (max-nums[i-1])*innerH/span
		fmt.Fprintf(&segments, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s"/>`, previousX, previousY, x, y, color)
	}
	var title strings.Builder
	for i, v := range values {
		if i > 0 {
			title.WriteString(", ")
		}
		title.WriteString(formatDelta(v))
	}
	return fmt.Sprintf(
		`<svg class="spark" viewBox="0 0 %.0f %.0f" width="200" height="24" role="img" aria-label="%s"><title>%s</title><g fill="none" stroke-width="1.5" stroke-linecap="round">%s</g></svg>`,
		w, h, title.String(), title.String(), segments.String(),
	)
}

func formatDelta(v *big.Int) string {
	if v == nil || v.Sign() == 0 {
		return "0"
	}
	sign := "+"
	if v.Sign() < 0 {
		sign = "-"
	}
	return sign + binaryx.FormatBigInt(new(big.Int).Abs(v), polygonscan.USDTDecimal)
}
