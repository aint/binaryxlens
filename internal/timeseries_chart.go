package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

// WriteDailySeriesHTML writes a single HTML file with embedded Chart.js (CDN) and
// daily + cumulative series from buildDailySeries (human token units per decimals).
func WriteDailySeriesHTML(path string, txs []polygonscan.TokenTransfer, tokenAddr string, decimals uint8) {
	series, err := buildDailySeries(txs, tokenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build timeline: %v\n", err)
		return
	}

	payload := chartPayload{
		Labels:     make([]string, 0, len(series)),
		Daily:      make([]float64, 0, len(series)),
		Cumulative: make([]float64, 0, len(series)),
		Title:      fmt.Sprintf("Daily buys (from token) — %s", tokenAddr),
	}
	cum := big.NewInt(0)
	for _, p := range series {
		payload.Labels = append(payload.Labels, p.Day.UTC().Format(timeDateOnly))
		payload.Daily = append(payload.Daily, rawToHumanFloat(p.Value, decimals))
		cum = new(big.Int).Add(cum, p.Value)
		payload.Cumulative = append(payload.Cumulative, rawToHumanFloat(cum, decimals))
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal chart data: %v\n", err)
		return
	}

	var buf bytes.Buffer
	buf.WriteString(htmlChartPrefix)
	buf.Write(jsonBytes)
	buf.WriteString(htmlChartSuffix)
	err = os.WriteFile(path, buf.Bytes(), 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write file: %v\n", err)
		return
	}
	fmt.Printf("wrote daily series HTML: %s\n", path)
}

type chartPayload struct {
	Labels      []string  `json:"labels"`
	Daily       []float64 `json:"daily"`
	Cumulative  []float64 `json:"cumulative"`
	Title       string    `json:"title"`
}

func rawToHumanFloat(raw *big.Int, decimals uint8) float64 {
	if raw == nil || raw.Sign() == 0 {
		return 0
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	r := new(big.Rat).SetFrac(new(big.Int).Set(raw), denom)
	f, _ := r.Float64()
	return f
}

const htmlChartPrefix = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Daily series</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.6/dist/chart.umd.min.js"></script>
<style>
body { font-family: system-ui, sans-serif; margin: 16px; }
h1 { font-size: 1.1rem; font-weight: 600; }
.chart-wrap {
	position: relative;
	width: 100%;
	height: 420px;
	max-width: 100%;
}
.chart-wrap canvas { display: block; max-width: 100%; }
</style>
</head>
<body>
<h1 id="title"></h1>
<div class="chart-wrap"><canvas id="c"></canvas></div>
<script>
const chartData = `

const htmlChartSuffix = `;
document.getElementById("title").textContent = chartData.title || "Daily series";
const ctx = document.getElementById("c");
new Chart(ctx, {
  type: "line",
  data: {
    labels: chartData.labels,
    datasets: [
      {
        label: "Daily Δ (tokens)",
        data: chartData.daily,
        borderColor: "rgb(75, 192, 192)",
        backgroundColor: "rgba(75, 192, 192, 0.2)",
        tension: 0,
        stepped: "before",
        fill: false,
        yAxisID: "y"
      },
      {
        label: "Cumulative (tokens)",
        data: chartData.cumulative,
        borderColor: "rgb(255, 99, 132)",
        tension: 0.05,
        fill: false,
        yAxisID: "y1"
      }
    ]
  },
  options: {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: "index", intersect: false },
    scales: {
      x: { ticks: { maxTicksLimit: 14 } },
      y: {
        type: "linear",
        position: "left",
        beginAtZero: true,
        title: { display: true, text: "Daily Δ" }
      },
      y1: {
        type: "linear",
        position: "right",
        beginAtZero: true,
        grid: { drawOnChartArea: false },
        title: { display: true, text: "Cumulative" }
      }
    }
  }
});
</script>
</body>
</html>
`