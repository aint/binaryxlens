package internal

import (
	"errors"
	"math/big"
	"fmt"
)

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
