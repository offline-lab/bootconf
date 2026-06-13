// Package output provides a Unicode box-drawing table renderer for CLI output.
// Tables use ┌─┬─┐ borders with rune-aware column width calculation.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Table renders rows of data as a Unicode box-drawing table. Column widths are
// auto-sized to fit the widest cell (header or data) in each column.
type Table struct {
	headers []string
	rows    [][]string
	out     io.Writer
}

// NewTable creates a Table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		out:     os.Stdout,
	}
}

// AddRow appends a data row. Returns the Table for chaining.
func (t *Table) AddRow(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// Render writes the complete table (header, separator, data rows, bottom border)
// to the configured output writer.
func (t *Table) Render() {
	widths := t.colWidths()
	t.border("┌", "┬", "┐", widths)
	t.data(t.headers, widths)
	t.border("├", "┼", "┤", widths)

	for _, r := range t.rows {
		t.data(r, widths)
	}

	t.border("└", "┴", "┘", widths)
}

func (t *Table) colWidths() []int {
	n := len(t.headers)
	w := make([]int, n)

	for i, h := range t.headers {
		w[i] = utf8.RuneCountInString(h)
	}

	for _, row := range t.rows {
		for i, cell := range row {
			if i < n {
				cw := utf8.RuneCountInString(cell)
				if cw > w[i] {
					w[i] = cw
				}
			}
		}
	}

	return w
}

func (t *Table) border(left, mid, right string, widths []int) {
	var b strings.Builder

	b.WriteString(left)

	for i, w := range widths {
		if i > 0 {
			b.WriteString(mid)
		}
		b.WriteString(strings.Repeat("─", w+2))
	}

	b.WriteString(right)

	_, _ = fmt.Fprintln(t.out, b.String())
}

func (t *Table) data(cells []string, widths []int) {
	var b strings.Builder

	b.WriteString("│")

	for i, w := range widths {
		cell := ""

		if i < len(cells) {
			cell = cells[i]
		}

		pad := w - utf8.RuneCountInString(cell)
		b.WriteString(" ")
		b.WriteString(cell)

		if pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}

		b.WriteString(" │")
	}
	_, _ = fmt.Fprintln(t.out, b.String())
}
