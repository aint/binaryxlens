package internal

import (
	_ "embed"
	"errors"
	"fmt"
	"math/big"
	"encoding/json"
	"bytes"
	"os"
	"strings"
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
		Decimal:        tokens[0].Decimal, // all tokens expected to have the same decimal
	}

	for _, token := range tokens {
		if token.Decimal != p.Decimal {
			return Project{}, fmt.Errorf("token %s has unexpected decimal %d", token.Name, token.Decimal)
		}

		p.TotalSupplyRaw.Add(p.TotalSupplyRaw, token.TotalSupplyRaw)
	}

	p.Holders = p.getHolders()
	return p, nil
}

func (p Project) GenerateReport() error {
	var payloads []chartPayload
	for _, token := range p.Tokens {
		payloads = append(payloads, buildChartPayload(token))
	}

	env := projectEnvelope{Name: p.Name, Tokens: payloads}
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

type projectEnvelope struct {
	Name   string         `json:"name"`
	Tokens []chartPayload `json:"tokens"`
}