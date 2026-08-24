package commands

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newFunctionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "functions",
		Aliases: []string{"function", "fn"},
		Short:   "Manage serverless functions",
	}
	cmd.AddCommand(newFunctionsListCmd())
	cmd.AddCommand(newFunctionsGetCmd())
	return cmd
}

func newFunctionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List functions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			fns, _, err := c.Functions.List(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(fns)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("NAME", "RUNTIME", "STATUS", "REV", "URL")
			for _, f := range fns {
				t.AddRow(f.Slug, f.Runtime, f.Status, strconv.Itoa(f.CurrentRevisionNumber), f.URL)
			}
			t.Flush()
			return nil
		},
	}
}

func newFunctionsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show one function",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			fn, _, err := c.Functions.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(fn)
				output.PrintJSON(data)
				return nil
			}
			output.PrintDetail([][2]string{
				{"Name", fn.Name},
				{"Slug", fn.Slug},
				{"Runtime", fn.Runtime},
				{"Status", fn.Status},
				{"URL", fn.URL},
				{"Region", fn.Region},
				{"Memory", strconv.Itoa(fn.MemoryMB) + " MB"},
				{"Timeout", strconv.Itoa(fn.TimeoutSeconds) + "s"},
				{"Scale", strconv.Itoa(fn.MinScale) + " - " + strconv.Itoa(fn.MaxScale)},
				{"Revision", "rev-" + strconv.Itoa(fn.CurrentRevisionNumber)},
				{"Source", fn.SourceType},
			})
			return nil
		},
	}
}
