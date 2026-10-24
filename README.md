# Binaryx Lens

On-chain analytics for tokenized real estate on [binaryx.com](https://binaryx.com).

**Live reports:** https://aint.github.io/binaryxlens/

## Why

Binaryx sells fractional shares of rental and development properties as ERC-20 tokens on Polygon. The platform shows marketing numbers. Binaryx Lens reads token transfers, marketplace fills, and posted rent from the chain and shows what is happening: how fast each property sells, who holds it, what they paid, and whether rental income starts on time.

## What it shows

The index lists every project, with search and filters for construction, rental, and redeemed. Each project is a static report page:

- **Summary** — properties, total supply, tokens bought.
- **Price** — area, Binaryx list price, USDT raised on the initial sale (on-chain), market USD/m² band, and an estimated value tier (underpriced, fair, rich, overpriced). Market ranges and the tier are estimates and can be wrong.
- **Rental status** — expected vs actual start quarter. On time, one quarter late, or later.
- **Posted rent** — monthly USDT from on-chain rent windows, plus **net rental yield** (annualized profit vs raised USDT) for the last quarter, the last 12 months, and the full history. A 12-month history shorter than a year is scaled to a year, and the window is shown. The 12-month yield also shows its change versus the full history, and the quarter yield shows its change versus the 12-month yield. Claims and wallet pushes are not included.
- **Initial sales** — weekly buys and the cumulative curve, with drag-to-zoom.
- **Sell-out ETA** — projected completion from the last week, the last four weeks, and full history. A sold-out property shows the sale period and average buy rate instead.
- **Secondary market** — weekly wallet-to-wallet volume, cumulative volume, and transaction count.
- **Holders** — balance, initial-sale and P2P tokens and USDT, week and month changes, entry and exit dates. Exited holders are listed separately.
- **Tier distribution** — holders grouped by share of supply: 🦐 shrimp ≤0.5%, 🦀 crab ≤1%, 🐟 fish ≤5%, 🐬 dolphin ≤10%, 🦈 shark ≤20%, 🐋 whale >20%.

Reports are available in English and Ukrainian.

`go run .` writes `index.html` and `reports/*.html`. A daily GitHub Action regenerates those files at 00:00 UTC. The property catalog is `internal/properties.go`.

## Screenshots

<!-- TODO: add screenshots to docs/screenshots/ -->

| Index | Project report |
| --- | --- |
| ![Index page](docs/screenshots/index.png) | ![Project report](docs/screenshots/report.png) |

| Initial sales & ETA | Holders & tiers |
| --- | --- |
| ![Weekly buys chart and ETA table](docs/screenshots/sales-eta.png) | ![Holder tier charts](docs/screenshots/holders.png) |

## Run locally

```bash
go run .
```

Local runs use a built-in Etherscan API v2 fallback when no key is set. Override with `POLYGONSCAN_API_KEY` or `-api-key` (Polygon, chain id 137). Prefer env or flag in shared clones so the fallback is not relied on.

Flags:

- `-api-key` — Etherscan API v2 key (overrides `POLYGONSCAN_API_KEY`).
- `-scan-pause` — extra pause between token-transfer pages (default `400ms`). The client also spaces every call.

Open `index.html` in a browser.

## Roadmap

### Holders

- [ ] Wallet pages — click a holder address to see all assets across the platform.
- [ ] Global top holders — leaderboard across all projects.
- [ ] Exited holders — former holdings (tokens and USDT invested), hold duration, exit date.

### Secondary market (P2P)

- [ ] Volume and price trends over time.
- [ ] Summary stats — total transaction count, average transactions per week, this week’s count, trend vs the previous week.

### Signals

- [ ] Insider trades — platform staff wallets (e.g. `0xdf24253c84f16aac030917c2672720b11610e109`).
- [ ] Platform entry and exit — new and fully exited investors.

### Analytics

- [ ] Side-by-side property or project comparison.
- [ ] Sell-out ETA — linear interpolation model in addition to current projections.
- [ ] Developers — track real-estate developers and delivery terms.
- [ ] Actual ARR and ROI per property.
- [ ] Rent consistency — posted rent vs promised amount; flag missed or skipped months.

### Property report

- [ ] Show token count (supply) per property on the report.
- [ ] Explain rent-delay scoring (expected vs actual quarters).
- [ ] Binaryx initial sale price vs current implied price (from latest token pricing).

### UI

- [ ] Tier pie charts — count per tier (e.g. crabs), short tier legend, clearer colors.
- [ ] Screenshots — refresh `docs/screenshots/` (see comment under **Screenshots** above).

## Rental Property Score

Each rental property can receive a score from **1 to 100**. The score combines three inputs: the current value, the trailing net rental yield, and how much of the lease has already run. A 100 means a price well under the market band, a high yield with a full year of history, and a lease that is still in its early years.

The number is a summary of those inputs. The market band is an estimate and can be wrong. The score is not a forecast and it is not financial advice.

<details>
<summary>How the score is calculated</summary>

### When a score is shown

The composite score is shown only when both the price part and the yield part can be computed. Each part is itself a number from 1 to 100. They are combined with fixed weights, rounded to the nearest integer, and clamped to 1–100.

The composite score is not shown when there is no yield yet or yield posted, but fewer than 3 months of it.

### Shared step: linear interpolation

Every part maps a raw input onto 1–100 through a list of knots. Between two knots `(x0, s0)` and `(x1, s1)`:

```
part = s0 + (s1 − s0) × (x − x0) / (x1 − x0)
```

At or beyond the first knot, the part is the first score. At or beyond the last knot, the part is the last score.

### 1. Initial sale price (weight 40%)

`current` is the current price in USDT compared with the middle of the market band, per square metre:

```
marketMid = (marketLow + marketHigh) / 2
delta     = (current / area − marketMid) / marketMid
```

`delta` is negative when buyers paid under the band and positive when they paid over it. The same comparison produces the underpriced / fair / rich / overpriced tier; the score keeps the continuous value instead of the four labels.

| `delta` | Price score |
| --- | --- |
| −35% or below | 100 |
| −30% | 90 |
| −20% | 80 |
| −10% | 60 |
| 0 (at the middle of the band) | 50 |
| +10% | 40 |
| +20% | 20 |
| +30% | 10 |
| +35% or above | 0 |

### 2. Net rental yield (weight 40%)

The yield input is the trailing net rental yield already shown on the report: posted rent profit, annualized, divided by the current USDT value. Use the 12-month figure. All-time yield is not used, because a strong early run can hide a weak recent year.

A history shorter than 12 months is annualized, then pulled toward 8% so a few strong months cannot look as established as a full year:

```
yieldUsed = 8% + (yield − 8%) × min(1, months / 12)
```

At 12 months or more, `yieldUsed` equals `yield`. Under 3 months, the yield part is unavailable.

| `yieldUsed` | Yield score |
| --- | --- |
| 0% or below | 0 |
| 3% | 20 |
| 5% | 40 |
| 8% | 60 |
| 12% | 80 |
| 16% or above | 100 |

### 3. Lease age (weight 20%)

Every property is a 30-year leasehold. Age is the number of years since rent actually started (`RentalStartActual`), the same figure the report shows as age. The first 8 years still have 22 or more years left on the lease, so they all score 100. The score then falls slowly while a long term remains, and faster in the last decade, when a short lease is hard to sell. Anything past year 30 is capped at 30.

| Years since rent started | Years left | Age score |
| --- | --- | --- |
| 0 to 8 | 22 to 30 | 100 |
| 15 | 15 | 80 |
| 22 | 8 | 40 |
| 27 | 3 | 20 |
| 30 or more | 0 | 0 |

### Combining the parts

```
score = round(0.40 × price + 0.40 × yield + 0.20 × age)
```

When age is unknown (no actual rent start date), only price and yield are averaged equally:

```
score = round(0.50 × price + 0.50 × yield)
```

Clamp the result to **1–100**. Tiers on the report: ≥70 is *high*, ≥40 is *mid*, ≥10 is *low*.

</details>

## Disclaimer

Not affiliated with Binaryx. Data comes from public blockchain records and the Binaryx website. Not financial advice.