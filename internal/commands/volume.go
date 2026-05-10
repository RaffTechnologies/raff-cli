package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "volume",
		Aliases: []string{"volumes", "vol"},
		Short:   "Manage block storage volumes",
	}
	cmd.AddCommand(newVolumeListCmd())
	cmd.AddCommand(newVolumeGetCmd())
	cmd.AddCommand(newVolumeCreateCmd())
	cmd.AddCommand(newVolumeDeleteCmd())
	cmd.AddCommand(newVolumeResizeCmd())
	cmd.AddCommand(newVolumeAttachCmd())
	cmd.AddCommand(newVolumeDetachCmd())
	return cmd
}

func newVolumeListCmd() *cobra.Command {
	var region, vmID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.VolumeListOptions{}
			if region != "" {
				r := spec.ListVolumesParamsRegion(region)
				opts.Region = &r
			}
			if vmID != "" {
				vid := openapi_types.UUID{}
				if err := vid.UnmarshalText([]byte(vmID)); err != nil {
					return fmt.Errorf("invalid --vm-id: %w", err)
				}
				opts.VMID = &vid
			}
			vols, _, err := c.Volumes.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vols)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "SIZE", "TYPE", "STATUS", "REGION", "ATTACHED VM")
			for _, v := range vols {
				attached := ""
				if v.ProductVM != nil {
					attached = v.ProductVM.String()
				}
				region := ""
				if v.Region != nil {
					region = string(*v.Region)
				}
				t.AddRow(
					fmt.Sprintf("%d", v.ID),
					v.Name,
					fmt.Sprintf("%d GB", v.Size),
					string(v.VolumeType),
					string(v.Status),
					region,
					attached,
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Filter by attached VM UUID")
	return cmd
}

func newVolumeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <volume-id>",
		Short: "Get volume details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "volume-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			v, _, err := c.Volumes.Get(context.Background(), id)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(v)
				output.PrintJSON(data)
				return nil
			}
			attached := ""
			if v.ProductVM != nil {
				attached = v.ProductVM.String()
			}
			region := ""
			if v.Region != nil {
				region = string(*v.Region)
			}
			output.PrintDetail([][2]string{
				{"ID", fmt.Sprintf("%d", v.ID)},
				{"Name", v.Name},
				{"Size", fmt.Sprintf("%d GB", v.Size)},
				{"Type", string(v.VolumeType)},
				{"Status", string(v.Status)},
				{"Region", region},
				{"Attached VM", attached},
			})
			return nil
		},
	}
}

func newVolumeCreateCmd() *cobra.Command {
	var name, region, volumeType, filesystem, vmID string
	var size int
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a block storage volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateVolumeRequest{
				Name:       name,
				Size:       size,
				VolumeType: spec.CreateVolumeRequestVolumeType(volumeType),
			}
			if region != "" {
				r := spec.CreateVolumeRequestRegion(region)
				req.Region = &r
			}
			if filesystem != "" {
				f := spec.CreateVolumeRequestFilesystemType(filesystem)
				req.FilesystemType = &f
			}
			if vmID != "" {
				vid := openapi_types.UUID{}
				if err := vid.UnmarshalText([]byte(vmID)); err != nil {
					return fmt.Errorf("invalid --vm-id: %w", err)
				}
				req.VMID = &vid
			}
			v, _, err := c.Volumes.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(v)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Volume %q created (id: %d, status: %s)\n", v.Name, v.ID, v.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Volume name (required)")
	cmd.Flags().IntVar(&size, "size", 0, "Size in GB (required)")
	cmd.Flags().StringVar(&volumeType, "type", "", "Storage class (required). One of: nvme")
	cmd.Flags().StringVar(&region, "region", "", "Region (defaults to project's default)")
	cmd.Flags().StringVar(&filesystem, "filesystem", "", "Filesystem type for first attach (e.g. ext4, xfs)")
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Attach to this VM at create time (must be in same region)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("size")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}

func newVolumeDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <volume-id>",
		Short: "Delete a volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "volume-id")
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to delete volume %d? [y/N]: ", id)
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Volumes.Delete(context.Background(), id); err != nil {
				return err
			}
			return printActionMessage("Volume deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newVolumeResizeCmd() *cobra.Command {
	var newSize int
	cmd := &cobra.Command{
		Use:   "resize <volume-id>",
		Short: "Grow a volume (must be larger than current size)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "volume-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			r, _, err := c.Volumes.Resize(context.Background(), id, &raff.ResizeVolumeRequest{NewSize: newSize})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Volume %d resized to %d GB.\n", id, newSize)
			return nil
		},
	}
	cmd.Flags().IntVar(&newSize, "new-size", 0, "New size in GB (required, must be larger)")
	_ = cmd.MarkFlagRequired("new-size")
	return cmd
}

func newVolumeAttachCmd() *cobra.Command {
	var vmID string
	cmd := &cobra.Command{
		Use:   "attach <volume-id>",
		Short: "Attach a volume to a VM (same region required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "volume-id")
			if err != nil {
				return err
			}
			vid := openapi_types.UUID{}
			if err := vid.UnmarshalText([]byte(vmID)); err != nil {
				return fmt.Errorf("invalid --vm-id: %w", err)
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			v, _, err := c.Volumes.Attach(context.Background(), id, &raff.AttachVolumeRequest{VMID: vid})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(v)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Volume %d attached to VM %s.\n", id, vmID)
			return nil
		},
	}
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Target VM UUID (required)")
	_ = cmd.MarkFlagRequired("vm-id")
	return cmd
}

func newVolumeDetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detach <volume-id>",
		Short: "Detach a volume from its VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "volume-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Volumes.Detach(context.Background(), id); err != nil {
				return err
			}
			return printActionMessage("Volume detached.")
		},
	}
}

// parseIntArg converts a positional CLI integer argument with a clear
// error. Volumes / snapshots / backup schedules use int IDs per the API.
func parseIntArg(s, name string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be an integer", name, s)
	}
	return n, nil
}
