package site

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/aint/binaryxlens/internal/binaryx"
	"github.com/aint/binaryxlens/internal/polygonscan"
)

const (
	reportsDir = "reports"
)

//go:embed project.html
var projectReport []byte
var projectReportDataPlaceholder = []byte("__PROJECT_DATA_JSON__")

func GenerateReport(pr *binaryx.Project) error {
	var payloads []propertyReportPayload
	for _, property := range pr.Properties {
		payload, err := buildPropertyReportPayload(property)
		if err != nil {
			return fmt.Errorf("build property payload %s: %w", property.Name, err)
		}
		payloads = append(payloads, payload)
	}

	holders, tierStats := buildHoldersPayload(pr)
	env := projectEnvelope{
		Name:        pr.Name,
		GeneratedAt: time.Now().UTC().Format(binaryx.TimeDateOnly),
		Summary:     buildSummary(pr),
		Properties:  payloads,
		Holders:     holders,
		TierStats:   tierStats,
	}
	jsonBytes, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if !bytes.Contains(projectReport, projectReportDataPlaceholder) || !bytes.Contains(projectReport, i18nPlaceholder) {
		return fmt.Errorf("project template missing placeholder")
	}
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return fmt.Errorf("create reports dir: %w", err)
	}
	reportPath := filepath.Join(reportsDir, projectReportFile(pr.Name))
	out := bytes.ReplaceAll(withI18n(projectReport), projectReportDataPlaceholder, jsonBytes)
	if err := os.WriteFile(reportPath, out, 0o644); err != nil {
		return err
	}
	fmt.Println("Project report is ready at", reportPath)

	return nil
}

func buildSummary(pr *binaryx.Project) projectSummaryPayload {
	score := pr.Score()
	return projectSummaryPayload{
		PropertyCount: len(pr.Properties),
		TotalSupply:   binaryx.FormatBigInt(pr.TotalSupplyRaw, pr.Decimal),
		Bought:        binaryx.FormatBigInt(pr.BoughtRaw, pr.Decimal),
		BoughtPct:     binaryx.PercentFloat(pr.BoughtRaw, pr.TotalSupplyRaw),
		Score:         score,
		ScoreTier:     binaryx.ScoreTier(score),
	}
}

func buildPropertyReportPayload(property *binaryx.Property) (propertyReportPayload, error) {
	p2pWeeklyPoints := property.P2PSaleWeeklyPoints
	initialSaleWeeklyPoints := property.InitialSaleWeeklyPoints
	holders, tierStats := buildPropertyHoldersPayload(property)
	score := property.Score()

	payload := propertyReportPayload{
		Name:      property.Name,
		Address:   property.Address,
		State:     string(property.State),
		Location:  string(property.Location),
		Score:     score,
		ScoreTier: binaryx.ScoreTier(score),
		Holders:   holders,
		TierStats: tierStats,
		Rental:    buildRentalPayload(property, time.Now().UTC()),
		Price: propertyPricePayload{
			AreaM2:             property.AreaM2,
			Current:            property.CurrentValue.Price,
			Raised:             property.RaisedValue.Price,
			Projected:          property.ProjectedValue.Price,
			MarketLow:          property.MarketLowValue,
			MarketHigh:         property.MarketHighValue,
			CurrentValueTier:   string(property.CurrentValue.Status),
			ProjectedValueTier: string(property.ProjectedValue.Status),
			RaisedValueTier:    string(property.RaisedValue.Status),
		},
		Initial: propertyInitialSalesPayload{
			Labels:     make([]string, 0, len(initialSaleWeeklyPoints)),
			Weekly:     make([]float64, 0, len(initialSaleWeeklyPoints)),
			Cumulative: make([]float64, 0, len(initialSaleWeeklyPoints)),
			ETAs:       make([]propertyETA, 0, len(property.ETAs)),
		},
		Secondary: propertyP2PSalesPayload{
			TxCount:    property.P2PTxCount(),
			Labels:     make([]string, 0, len(p2pWeeklyPoints)),
			Weekly:     make([]float64, 0, len(p2pWeeklyPoints)),
			Cumulative: make([]float64, 0, len(p2pWeeklyPoints)),
		},
	}

	for _, p := range initialSaleWeeklyPoints {
		payload.Initial.Labels = append(payload.Initial.Labels, p.Week.UTC().Format(binaryx.TimeDateOnly))
		payload.Initial.Weekly = append(payload.Initial.Weekly, binaryx.BigIntToFloat(p.Value, property.Decimal))
		payload.Initial.Cumulative = append(payload.Initial.Cumulative, binaryx.BigIntToFloat(p.CumValue, property.Decimal))
	}
	for _, e := range property.ETAs {
		payload.Initial.ETAs = append(payload.Initial.ETAs, propertyETA{
			Window: e.Window,
			Rate:   e.Rate,
			Days:   e.Days,
			Date:   e.Time.UTC().Format(time.DateOnly),
		})
	}
	for _, p := range p2pWeeklyPoints {
		payload.Secondary.Labels = append(payload.Secondary.Labels, p.Week.UTC().Format(binaryx.TimeDateOnly))
		payload.Secondary.Weekly = append(payload.Secondary.Weekly, binaryx.BigIntToFloat(p.Value, property.Decimal))
		payload.Secondary.Cumulative = append(payload.Secondary.Cumulative, binaryx.BigIntToFloat(p.CumValue, property.Decimal))
	}
	return payload, nil
}

