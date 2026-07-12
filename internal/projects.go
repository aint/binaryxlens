package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const projectReportPath = "%s_report.html"

//go:embed project_report.html
var projectReport []byte
var projectReportDataPlaceholder = []byte("__PROJECT_DATA_JSON__")

// Project is a group of properties that are related to each other.
type Project struct {
	Name           string
	Properties     []*Property
	Holders        []ProjectHolder
	TotalSupplyRaw *big.Int
	BoughtRaw      *big.Int
	Decimal        uint8
}

func NewProject(name string, properties []*Property) (*Project, error) {
	if len(properties) == 0 {
		return nil, errors.New("project needs at least one property")
	}

	pr := &Project{
		Name:           name,
		Properties:     properties,
		TotalSupplyRaw: big.NewInt(0),
		BoughtRaw:      big.NewInt(0),
		Decimal:        properties[0].Decimal, // all properties expected to have the same decimal
	}

	for _, property := range properties {
		if property.Decimal != pr.Decimal {
			return nil, fmt.Errorf("property %s has unexpected decimal %d", property.Name, property.Decimal)
		}

		pr.TotalSupplyRaw.Add(pr.TotalSupplyRaw, property.TotalSupplyRaw)
		pr.BoughtRaw.Add(pr.BoughtRaw, property.BoughtRaw)
	}

	pr.buildHolders()

	return pr, nil
}

func (pr *Project) GenerateReport(topHolders int) error {
	var payloads []propertyTimeseriesChartPayload
	for _, property := range pr.Properties {
		payloads = append(payloads, buildPropertyTimeSeriesChartPayload(property))
	}

	holders, tierStats := pr.buildHoldersPayload()
	env := projectEnvelope{
		Name:        pr.Name,
		GeneratedAt: time.Now().UTC().Format(timeDateOnly),
		Summary:     pr.buildSummary(),
		Properties:  payloads,
		Holders:     holders,
		TierStats:   tierStats,
		HoldersTop:  topHolders,
	}
	jsonBytes, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if !bytes.Contains(projectReport, projectReportDataPlaceholder) {
		return fmt.Errorf("project template missing placeholder")
	}
	reportPath := fmt.Sprintf(projectReportPath, strings.ReplaceAll(strings.ToLower(pr.Name), " ", "_"))
	out := bytes.ReplaceAll(projectReport, projectReportDataPlaceholder, jsonBytes)
	if err := os.WriteFile(reportPath, out, 0o644); err != nil {
		return err
	}
	fmt.Println("Project report is ready at", reportPath)

	return nil
}

func (pr *Project) buildSummary() projectSummaryPayload {
	return projectSummaryPayload{
		PropertyCount: len(pr.Properties),
		TotalSupply:   FormatBigInt(pr.TotalSupplyRaw, pr.Decimal),
		Bought:        FormatBigInt(pr.BoughtRaw, pr.Decimal),
		BoughtPct:     PercentFloat(pr.BoughtRaw, pr.TotalSupplyRaw),
	}
}

func buildPropertyTimeSeriesChartPayload(property *Property) propertyTimeseriesChartPayload {
	payload := propertyTimeseriesChartPayload{
		Labels:     make([]string, 0, len(property.DailyPoints)),
		Daily:      make([]float64, 0, len(property.DailyPoints)),
		Cumulative: make([]float64, 0, len(property.DailyPoints)),
		Title:      fmt.Sprintf("Daily buys — %s", property.Name),
		ETAs:       make([]propertyETA, 0, len(property.ETAs)),
	}
	for _, p := range property.DailyPoints {
		payload.Labels = append(payload.Labels, p.Day.UTC().Format(timeDateOnly))
		payload.Daily = append(payload.Daily, bigIntToFloat(p.Value, property.Decimal))
		payload.Cumulative = append(payload.Cumulative, bigIntToFloat(p.CumValue, property.Decimal))
	}
	for _, e := range property.ETAs {
		payload.ETAs = append(payload.ETAs, propertyETA{
			Window: e.Window,
			Rate:   e.Rate,
			Days:   e.Days,
			Date:   e.Time.UTC().Format(time.DateOnly),
		})
	}
	return payload
}

type projectEnvelope struct {
	Name        string                           `json:"name"`
	GeneratedAt string                           `json:"generatedAt"`
	Summary     projectSummaryPayload            `json:"summary"`
	Properties  []propertyTimeseriesChartPayload `json:"properties"`
	Holders     []projectHolderPayload           `json:"holders"`
	TierStats   []tierStatPayload                `json:"tierStats"`
	HoldersTop  int                              `json:"holdersTop"`
}

type projectSummaryPayload struct {
	PropertyCount int     `json:"propertyCount"`
	TotalSupply   string  `json:"totalSupply"`
	Bought        string  `json:"bought"`
	BoughtPct     float64 `json:"boughtPct"`
}

type propertyTimeseriesChartPayload struct {
	Title      string        `json:"title"`
	Labels     []string      `json:"labels"`
	Daily      []float64     `json:"daily"`
	Cumulative []float64     `json:"cumulative"`
	ETAs       []propertyETA `json:"etas"`
}

type propertyETA struct {
	Window string `json:"window"`
	Rate   string `json:"rate"`
	Days   int64  `json:"days"`
	Date   string `json:"date"`
}

type projectHolderPayload struct {
	Address       string   `json:"address"`
	PropertyNames []string `json:"propertyNames"`
	Balance       string   `json:"balance"`
	SupplyPct     float64  `json:"supplyPct"`
	Tier          string   `json:"tier"`
}

type tierStatPayload struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	HoldersPct float64 `json:"holdersPct"`
	SupplyPct  float64 `json:"supplyPct"`
}
