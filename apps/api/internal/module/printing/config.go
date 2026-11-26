package printing

import (
	"slices"
	"strings"

	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Cell struct {
	Label string `json:"label"`
	Field string `json:"field"`
	Text  string `json:"text"`
	Span  int    `json:"span"`
}

type Column struct {
	Field string `json:"field"`
	Label string `json:"label"`
	Width int    `json:"width"`
	Align string `json:"align"`
}

type Configuration struct {
	Title         string   `json:"title"`
	Paper         string   `json:"paper"`
	Orientation   string   `json:"orientation"`
	ColumnsPerRow int      `json:"columnsPerRow"`
	Header        []Cell   `json:"header"`
	Columns       []Column `json:"columns"`
	Footer        []Cell   `json:"footer"`
}

type Definition struct {
	Label   string        `json:"label"`
	Fields  []Field       `json:"fields"`
	Detail  []Field       `json:"detail"`
	Default Configuration `json:"default"`
}

func (c Configuration) validate(d Definition) error {
	if strings.TrimSpace(c.Title) == "" || len([]rune(c.Title)) > 80 ||
		(c.Paper != "A4" && c.Paper != "A5") ||
		(c.Orientation != "portrait" && c.Orientation != "landscape") ||
		c.ColumnsPerRow < 1 || c.ColumnsPerRow > 4 {
		return store.Invalid("invalid print title, paper, orientation or columns per row")
	}
	for _, cells := range [][]Cell{c.Header, c.Footer} {
		if len(cells) > 24 {
			return store.Invalid("a print header or footer can have at most 24 cells")
		}
		for _, cell := range cells {
			if len([]rune(cell.Label)) > 80 || len([]rune(cell.Text)) > 500 ||
				cell.Span < 1 || cell.Span > c.ColumnsPerRow ||
				(cell.Field == "" && strings.TrimSpace(cell.Text) == "" && strings.TrimSpace(cell.Label) == "") ||
				(cell.Field != "" && (cell.Text != "" || !hasField(d.Fields, cell.Field))) {
				return store.Invalid("invalid print header or footer cell")
			}
		}
	}
	if len(c.Columns) < 1 || len(c.Columns) > 12 {
		return store.Invalid("a print table requires 1 to 12 columns")
	}
	for _, column := range c.Columns {
		if !hasField(d.Detail, column.Field) || strings.TrimSpace(column.Label) == "" ||
			len([]rune(column.Label)) > 80 || column.Width < 1 || column.Width > 100 ||
			!slices.Contains([]string{"left", "center", "right"}, column.Align) {
			return store.Invalid("invalid print table column")
		}
	}
	return nil
}

func hasField(fields []Field, key string) bool {
	return slices.ContainsFunc(fields, func(field Field) bool { return field.Key == key })
}

// Standard builds the built-in layout from the provider's field dictionary.
func Standard(label string, fields, detail []Field, footer []Cell) Definition {
	configuration := Configuration{
		Title: label, Paper: "A4", Orientation: "portrait", ColumnsPerRow: 2,
		Footer: footer,
	}
	for _, field := range fields {
		if !slices.ContainsFunc(footer, func(cell Cell) bool { return cell.Field == field.Key }) {
			configuration.Header = append(configuration.Header, Cell{Label: field.Label, Field: field.Key, Span: 1})
		}
	}
	for _, field := range detail {
		align := "left"
		if field.Key == "quantity" || field.Key == "amount" {
			align = "right"
		}
		configuration.Columns = append(configuration.Columns, Column{
			Field: field.Key, Label: field.Label, Width: 1, Align: align,
		})
	}
	return Definition{Label: label, Fields: fields, Detail: detail, Default: configuration}
}