func buildRentalPayload(property *binaryx.Property, now time.Time) *propertyRentalPayload {
	delayQuarters, ok := property.RentalStartDelay(now)
	if !ok {
		return nil
	}

	var score string
	switch {
	case delayQuarters <= 0:
		score = "green"
	case delayQuarters == 1:
		score = "yellow"
	default:
		score = "red"
	}

	payload := &propertyRentalPayload{
		Expected:      property.RentalStartExpected.String(),
		Score:         score,
		DelayQuarters: delayQuarters,
	}
	if property.RentalStartActual != nil {
		payload.Actual = property.RentalStartActual.String()
		ageQuarters := binaryx.NewYearQuarter(now).QuartersSince(*property.RentalStartActual)
		if ageQuarters >= 0 {
			ageYears := float64(ageQuarters) / 4
			payload.AgeYears = &ageYears
		}
	}
	payload.rentalProfit = rentalProfitSeries(property.RentalProfitMonths, property.CurrentValue.Price)
	return payload
}

func rentalProfitSeries(months []binaryx.RentalProfitMonth, value uint) rentalProfit {
	profit := rentalProfit{}
	if percent, _, ok := binaryx.TrailingNetRentalYield(months, value, 3, false); ok {
		profit.NetYieldQuarter = &percent
	}
	if percent, basis, ok := binaryx.TrailingNetRentalYield(months, value, 12, true); ok {
		profit.NetYieldYear = &percent
		profit.NetYieldYearMonths = basis
	}
	if percent, ok := binaryx.AllTimeNetRentalYield(months, value); ok {
		profit.NetYieldAllTime = &percent
	}
	if profit.NetYieldYear != nil && profit.NetYieldAllTime != nil {
		delta := math.Round((*profit.NetYieldYear-*profit.NetYieldAllTime)*100) / 100
		profit.NetYieldYearTrend = &delta
	}
	if profit.NetYieldQuarter != nil && profit.NetYieldYear != nil {
		delta := math.Round((*profit.NetYieldQuarter-*profit.NetYieldYear)*100) / 100
		profit.NetYieldQuarterTrend = &delta
	}
	if len(months) == 0 {
		return profit
	}
	profit.ProfitAllTime = binaryx.FormatBigInt(binaryx.SumRentUSDT(months), polygonscan.USDTDecimal)
	window := months
	if len(months) > 12 {
		window = months[len(months)-12:]
	}
	profit.ProfitYear = binaryx.FormatBigInt(binaryx.SumRentUSDT(window), polygonscan.USDTDecimal)
	profit.ProfitYearMonths = len(window)
	profit.Labels = make([]string, 0, len(months))
	profit.Monthly = make([]string, 0, len(months))
	for _, month := range months {
		profit.Labels = append(profit.Labels, month.Month.UTC().Format("2006-01"))
		profit.Monthly = append(profit.Monthly, binaryx.FormatBigInt(month.USDT, polygonscan.USDTDecimal))
	}
	return profit
}

