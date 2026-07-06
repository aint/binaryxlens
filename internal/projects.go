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

// Project is a group of tokens that are related to each other.
type Project struct {
	Name           string
	Tokens         []Token
	Holders        []ProjectHolder
	TotalSupplyRaw *big.Int
	BoughtRaw      *big.Int
	Decimal        uint8
}

func NewProject(name string, tokens []Token) (Project, error) {
	if len(tokens) == 0 {
		return Project{}, errors.New("project needs at least one token")
	}

	p := Project{
		Name:           name,
		Tokens:         tokens,
		TotalSupplyRaw: big.NewInt(0),
		BoughtRaw:      big.NewInt(0),
		Decimal:        tokens[0].Decimal, // all tokens expected to have the same decimal
	}

	for _, token := range tokens {
		if token.Decimal != p.Decimal {
			return Project{}, fmt.Errorf("token %s has unexpected decimal %d", token.Name, token.Decimal)
		}

		p.TotalSupplyRaw.Add(p.TotalSupplyRaw, token.TotalSupplyRaw)
		p.BoughtRaw.Add(p.BoughtRaw, token.BoughtRaw)
	}

	p.Holders = p.getHolders()
	return p, nil
}

func (p Project) GenerateReport(topHolders int) error {
	var payloads []tokenTimeseriesChartPayload
	for _, token := range p.Tokens {
		payloads = append(payloads, buildTokenTimeSeriesChartPayload(token))
	}

	holders, tierStats := p.buildHoldersPayload()
	env := projectEnvelope{
		Name:        p.Name,
		GeneratedAt: time.Now().UTC().Format(timeDateOnly),
		Summary:     p.buildSummary(),
		Tokens:      payloads,
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
	reportPath := fmt.Sprintf(projectReportPath, strings.ReplaceAll(strings.ToLower(p.Name), " ", "_"))
	out := bytes.ReplaceAll(projectReport, projectReportDataPlaceholder, jsonBytes)
	if err := os.WriteFile(reportPath, out, 0o644); err != nil {
		return err
	}
	fmt.Println("Project report is ready at", reportPath)

	return nil
}

func (p Project) buildSummary() projectSummaryPayload {
	return projectSummaryPayload{
		TokenCount:  len(p.Tokens),
		TotalSupply: FormatBigInt(p.TotalSupplyRaw, p.Decimal),
		Bought:      FormatBigInt(p.BoughtRaw, p.Decimal),
		BoughtPct:   PercentFloat(p.BoughtRaw, p.TotalSupplyRaw),
	}
}

func buildTokenTimeSeriesChartPayload(token Token) tokenTimeseriesChartPayload {
	payload := tokenTimeseriesChartPayload{
		Labels:     make([]string, 0, len(token.DailyPoints)),
		Daily:      make([]float64, 0, len(token.DailyPoints)),
		Cumulative: make([]float64, 0, len(token.DailyPoints)),
		Title:      fmt.Sprintf("Daily buys — %s", token.Name),
		ETAs:       make([]tokenETA, 0, len(token.ETAs)),
	}
	for _, p := range token.DailyPoints {
		payload.Labels = append(payload.Labels, p.Day.UTC().Format(timeDateOnly))
		payload.Daily = append(payload.Daily, bigIntToFloat(p.Value, token.Decimal))
		payload.Cumulative = append(payload.Cumulative, bigIntToFloat(p.CumValue, token.Decimal))
	}
	for _, e := range token.ETAs {
		payload.ETAs = append(payload.ETAs, tokenETA{
			Window: e.Window,
			Rate:   e.Rate,
			Days:   e.Days,
			Date:   e.Time.UTC().Format(time.DateOnly),
		})
	}
	return payload
}

type projectEnvelope struct {
	Name        string                        `json:"name"`
	GeneratedAt string                        `json:"generatedAt"`
	Summary     projectSummaryPayload         `json:"summary"`
	Tokens      []tokenTimeseriesChartPayload `json:"tokens"`
	Holders     []projectHolderPayload        `json:"holders"`
	TierStats   []tierStatPayload             `json:"tierStats"`
	HoldersTop  int                           `json:"holdersTop"`
}

type projectSummaryPayload struct {
	TokenCount  int     `json:"tokenCount"`
	TotalSupply string  `json:"totalSupply"`
	Bought      string  `json:"bought"`
	BoughtPct   float64 `json:"boughtPct"`
}

type tokenTimeseriesChartPayload struct {
	Title      string     `json:"title"`
	Labels     []string   `json:"labels"`
	Daily      []float64  `json:"daily"`
	Cumulative []float64  `json:"cumulative"`
	ETAs       []tokenETA `json:"etas"`
}

type tokenETA struct {
	Window string `json:"window"`
	Rate   string `json:"rate"`
	Days   int64  `json:"days"`
	Date   string `json:"date"`
}

type projectHolderPayload struct {
	Address    string   `json:"address"`
	TokenNames []string `json:"tokenNames"`
	Balance    string   `json:"balance"`
	SupplyPct  float64  `json:"supplyPct"`
	Tier       string   `json:"tier"`
}

type tierStatPayload struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	HoldersPct float64 `json:"holdersPct"`
	SupplyPct  float64 `json:"supplyPct"`
}
