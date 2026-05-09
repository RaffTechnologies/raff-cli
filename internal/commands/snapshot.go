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

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snapshots", "snap"},
		Short:   "Manage VM and volume snapshots",
	}
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotGetCmd())
	cmd.AddCommand(newSnapshotCreateCmd())
	cmd.AddCommand(newSnapshotRenameCmd())
	cmd.AddCommand(newSnapshotRestoreCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())
	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	var vmID, volumeIDStr, snapType string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snapshots, optionally filtered by VM, volume, or type",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.SnapshotListOptions{}
			if vmID != "" {
				vid := openapi_types.UUID{}
				if err := vid.UnmarshalText([]byte(vmID)); err != nil {
					return fmt.Errorf("invalid --vm-id: %w", err)
				}
				opts.VMID = &vid
			}
			if volumeIDStr != "" {
				v, err := parseIntArg(volumeIDStr, "volume-id")
				if err != nil {
					return err
				}
				opts.VolumeID = &v
			}
			if snapType != "" {
				st := spec.ListSnapshotsParamsType(snapType)
				opts.Type = &st
			}
			snaps, _, err := c.Snapshots.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(snaps)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "SIZE", "TYPE", "STATUS", "CREATED")
			for _, s := range snaps {
				size := ""
				if s.Size != nil {
					size = *s.Size
				}
				created := ""
				if s.CreatedAt != nil {
					created = s.CreatedAt.Format("2006-01-02")
				}
				status := ""
				if s.Status != nil {
					status = string(*s.Status)
				}
				t.AddRow(
					fmt.Sprintf("%d", s.ID),
					s.Name,
					size,
					string(s.Type),
					status,
					created,
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Filter by source VM UUID")
	cmd.Flags().StringVar(&volumeIDStr, "volume-id", "", "Filter by source volume ID")
	cmd.Flags().StringVar(&snapType, "type", "", "Filter by snapshot type: vm or volume")
	return cmd
}

func newSnapshotGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <snapshot-id>",
		Short: "Get snapshot details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "snapshot-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			s, _, err := c.Snapshots.Get(context.Background(), id)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			size := ""
			if s.Size != nil {
				size = *s.Size
			}
			status := ""
			if s.Status != nil {
				status = string(*s.Status)
			}
			output.PrintDetail([][2]string{
				{"ID", fmt.Sprintf("%d", s.ID)},
				{"Name", s.Name},
				{"Size", size},
				{"Type", string(s.Type)},
				{"Status", status},
			})
			return nil
		},
	}
}

func newSnapshotCreateCmd() *cobra.Command {
	var name, resourceType, resourceID, volumeIDStr string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Take a snapshot of a VM disk or a volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateSnapshotRequest{
				Name:         name,
				ResourceType: spec.CreateSnapshotRequestResourceType(resourceType),
			}
			if resourceID != "" {
				rid := openapi_types.UUID{}
				if err := rid.UnmarshalText([]byte(resourceID)); err != nil {
					return fmt.Errorf("invalid --resource-id: %w", err)
				}
				req.ResourceID = &rid
			}
			if volumeIDStr != "" {
				v, err := parseIntArg(volumeIDStr, "volume-id")
				if err != nil {
					return err
				}
				req.VolumeID = &v
			}
			s, _, err := c.Snapshots.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Snapshot %q created (id: %d)\n", s.Name, s.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Source: vm or volume (required)")
	cmd.Flags().StringVar(&resourceID, "resource-id", "", "Source VM UUID (required when --resource-type=vm)")
	cmd.Flags().StringVar(&volumeIDStr, "volume-id", "", "Source volume ID (required when --resource-type=volume)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("resource-type")
	return cmd
}

func newSnapshotRenameCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "rename <snapshot-id>",
		Short: "Rename a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "snapshot-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			s, _, err := c.Snapshots.Rename(context.Background(), id, &raff.RenameSnapshotRequest{Name: name})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Snapshot renamed to %q.\n", s.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSnapshotRestoreCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <snapshot-id>",
		Short: "Restore a VM or volume from a snapshot (irreversible — overwrites current state)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "snapshot-id")
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Restore from snapshot %d? Current disk state will be lost. [y/N]: ", id)
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
			if _, err := c.Snapshots.Restore(context.Background(), id); err != nil {
				return err
			}
			return printActionMessage("Snapshot restore initiated.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newSnapshotDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <snapshot-id>",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "snapshot-id")
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Delete snapshot %d? [y/N]: ", id)
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
			if _, err := c.Snapshots.Delete(context.Background(), id); err != nil {
				return err
			}
			return printActionMessage("Snapshot deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