func buildPropertyHoldersPayload(property *binaryx.Property) ([]propertyHolderRow, []tierStatPayload) {
	holders := make([]propertyHolderRow, 0, len(property.Holders))
	pcts := make([]float64, 0, len(property.Holders))
	for _, h := range property.Holders {
		pct := binaryx.PercentFloat(h.Balance, property.TotalSupplyRaw)
		tier := ""
		if h.Balance.Sign() > 0 {
			tier = holderTier(pct)
			pcts = append(pcts, pct)
		}
		holders = append(holders, propertyHolderRow{
			Address:          h.Address,
			Balance:          binaryx.FormatBigInt(h.Balance, property.Decimal),
			WeekDeltaTokens:  binaryx.FormatDelta(h.WeekDeltaTokens, property.Decimal),
			MonthDeltaTokens: binaryx.FormatDelta(h.MonthDeltaTokens, property.Decimal),
			WeekDeltaUSDT:    binaryx.FormatDelta(h.WeekDeltaUSDT, polygonscan.USDTDecimal),
			MonthDeltaUSDT:   binaryx.FormatDelta(h.MonthDeltaUSDT, polygonscan.USDTDecimal),
			InitialBought:    binaryx.FormatBigInt(h.InitialBought, property.Decimal),
			P2PBought:        binaryx.FormatBigInt(h.P2PBought, property.Decimal),
			InitSaleUSDT:     binaryx.FormatBigInt(h.InitSaleUSDT, polygonscan.USDTDecimal),
			P2PUSDT:          binaryx.FormatBigInt(h.P2PUSDT, polygonscan.USDTDecimal),
			OffPlatformP2P:   h.OffPlatformP2P,
			SupplyPct:        pct,
			Tier:             tier,
			Entered:          binaryx.FormatDate(h.Entered),
			Exited:           binaryx.FormatDate(h.Exited),
		})
	}
	return holders, buildTierStatsPayload(pcts)
}

func buildHoldersPayload(pr *binaryx.Project) ([]projectHolderPayload, []tierStatPayload) {
	holders := make([]projectHolderPayload, 0, len(pr.Holders))
	pcts := make([]float64, 0, len(pr.Holders))
	for _, h := range pr.Holders {
		pct := binaryx.PercentFloat(h.TotalBalance, pr.TotalSupplyRaw)
		tier := ""
		if h.TotalBalance.Sign() > 0 {
			tier = holderTier(pct)
			pcts = append(pcts, pct)
		}
		holders = append(holders, projectHolderPayload{
			Address:          h.Address,
			PropertyNames:    slices.Sorted(maps.Keys(h.PropertyBalances)),
			Balance:          binaryx.FormatBigInt(h.TotalBalance, pr.Decimal),
			WeekDeltaTokens:  binaryx.FormatDelta(h.WeekDeltaTokens, pr.Decimal),
			MonthDeltaTokens: binaryx.FormatDelta(h.MonthDeltaTokens, pr.Decimal),
			WeekDeltaUSDT:    binaryx.FormatDelta(h.WeekDeltaUSDT, polygonscan.USDTDecimal),
			MonthDeltaUSDT:   binaryx.FormatDelta(h.MonthDeltaUSDT, polygonscan.USDTDecimal),
			InitialBought:    binaryx.FormatBigInt(h.InitialBought, pr.Decimal),
			P2PBought:        binaryx.FormatBigInt(h.P2PBought, pr.Decimal),
			InitSaleUSDT:     binaryx.FormatBigInt(h.InitSaleUSDT, polygonscan.USDTDecimal),
			P2PUSDT:          binaryx.FormatBigInt(h.P2PUSDT, polygonscan.USDTDecimal),
			OffPlatformP2P:   h.OffPlatformP2P,
			SupplyPct:        pct,
			Tier:             tier,
			Entered:          binaryx.FormatDate(h.Entered),
			Exited:           binaryx.FormatDate(h.Exited),
		})
	}
	return holders, buildTierStatsPayload(pcts)
}

