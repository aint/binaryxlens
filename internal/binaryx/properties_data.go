package binaryx

import (
	"math/big"
	"time"
)

var AllProperties = map[string][]*Property{
	"AWWA Hotel by Ribas": {
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas B14",
				Address:             "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas B22",
				Address:             "0xe725a80f426a7d7f5734ba69ccec507251109d09",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "AWWA Hotel by Ribas A16",
				Address:             "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              33,
				Raised:              157_500,
				MarketLow:           2_800,
				MarketHigh:          3_800,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			},
		),
	},
	"Aurora Villas (Bali Balance Ocean Villas)": {
		newProperty(
			propertyParams{
				Name:                "Villa Aurora 1 (Ocean Villa 3)",
				Address:             "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
				State:               Rental,
				Location:            Bali, // Uluwatu
				AreaM2:              165,
				Raised:              380_000,
				Projected:           428_500,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Villa Aurora 2 (Ocean Villa 4)",
				Address:             "0x17236ed296fbd00d3dfa016879833776dd207fd6",
				State:               Rental,
				Location:            Bali, // Uluwatu
				AreaM2:              165,
				Raised:              380_000,
				Projected:           428_500,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Bingin Magic Story Villas": {
		newProperty(
			propertyParams{
				Name:                "Bingin Magic Story Villa 3",
				Address:             "0xe5f846592a58bcfce912bc6fc594649397b6f519",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              115,
				Raised:              249_000,
				Projected:           280_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Bubbles Boutique Complex": {
		newProperty(
			propertyParams{
				Name:                "Bubbles Boutique Complex",
				Address:             "0xC1EA0Ccd94F17Ec0580DD57A34C2B521360ad4b1",
				State:               Rental,
				Location:            Bali, //Canggu
				AreaM2:              555,
				Raised:              1_250_000,
				MarketLow:           2_100,
				MarketHigh:          2_700,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
				RentalProfitMonths: []RentalProfitMonth{
					{Month: time.Date(2024, time.August, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(1828000000)},
					{Month: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2968000000)},
					{Month: time.Date(2024, time.October, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2290000000)},
					{Month: time.Date(2024, time.November, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(2882000000)},
					{Month: time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(6870000000)},
					{Month: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(6870000000)},
					{Month: time.Date(2025, time.February, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(8319000000)},
					{Month: time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC), USDT: big.NewInt(11298000000)},
				},
			},
		),
	},
	"CASCADE Villas": {
		newProperty(
			propertyParams{
				Name:                "CASCADE Villa 2",
				Address:             "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              176,
				Raised:              265_000,
				Projected:           300_000,
				MarketLow:           1_700,
				MarketHigh:          2_400,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
			},
		),
		newProperty(
			propertyParams{
				Name:                "CASCADE Villa 3",
				Address:             "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              176,
				Raised:              265_000,
				Projected:           300_000,
				MarketLow:           1_700,
				MarketHigh:          2_400,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 1},
			},
		),
	},
	"CEMAGI Units": {
		newProperty(
			propertyParams{
				Name:                "CEMAGI Unit 3.44",
				Address:             "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
				State:               Construction,
				Location:            Bali, // Canggu
				AreaM2:              26,
				Raised:              112_000,
				Projected:           130_000,
				MarketLow:           3_000,
				MarketHigh:          4_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "CEMAGI Unit 3.46",
				Address:             "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
				State:               Construction,
				Location:            Bali, // Canggu
				AreaM2:              26,
				Raised:              112_000,
				Projected:           130_000,
				MarketLow:           3_000,
				MarketHigh:          4_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 2},
			},
		),
	},
	"Dukley Glamping 1": {
		newProperty(
			propertyParams{
				Name:                "Dukley Glamping 1",
				Address:             "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
				State:               Rental,
				Location:            Montenegro, // Kolasin
				AreaM2:              240,
				Raised:              940_000,
				Projected:           1_050_000,
				MarketLow:           2_300,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
	},
	"Mountain Retreat by Dukley": {
		newProperty(
			propertyParams{
				Name:                "Mountain Retreat by Dukley",
				Address:             "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
				State:               Redeemed,
				Location:            Montenegro, // Kolasin
				AreaM2:              73,
				Raised:              385_000,
				Projected:           440_000,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
			},
		),
	},
	"Ecoverse Suites": {
		newProperty(
			propertyParams{
				Name:                "Ecoverse Suite",
				Address:             "0x30ed65e470be4f351abf5311769505e3f977deca",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              41,
				Raised:              178_500,
				Projected:           214_200,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 3},
			},
		),
	},
	"Hayat Green Tower": {
		newProperty(
			propertyParams{
				Name:                "Hayat Green Tower",
				Address:             "0xF9d43a9F7Fc7ee7ac47fE96De20e545243450b27",
				State:               Rental,
				Location:            Turkey, // Avsallar, Alanya
				AreaM2:              105,
				Raised:              305_000,
				MarketLow:           1_000,
				MarketHigh:          1_300,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			},
		),
	},
	"Kammara Loft": {
		newProperty(
			propertyParams{
				Name:                "Kammara Loft",
				Address:             "0xB1B987FF1F317A47876185dE4dE9C430823Ad8c5",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              70,
				Raised:              200_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 4},
			},
		),
	},
	"Kammora Living": {
		newProperty(
			propertyParams{
				Name:                "Kammora Living",
				Address:             "0x8389AcD0e05990eF0e087A3BCed3E9C5443d0455",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              49,
				Raised:              150_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2024, Quarter: 3},
			},
		),
	},
	"Karra Loft": {
		newProperty(
			propertyParams{
				Name:                "Karra Loft 3A",
				Address:             "0x27Ceb34AC7545F78A97C0500465aCE9fA10570af",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              68,
				Raised:              220_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Karra Loft 5",
				Address:             "0x2d02E704174635F5E88E17995C3a5E29f283C033",
				State:               Rental,
				Location:            Bali, // Canggu
				AreaM2:              70,
				Raised:              220_000,
				MarketLow:           2_200,
				MarketHigh:          3_000,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 3},
			},
		),
	},
	"La Casa Española Villas": {
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 4",
				Address:             "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              166,  // binaryx website: 230
				Raised:              385_000,
				Projected:           460_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 6",
				Address:             "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              161,  // binaryx: 186
				Raised:              350_000,
				Projected:           425_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 8",
				Address:             "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              171,  // binaryx: 211
				Raised:              350_000,
				Projected:           425_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "La Casa Española Villa 9",
				Address:             "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
				State:               Construction,
				Location:            Bali, // Bingin, Uluwatu
				AreaM2:              200,  // binaryx: 241
				Raised:              385_000,
				Projected:           460_000,
				MarketLow:           2_100,
				MarketHigh:          3_200,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 4},
			},
		),
	},
	"Oasis Royal Collection": {
		newProperty(
			propertyParams{
				Name:                "Oasis Royal Collection 11a",
				Address:             "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
				State:               Rental,
				Location:            Bali, //Pererenan, north of Canggu
				AreaM2:              40,
				Raised:              120_000,
				Projected:           144_000,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Oasis Royal Collection 18b",
				Address:             "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
				State:               Rental,
				Location:            Bali, //Pererenan, north of Canggu
				AreaM2:              40,
				Raised:              120_000,
				Projected:           144_000,
				MarketLow:           2_500,
				MarketHigh:          3_500,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 4},
			},
		),
	},
	"Onyx PARQ Resort 61": {
		newProperty(
			propertyParams{
				Name:                "Onyx PARQ Resort 61",
				Address:             "0xA07DB641FC95067a2Fe68b6224a9dD39564bFd57",
				State:               Rental,
				Location:            Bali, // Parq, Ubud
				AreaM2:              64,
				Raised:              171_612,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2023, Quarter: 2},
			},
		),
	},
	"Roots Villas": {
		newProperty(
			propertyParams{
				Name:                "Roots Villa 1",
				Address:             "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              161,
				Raised:              303_000,
				Projected:           353_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 3",
				Address:             "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 4",
				Address:             "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Roots Villa 5",
				Address:             "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
				State:               Construction,
				Location:            Bali, // Ubud
				AreaM2:              188,
				Raised:              350_000,
				Projected:           405_000,
				MarketLow:           1_800,
				MarketHigh:          2_600,
				RentalStartExpected: YearQuarter{Year: 2026, Quarter: 3},
			},
		),
	},
	"Taryan Dragon Jungle View": {
		newProperty(
			propertyParams{
				Name:       "Taryan Dragon Jungle View",
				Address:    "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
				State:      Construction,
				Location:   Bali, // Canggu
				AreaM2:     55,
				Projected:  340_000,
				MarketLow:  3_500,
				MarketHigh: 5_000,
			},
		),
	},
	"Tropical Loft Villas": {
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 2",
				Address:             "0x4b17845F255cC19dB2612ab8577Ea1e0852BBBd7",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 1},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 3",
				Address:             "0x56467B7E0cF2116A1C7664eE60db77ED24709293",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			},
		),
		newProperty(
			propertyParams{
				Name:                "Tropical Loft Villa 4",
				Address:             "0x6a6F5681678Cc599d4d4Ae55270070406561DCa7",
				State:               Rental,
				Location:            Bali, //Padonan, Canggu
				AreaM2:              126,
				Raised:              225_750,
				MarketLow:           1_850,
				MarketHigh:          2_300,
				RentalStartExpected: YearQuarter{Year: 2025, Quarter: 2},
			},
		),
	},
	"Vesna Townhouse": {
		newProperty(
			propertyParams{
				Name:                "Vesna Townhouse",
				Address:             "0x09558935e9cA1c4985F96163b62Fd850616F6e33",
				State:               Rental,
				Location:            Bali, // Berawa area of Canggu
				AreaM2:              64,
				Raised:              220_000,
				MarketLow:           1_900,
				MarketHigh:          2_500,
				RentalStartExpected: YearQuarter{Year: 2024, Quarter: 2},
			},
		),
	},
}
