package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPricingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Browse the public pricing catalog",
	}
	cmd.AddCommand(newPricingVMCmd())
	cmd.AddCommand(newPricingVolumeCmd())
	cmd.AddCommand(newPricingBackupCmd())
	cmd.AddCommand(newPricingSnapshotCmd())
	cmd.AddCommand(newPricingIPCmd())
	return cmd
}

func newPricingVMCmd() *cobra.Command {
	var region, vmType string
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "List VM pricing plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.VMPricingListOptions{}
			if region != "" {
				r := spec.ListVMPricingParamsRegion(region)
				opts.Region = &r
			}
			if vmType != "" {
				t := spec.ListVMPricingParamsType(vmType)
				opts.Type = &t
			}
			plans, _, err := c.Pricing.ListVM(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(plans)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "TYPE", "REGION", "vCPU", "RAM", "SSD", "TRANSFER", "$/HR", "$/MO", "STOCK")
			for _, p := range plans {
				stock := "In Stock"
				if p.OutOfStock != nil && *p.OutOfStock {
					stock = "Sold Out"
				}
				t.AddRow(
					fmt.Sprintf("%d", p.ID),
					string(p.VMType),
					string(p.Region),
					fmt.Sprintf("%d", p.Vcpu),
					fmt.Sprintf("%d GiB", p.MemoryGib),
					fmt.Sprintf("%d GiB", p.SsdGib),
					fmt.Sprintf("%d GiB", p.TransferGib),
					fmt.Sprintf("%.4f", p.PricePerHour),
					fmt.Sprintf("%.2f", p.MonthlyPrice),
					stock,
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	cmd.Flags().StringVar(&vmType, "vm-type", "", "Filter by type (standard, premium)")
	return cmd
}

func newPricingVolumeCmd() *cobra.Command {
	var region string
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Show volume (block storage) pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.VolumePricingListOptions{}
			if region != "" {
				r := spec.ListVolumePricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err := c.Pricing.ListVolume(context.Background(), opts)
			if err != nil {
				return err
			}
			return printStoragePricing(p, "volume")
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	return cmd
}

func newPricingBackupCmd() *cobra.Command {
	var region string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Show backup storage pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.BackupPricingListOptions{}
			if region != "" {
				r := spec.ListBackupPricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err := c.Pricing.ListBackup(context.Background(), opts)
			if err != nil {
				return err
			}
			return printStoragePricing(p, "backup")
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	return cmd
}

func newPricingSnapshotCmd() *cobra.Command {
	var region string
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Show snapshot storage pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.SnapshotPricingListOptions{}
			if region != "" {
				r := spec.ListSnapshotPricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err := c.Pricing.ListSnapshot(context.Background(), opts)
			if err != nil {
				return err
			}
			return printStoragePricing(p, "snapshot")
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	return cmd
}

func newPricingIPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "Show floating IP pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			p, _, err := c.Pricing.ListIP(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(p)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("FAMILY", "$/HR", "$/MO", "$/YR", "$/24MO")
			if p.Ipv4 != nil {
				t.AddRow("IPv4", floatVal(p.Ipv4.PricePerHour), floatVal(p.Ipv4.MonthlyPrice), floatVal(p.Ipv4.YearlyPrice), floatVal(p.Ipv4.TwentyFourMonthPrice))
			}
			if p.Ipv6 != nil {
				t.AddRow("IPv6", floatVal(p.Ipv6.PricePerHour), floatVal(p.Ipv6.MonthlyPrice), floatVal(p.Ipv6.YearlyPrice), floatVal(p.Ipv6.TwentyFourMonthPrice))
			}
			t.Flush()
			return nil
		},
	}
}

func printStoragePricing(p *raff.StoragePricing, label string) error {
	if outputFormat() == output.FormatJSON {
		data, _ := json.Marshal(p)
		output.PrintJSON(data)
		return nil
	}
	region := ""
	if p.Region != nil {
		region = string(*p.Region)
	}
	id := ""
	if p.ID != nil {
		id = strconv.Itoa(*p.ID)
	}
	output.PrintDetail([][2]string{
		{"ID", id},
		{"Type", label},
		{"Region", region},
		{"Per GB / hour", floatVal(p.PricePerGbHour)},
		{"Per GB / month", floatVal(p.PricePerGbMonth)},
	})
	return nil
}

func floatVal(p *float32) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *p)
}