func buildTierStatsPayload(pcts []float64) []tierStatPayload {
	total := len(pcts)
	if total == 0 {
		return nil
	}

	type tierStat struct {
		count     int
		supplyPct float64
	}
	stats := make(map[string]tierStat)
	for _, pct := range pcts {
		tier := holderTier(pct)
		s := stats[tier]
		s.count++
		s.supplyPct += pct
		stats[tier] = s
	}

	out := make([]tierStatPayload, 0, len(holderTierThresholds))
	for _, t := range holderTierThresholds {
		s := stats[t.name]
		out = append(out, tierStatPayload{
			Name:       t.name,
			Count:      s.count,
			HoldersPct: float64(s.count) / float64(total) * 100,
			SupplyPct:  s.supplyPct,
		})
	}
	return out
}

func holderTier(percent float64) string {
	for _, t := range holderTierThresholds {
		if percent <= t.max {
			return t.name
		}
	}
	return holderTierThresholds[len(holderTierThresholds)-1].name
}

var holderTierThresholds = []struct {
	max  float64
	name string
}{
	{0.5, "shrimp"},
	{1, "crab"},
	{5, "fish"},
	{10, "dolphin"},
	{20, "shark"},
	{100, "whale"},
}

type projectEnvelope struct {
	Name        string                  `json:"name"`
	GeneratedAt string                  `json:"generatedAt"`
	Summary     projectSummaryPayload   `json:"summary"`
	Properties  []propertyReportPayload `json:"properties"`
	Holders     []projectHolderPayload  `json:"holders"`
	TierStats   []tierStatPayload       `json:"tierStats"`
}

type projectSummaryPayload struct {
	PropertyCount int     `json:"propertyCount"`
	TotalSupply   string  `json:"totalSupply"`
	Bought        string  `json:"bought"`
	BoughtPct     float64 `json:"boughtPct"`
	Score         int     `json:"score"`
	ScoreTier     string  `json:"scoreTier"`
}

type propertyReportPayload struct {
	Name      string                      `json:"name"`
	Address   string                      `json:"address"`
	State     string                      `json:"state"`
	Location  string                      `json:"location"`
	Score     int                         `json:"score"`
	ScoreTier string                      `json:"scoreTier"`
	Price     propertyPricePayload        `json:"price"`
	Rental    *propertyRentalPayload      `json:"rental,omitempty"`
	Holders   []propertyHolderRow         `json:"holders"`
	TierStats []tierStatPayload           `json:"tierStats"`
	Initial   propertyInitialSalesPayload `json:"initial_sales"`
	Secondary propertyP2PSalesPayload     `json:"p2p_sales"`
}

type propertyPricePayload struct {
	AreaM2             uint   `json:"areaM2"`
	Current            uint   `json:"current"`
	Raised             uint   `json:"raised"`
	Projected          uint   `json:"projected"`
	MarketLow          uint   `json:"marketLow"`
	MarketHigh         uint   `json:"marketHigh"`
	CurrentValueTier   string `json:"currentValueTier"`
	ProjectedValueTier string `json:"projectedValueTier"`
	RaisedValueTier    string `json:"raisedValueTier"`
}

