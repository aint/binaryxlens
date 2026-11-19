package site

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
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
var propertyYieldTrendsPlaceholder = []byte("__PROPERTY_YIELD_TRENDS__")

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
	payload := []byte(holderTierTrendsHTML(projects))
	template := bytes.ReplaceAll(trendsTemplateSrc, propertyYieldTrendsPlaceholder, []byte(propertyYieldTrendsHTML(projects)))
	return write(path, updated, template, trendsPlaceholder, payload, headerConfig{activePage: "trends"})
}

func holderTierTrendsHTML(projects []*binaryx.Project) string {
	var b strings.Builder
	for _, t := range holderTierTrends(globalHolders(projects)) {
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
			spark = holderSparkline(series)
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

func holderTierTrends(holders []globalHolder) []trend {
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
	trendWindowCount = 6
	trendWindowDays  = 30
)

func activityUSDTSeries(activity binaryx.DailyActivity) []*big.Int {
	series := make([]*big.Int, trendWindowCount)
	for i := range series {
		window := trendWindowCount - i - 1
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

func holderSparkline(values []*big.Int) string {
	if len(values) < 2 {
		return ""
	}
	nums := make([]float64, len(values))
	var title strings.Builder
	for i, v := range values {
		if i > 0 {
			title.WriteString(", ")
		}
		n := float64(0)
		if v != nil {
			n, _ = new(big.Float).SetInt(v).Float64()
		}
		nums[i] = n
		title.WriteString(formatDelta(v))
	}
	return renderSparkline(nums, title.String(), func(i int) string {
		if values[i] != nil && values[i].Sign() > 0 {
			return "#26954f"
		} else if values[i] != nil && values[i].Sign() < 0 {
			return "#cf291d"
		}
		return "#a6a5a5"
	})
}

const propertyYieldTrendMonths = 6

func propertyYieldTrendsHTML(projects []*binaryx.Project) string {
	var rows strings.Builder
	for _, project := range projects {
		for _, property := range project.Properties {
			value := property.CurrentValue.Price
			profitMonths := property.RentalProfitMonths
			quarterYield, _, ok := binaryx.TrailingNetRentalYield(profitMonths, value, binaryx.YieldTrailingMonths, false)
			if !ok {
				continue
			}

			yields := make([]float64, 0, len(profitMonths))
			labels := make([]string, 0, len(profitMonths))
			for _, month := range profitMonths {
				yield, ok := binaryx.AllTimeNetRentalYield([]binaryx.RentalProfitMonth{month}, value)
				if !ok {
					continue
				}
				yields = append(yields, yield)
				labels = append(labels, month.Month.UTC().Format("2006-01"))
			}
			if len(yields) > propertyYieldTrendMonths {
				yields = yields[len(yields)-propertyYieldTrendMonths:]
				labels = labels[len(labels)-propertyYieldTrendMonths:]
			}
			if len(yields) == 0 {
				continue
			}
			fmt.Fprintf(
				&rows,
				`<div class="trend-row">
					<span class="item">
						<span class="name">%s</span>
						<span class="props">%s</span>
					</span>
					<span class="trend-metrics">%s<span class="delta flat">%.2f%%</span></span>
				</div>`,
				html.EscapeString(property.Name), html.EscapeString(project.Name), yieldSparkline(yields, labels), quarterYield,
			)
		}
	}
	if rows.Len() == 0 {
		return ""
	}
	return `<section class="yield-trends">
				<h2>Property rental yield</h2>
				<p class="trends-note">Net rental yield for past 6 months as a percentage of each property’s current value.</p>
				<nav>
					<div class="trend-head">
						<span>Property</span>
						<span>3 months net rental yield</span>
					</div>` + rows.String() +
		`</nav>
			</section>`
}

func yieldSparkline(values []float64, labels []string) string {
	if len(values) < 2 {
		return ""
	}
	var title strings.Builder
	for i, value := range values {
		if i > 0 {
			title.WriteString(", ")
		}
		fmt.Fprintf(&title, "%s: %.2f%%", html.EscapeString(labels[i]), value)
	}
	return renderSparkline(values, title.String(), func(i int) string {
		if values[i] > values[i-1] {
			return "#26954f"
		} else if values[i] < values[i-1] {
			return "#cf291d"
		}
		return "#a6a5a5"
	})
}

func renderSparkline(values []float64, title string, segmentColor func(int) string) string {
	const w, h, pad = 200.0, 24.0, 2.0
	min, max := values[0], values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	if min == max {
		min, max = min-1, max+1
	}
	innerW, innerH := w-2*pad, h-2*pad
	span := max - min
	var segments strings.Builder
	for i := 1; i < len(values); i++ {
		x1 := pad + float64(i-1)*innerW/float64(len(values)-1)
		y1 := pad + (max-values[i-1])*innerH/span
		x2 := pad + float64(i)*innerW/float64(len(values)-1)
		y2 := pad + (max-values[i])*innerH/span
		fmt.Fprintf(&segments, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s"/>`, x1, y1, x2, y2, segmentColor(i))
	}
	return fmt.Sprintf(
		`<svg class="spark" viewBox="0 0 %.0f %.0f" width="200" height="24" role="img" aria-label="%s">
			<title>%s</title>
			<g fill="none" stroke-width="1.5" stroke-linecap="round">%s</g>
		</svg>`,
		w, h, title, title, segments.String(),
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
