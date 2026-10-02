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
	indexUpdatedPlaceholder = []byte("__REPORTS_UPDATED__")
)

const reportsUpdatedLayout = "2006-01-02 15:04"

type projectRow struct {
	Href   string
	Name   string
	Props  string
	Labels []string
}

func WriteIndex(path string, updated time.Time) error {
	stamp := updated.UTC().Format(reportsUpdatedLayout)
	if !bytes.Contains(indexTemplateSrc, indexNavPlaceholder) || !bytes.Contains(indexTemplateSrc, indexUpdatedPlaceholder) || !bytes.Contains(indexTemplateSrc, i18nPlaceholder) {
		return fmt.Errorf("index template missing placeholder")
	}
	out := bytes.ReplaceAll(withI18n(indexTemplateSrc), indexUpdatedPlaceholder, []byte(stamp))
	out = bytes.ReplaceAll(out, indexNavPlaceholder, []byte(projectLinksHTML(projectRows())))
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
		const projectLinkHTML = `
			<a href="%s">
        		<span class="item"><span class="name">%s</span><span class="props">%s</span></span>
        		<span class="tags">%s</span>
    		</a>`
		fmt.Fprintf(&b, projectLinkHTML,
			html.EscapeString(project.Href),
			html.EscapeString(project.Name),
			html.EscapeString(project.Props),
			labels.String(),
		)
	}
	return b.String()
}

func projectRows() []projectRow {
	names := make([]string, 0, len(AllProperties))
	for name := range AllProperties {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		li, lj := strings.ToLower(names[i]), strings.ToLower(names[j])
		if li == lj {
			return names[i] < names[j]
		}
		return li < lj
	})

	projects := make([]projectRow, 0, len(names))
	for _, name := range names {
		properties := AllProperties[name]
		projects = append(projects, projectRow{
			Href:   "reports/" + projectReportFile(name),
			Name:   name,
			Props:  propertyNames(name, properties),
			Labels: propertyTypeLabels(properties),
		})
	}
	return projects
}

func projectReportFile(name string) string {
	slug := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	return slug + "_report.html"
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