type propertyRentalPayload struct {
	Expected      string   `json:"expected"`
	Actual        string   `json:"actual,omitempty"`
	Score         string   `json:"score"`
	DelayQuarters int      `json:"delayQuarters"`
	AgeYears      *float64 `json:"ageYears,omitempty"`
	rentalProfit
}

type rentalProfit struct {
	Labels               []string `json:"labels,omitempty"`
	Monthly              []string `json:"monthly,omitempty"`
	NetYieldQuarter      *float64 `json:"netYieldQuarter,omitempty"`
	NetYieldQuarterTrend *float64 `json:"netYieldQuarterTrend,omitempty"`
	NetYieldYear         *float64 `json:"netYieldYear,omitempty"`
	NetYieldYearMonths   int      `json:"netYieldYearMonths,omitempty"`
	NetYieldYearTrend    *float64 `json:"netYieldYearTrend,omitempty"`
	NetYieldAllTime      *float64 `json:"netYieldAllTime,omitempty"`
	ProfitAllTime        string   `json:"profitAllTime,omitempty"`
	ProfitYear           string   `json:"profitYear,omitempty"`
	ProfitYearMonths     int      `json:"profitYearMonths,omitempty"`
}

type propertyInitialSalesPayload struct {
	Labels     []string      `json:"labels"`
	Weekly     []float64     `json:"weekly"`
	Cumulative []float64     `json:"cumulative"`
	ETAs       []propertyETA `json:"etas"`
}

type propertyP2PSalesPayload struct {
	TxCount    int       `json:"txCount"`
	Labels     []string  `json:"labels"`
	Weekly     []float64 `json:"weekly"`
	Cumulative []float64 `json:"cumulative"`
}

type propertyETA struct {
	Window string `json:"window"`
	Rate   string `json:"rate"`
	Days   int64  `json:"days"`
	Date   string `json:"date,omitempty"`
}

type projectHolderPayload struct {
	Address          string   `json:"address"`
	PropertyNames    []string `json:"propertyNames"`
	Balance          string   `json:"balance"`
	WeekDeltaTokens  string   `json:"weekDeltaTokens"`
	MonthDeltaTokens string   `json:"monthDeltaTokens"`
	WeekDeltaUSDT    string   `json:"weekDeltaUSDT"`
	MonthDeltaUSDT   string   `json:"monthDeltaUSDT"`
	InitialBought    string   `json:"initialBought"`
	P2PBought        string   `json:"p2pBought"`
	InitSaleUSDT     string   `json:"initSaleUSDT"`
	P2PUSDT          string   `json:"p2pUSDT"`
	OffPlatformP2P   bool     `json:"offPlatformP2P,omitempty"`
	SupplyPct        float64  `json:"supplyPct"`
	Tier             string   `json:"tier"`
	Entered          string   `json:"entered,omitempty"`
	Exited           string   `json:"exited,omitempty"`
}

type propertyHolderRow struct {
	Address          string  `json:"address"`
	Balance          string  `json:"balance"`
	WeekDeltaTokens  string  `json:"weekDeltaTokens"`
	MonthDeltaTokens string  `json:"monthDeltaTokens"`
	WeekDeltaUSDT    string  `json:"weekDeltaUSDT"`
	MonthDeltaUSDT   string  `json:"monthDeltaUSDT"`
	InitialBought    string  `json:"initialBought"`
	P2PBought        string  `json:"p2pBought"`
	InitSaleUSDT     string  `json:"initSaleUSDT"`
	P2PUSDT          string  `json:"p2pUSDT"`
	OffPlatformP2P   bool    `json:"offPlatformP2P,omitempty"`
	SupplyPct        float64 `json:"supplyPct"`
	Tier             string  `json:"tier"`
	Entered          string  `json:"entered,omitempty"`
	Exited           string  `json:"exited,omitempty"`
}

type tierStatPayload struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	HoldersPct float64 `json:"holdersPct"`
	SupplyPct  float64 `json:"supplyPct"`
}
