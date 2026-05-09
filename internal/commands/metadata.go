package commands

import (
	"context"
	"encoding/json"
	"fmt"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newRegionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region",
		Aliases: []string{"regions"},
		Short:   "List datacenter regions",
	}
	cmd.AddCommand(newRegionListCmd())
	return cmd
}

func newRegionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available regions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			regions, _, err := c.Metadata.ListRegions(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(regions)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("CODE", "NAME", "COUNTRY", "DEFAULT")
			for _, r := range regions {
				flag := ""
				if r.Flag != nil {
					flag = *r.Flag
				}
				t.AddRow(r.Code, r.Name, flag+" "+r.CountryCode, fmt.Sprintf("%v", r.IsDefault))
			}
			t.Flush()
			return nil
		},
	}
}

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates"},
		Short:   "List OS templates",
	}
	cmd.AddCommand(newTemplateListCmd())
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	var category, vmType, region string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List OS templates available for VM creation",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.TemplateListOptions{}
			if category != "" {
				cat := spec.ListTemplatesParamsCategory(category)
				opts.Category = &cat
			}
			if vmType != "" {
				v := spec.ListTemplatesParamsVMType(vmType)
				opts.VMType = &v
			}
			if region != "" {
				r := spec.ListTemplatesParamsRegion(region)
				opts.Region = &r
			}
			templates, _, err := c.Metadata.ListTemplates(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(templates)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "VERSION", "OS", "CATEGORY", "REGION", "MIN CPU/RAM/DISK")
			for _, tmpl := range templates {
				t.AddRow(
					tmpl.ID.String(),
					tmpl.Name,
					tmpl.Version,
					tmpl.OsType,
					string(tmpl.Category),
					string(tmpl.Region),
					fmt.Sprintf("%dvCPU / %dGB / %dGB", tmpl.CPU, tmpl.RAM, tmpl.Storage),
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Filter by category (linux, windows)")
	cmd.Flags().StringVar(&vmType, "vm-type", "", "Filter by VM type (standard, premium)")
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	return cmd
}
