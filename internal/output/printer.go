package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

func ParseFormat(s string) Format {
	if strings.EqualFold(s, "json") {
		return FormatJSON
	}
	return FormatTable
}

// PrintJSON prints raw JSON data (already a json.RawMessage from the API).
func PrintJSON(data json.RawMessage) {
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(data))
	}
}

// Table writes aligned columns to stdout.
type Table struct {
	w       *tabwriter.Writer
	headers []string
}

func NewTable(headers ...string) *Table {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	t := &Table{w: w, headers: headers}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	return t
}

func (t *Table) AddRow(values ...string) {
	fmt.Fprintln(t.w, strings.Join(values, "\t"))
}

func (t *Table) Flush() {
	t.w.Flush()
}

// Detail prints key-value pairs, aligned.
func PrintDetail(pairs [][2]string) {
	maxLen := 0
	for _, p := range pairs {
		if len(p[0]) > maxLen {
			maxLen = len(p[0])
		}
	}
	for _, p := range pairs {
		fmt.Printf("%-*s  %s\n", maxLen, p[0]+":", p[1])
	}
}
