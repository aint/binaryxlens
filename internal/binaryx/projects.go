package binaryx

import (
	_ "embed"
	"errors"
	"fmt"
	"math/big"
)

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

	pr.BuildHolders()

	return pr, nil
}
