package internal

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
	"time"
)

//go:embed index.html
var indexTemplateSrc []byte

var (
	indexNavPlaceholder     = []byte("__INDEX_NAV__")
	indexUpdatedPlaceholder = []byte("__UPDATED__")
)

const reportsUpdatedLayout = "2006-01-02 15:04"

type projectRow struct {
	Href      string
	Name      string
	Flag      string
	Props     string
	Labels    []string
	Score     int
	ScoreTier string
}

func WriteIndex(path string, updated time.Time, projects []*Project) error {
	stamp := updated.UTC().Format(reportsUpdatedLayout)
	if !bytes.Contains(indexTemplateSrc, indexNavPlaceholder) || !bytes.Contains(indexTemplateSrc, indexUpdatedPlaceholder) || !bytes.Contains(indexTemplateSrc, i18nPlaceholder) {
		return fmt.Errorf("index template missing placeholder")
	}
	out := bytes.ReplaceAll(withI18n(indexTemplateSrc), indexUpdatedPlaceholder, []byte(stamp))
	out = bytes.ReplaceAll(out, indexNavPlaceholder, []byte(projectLinksHTML(projectRows(projects))))
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	fmt.Println("Index is ready at", path)
	return nil
}

func projectLinksHTML(projects []projectRow) string {
	var b strings.Builder
	for _, project := range projects {
		var labels strings.Builder
		for _, label := range project.Labels {
			fmt.Fprintf(&labels, `<span class="tag tag-%[1]s" data-i18n="type.%[1]s">%[1]s</span>`, html.EscapeString(label))
		}
		rating := ""
		if project.Score != -1 {
			rating = fmt.Sprintf(`<span class="rating rating-%s" data-i18n-title="score.note">%d/100</span>`,
				html.EscapeString(project.ScoreTier),
				project.Score,
			)
		}
		const projectLinkHTML = `
			<a href="%s">
				<span class="item"><span class="name">%s %s</span><span class="props">%s</span></span>
				<span class="tags">%s%s</span>
			</a>`
		fmt.Fprintf(&b, projectLinkHTML,
			html.EscapeString(project.Href),
			html.EscapeString(project.Flag),
			html.EscapeString(project.Name),
			html.EscapeString(project.Props),
			rating,
			labels.String(),
		)
	}
	return b.String()
}

func projectRows(projects []*Project) []projectRow {
	projectByName := make(map[string]*Project, len(projects))
	for _, project := range projects {
		projectByName[project.Name] = project
	}

	names := make([]string, 0, len(projectByName))
	for name := range projectByName {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		li, lj := strings.ToLower(names[i]), strings.ToLower(names[j])
		if li == lj {
			return names[i] < names[j]
		}
		return li < lj
	})

	projectRows := make([]projectRow, 0, len(names))
	for _, name := range names {
		project := projectByName[name]
		properties := project.Properties
		score := project.score()
		projectRow := projectRow{
			Name:      name,
			Href:      "reports/" + projectReportFile(name),
			Flag:      projectCountryFlag(properties),
			Props:     propertyNames(name, properties),
			Labels:    propertyTypeLabels(properties),
			Score:     score,
			ScoreTier: scoreTier(score),
		}
		projectRows = append(projectRows, projectRow)
	}
	return projectRows
}

func projectReportFile(name string) string {
	slug := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	return slug + "_report.html"
}

func projectCountryFlag(properties []*Property) string {
	if len(properties) == 0 {
		return ""
	}
	return properties[0].Location.Flag()
}

func propertyNames(projectName string, properties []*Property) string {
	if len(properties) == 1 && properties[0].Name == projectName {
		return ""
	}
	names := make([]string, len(properties))
	for i, property := range properties {
		names[i] = property.Name
	}
	return strings.Join(names, " · ")
}

func propertyTypeLabels(properties []*Property) []string {
	seen := make(map[PropertyType]bool)
	var labels []string
	for _, property := range properties {
		if seen[property.Type] {
			continue
		}
		seen[property.Type] = true
		labels = append(labels, property.Type.Label())
	}
	return labels
}
