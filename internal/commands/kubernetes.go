package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newKubernetesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s"},
		Short:   "Manage Kubernetes clusters",
	}
	cmd.AddCommand(newK8sListCmd())
	cmd.AddCommand(newK8sGetCmd())
	cmd.AddCommand(newK8sCreateCmd())
	cmd.AddCommand(newK8sDeleteCmd())
	cmd.AddCommand(newK8sRenameCmd())
	cmd.AddCommand(newK8sKubeconfigCmd())
	cmd.AddCommand(newK8sNodesCmd())
	cmd.AddCommand(newK8sPoolCmd())
	cmd.AddCommand(newK8sUpgradeHACmd())
	cmd.AddCommand(newK8sVersionsCmd())
	cmd.AddCommand(newK8sPlansCmd())
	return cmd
}

func newK8sListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Kubernetes clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			clusters, _, err := c.Kubernetes.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(clusters)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("CLUSTER ID", "NAME", "VERSION", "STATUS", "WORKERS", "HA", "PRICE/MO")
			for _, cl := range clusters {
				version := ""
				if cl.K8SVersion != nil {
					version = *cl.K8SVersion
				}
				workers := ""
				if cl.WorkerCount != nil {
					workers = strconv.Itoa(*cl.WorkerCount)
				}
				ha := "no"
				if cl.HaEnabled != nil && *cl.HaEnabled {
					ha = "yes"
				}
				price := ""
				if cl.PricePerMonth != nil {
					price = fmt.Sprintf("$%.2f", *cl.PricePerMonth)
				}
				t.AddRow(cl.ClusterID, cl.Name, version, string(cl.Status), workers, ha, price)
			}
			t.Flush()
			return nil
		},
	}
}

func newK8sGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <cluster-id>",
		Short: "Get cluster details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			cl, _, err := c.Kubernetes.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(cl)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Cluster ID:   %s\n", cl.ClusterID)
			fmt.Printf("Name:         %s\n", cl.Name)
			fmt.Printf("Status:       %s", cl.Status)
			if cl.StatusMessage != nil && *cl.StatusMessage != "" {
				fmt.Printf(" (%s)", *cl.StatusMessage)
			}
			fmt.Println()
			fmt.Printf("Ready:        %t\n", cl.Ready)
			if cl.K8SVersion != nil {
				fmt.Printf("Version:      %s\n", *cl.K8SVersion)
			}
			if cl.APIEndpoint != nil {
				fmt.Printf("API endpoint: %s\n", *cl.APIEndpoint)
			}
			ha := false
			if cl.HaEnabled != nil {
				ha = *cl.HaEnabled
			}
			fmt.Printf("HA:           %t\n", ha)
			if cl.WorkerCount != nil {
				fmt.Printf("Workers:      %d\n", *cl.WorkerCount)
			}
			if cl.PricePerMonth != nil {
				fmt.Printf("Price:        $%.2f/mo\n", *cl.PricePerMonth)
			}
			if cl.NodePools != nil && len(*cl.NodePools) > 0 {
				fmt.Println("\nNode pools:")
				t := output.NewTable("ID", "NAME", "NODES", "AUTOSCALE", "STATUS")
				for _, p := range *cl.NodePools {
					autoscale := "off"
					if p.AutoscaleEnabled != nil && *p.AutoscaleEnabled {
						minN, maxN := 0, 0
						if p.MinNodes != nil {
							minN = *p.MinNodes
						}
						if p.MaxNodes != nil {
							maxN = *p.MaxNodes
						}
						autoscale = fmt.Sprintf("%d–%d", minN, maxN)
					}
					t.AddRow(p.ID.String(), p.Name, strconv.Itoa(p.NodeCount), autoscale, string(p.Status))
				}
				t.Flush()
			}
			return nil
		},
	}
}

func newK8sCreateCmd() *cobra.Command {
	var (
		name           string
		planID         int
		nodes          int
		versionID      int
		ha             bool
		wait           bool
		idempotencyKey string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Kubernetes cluster (pay-as-you-go accounts)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if planID == 0 {
				return fmt.Errorf("--plan is required (see 'raff kubernetes plans')")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateK8sClusterRequest{
				Name: name,
				NodePools: []raff.K8sNodePoolInput{{
					Name:      "default-pool",
					NodeCount: nodes,
					PlanID:    planID,
				}},
			}
			if versionID != 0 {
				req.K8SVersionID = &versionID
			}
			if ha {
				req.HaEnabled = &ha
			}
			cl, _, err := c.Kubernetes.Create(context.Background(), req, idempotencyKey)
			if err != nil {
				return err
			}
			if wait {
				fmt.Fprintf(os.Stderr, "Cluster %s creating — waiting for running (8–12 minutes)...\n", cl.ClusterID)
				cl, err = c.Kubernetes.WaitForStatus(context.Background(), cl.ClusterID, raff.K8sClusterStatusRunning)
				if err != nil {
					return err
				}
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(cl)
				output.PrintJSON(data)
				return nil
			}
			return printActionMessage(fmt.Sprintf("Cluster %s (%s) %s.", cl.ClusterID, cl.Name, cl.Status))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name (1–63 lowercase letters, digits or hyphens)")
	cmd.Flags().IntVar(&planID, "plan", 0, "Worker node plan ID (see 'raff kubernetes plans')")
	cmd.Flags().IntVar(&nodes, "nodes", 2, "Worker node count for the default pool (2–20)")
	cmd.Flags().IntVar(&versionID, "version", 0, "Kubernetes version ID (default: platform default)")
	cmd.Flags().BoolVar(&ha, "ha", false, "HA control plane (3 masters, flat monthly fee)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until the cluster is running")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Unique key making a retried create return the same cluster")
	return cmd
}

func newK8sDeleteCmd() *cobra.Command {
	var force, wait bool
	cmd := &cobra.Command{
		Use:   "delete <cluster-id>",
		Short: "Delete a cluster and all its infrastructure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to delete cluster %s and ALL its nodes? [y/N]: ", args[0])
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
			if _, err := c.Kubernetes.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			if wait {
				fmt.Fprintln(os.Stderr, "Deletion started — waiting...")
				if err := c.Kubernetes.WaitForDeleted(context.Background(), args[0]); err != nil {
					return err
				}
			}
			return printActionMessage("Cluster deletion completed." )
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until the cluster is fully deleted")
	return cmd
}

func newK8sRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <cluster-id> <new-name>",
		Short: "Rename a cluster (endpoint and DNS stay the same)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Kubernetes.Rename(context.Background(), args[0], args[1]); err != nil {
				return err
			}
			return printActionMessage("Cluster renamed.")
		},
	}
}

func newK8sKubeconfigCmd() *cobra.Command {
	var save string
	cmd := &cobra.Command{
		Use:   "kubeconfig <cluster-id>",
		Short: "Print the cluster's kubeconfig (or save it with --save)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			kc, _, err := c.Kubernetes.Kubeconfig(context.Background(), args[0])
			if err != nil {
				return err
			}
			if save != "" {
				if err := os.WriteFile(save, []byte(kc.Kubeconfig), 0o600); err != nil {
					return fmt.Errorf("failed to write %s: %w", save, err)
				}
				return printActionMessage(fmt.Sprintf("Kubeconfig saved to %s (endpoint %s).", save, kc.APIEndpoint))
			}
			fmt.Print(kc.Kubeconfig)
			return nil
		},
	}
	cmd.Flags().StringVar(&save, "save", "", "Write the kubeconfig to this file (mode 0600) instead of stdout")
	return cmd
}

func newK8sNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes <cluster-id>",
		Short: "List the cluster's nodes with live status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			nodes, _, err := c.Kubernetes.ListNodes(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(nodes)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("NAME", "ROLE", "IP", "STATUS", "READY")
			for _, n := range nodes {
				name := ""
				if n.LiveName != nil && *n.LiveName != "" {
					name = *n.LiveName
				} else if n.Name != nil {
					name = *n.Name
				}
				role := ""
				if n.Role != nil {
					role = string(*n.Role)
				}
				ip := ""
				if n.IPAddress != nil {
					ip = *n.IPAddress
				}
				status := ""
				if n.Status != nil {
					status = string(*n.Status)
				}
				ready := ""
				if n.Ready != nil {
					ready = strconv.FormatBool(*n.Ready)
				}
				t.AddRow(name, role, ip, status, ready)
			}
			t.Flush()
			return nil
		},
	}
}

func newK8sPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Manage node pools",
	}
	cmd.AddCommand(newK8sPoolListCmd())
	cmd.AddCommand(newK8sPoolAddCmd())
	cmd.AddCommand(newK8sPoolScaleCmd())
	cmd.AddCommand(newK8sPoolDeleteCmd())
	return cmd
}

func newK8sPoolListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <cluster-id>",
		Short: "List the cluster's node pools",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			pools, _, err := c.Kubernetes.ListNodePools(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(pools)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "NODES", "ACTUAL", "AUTOSCALE", "STATUS")
			for _, p := range pools {
				actual := ""
				if p.ActualNodeCount != nil {
					actual = strconv.Itoa(*p.ActualNodeCount)
				}
				autoscale := "off"
				if p.AutoscaleEnabled != nil && *p.AutoscaleEnabled {
					minN, maxN := 0, 0
					if p.MinNodes != nil {
						minN = *p.MinNodes
					}
					if p.MaxNodes != nil {
						maxN = *p.MaxNodes
					}
					autoscale = fmt.Sprintf("%d–%d", minN, maxN)
				}
				t.AddRow(p.ID.String(), p.Name, strconv.Itoa(p.NodeCount), actual, autoscale, string(p.Status))
			}
			t.Flush()
			return nil
		},
	}
}

func newK8sPoolAddCmd() *cobra.Command {
	var (
		name   string
		planID int
		nodes  int
	)
	cmd := &cobra.Command{
		Use:   "add <cluster-id>",
		Short: "Add a node pool to a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if planID == 0 {
				return fmt.Errorf("--plan is required (see 'raff kubernetes plans')")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			pool, _, err := c.Kubernetes.AddNodePool(context.Background(), args[0], &raff.AddK8sNodePoolRequest{
				Name:      name,
				NodeCount: nodes,
				PlanID:    &planID,
			})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(pool)
				output.PrintJSON(data)
				return nil
			}
			return printActionMessage(fmt.Sprintf("Node pool %s (%s) creating.", pool.Name, pool.ID.String()))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Pool name")
	cmd.Flags().IntVar(&planID, "plan", 0, "Worker node plan ID")
	cmd.Flags().IntVar(&nodes, "nodes", 2, "Node count (2–20)")
	return cmd
}

func newK8sPoolScaleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scale <cluster-id> <pool-id> <node-count>",
		Short: "Scale a node pool (scale-down drains nodes first)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := parseIntArg(args[2], "node-count")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Kubernetes.ScaleNodePool(context.Background(), args[0], args[1], count); err != nil {
				return err
			}
			return printActionMessage(fmt.Sprintf("Node pool scaling to %d nodes.", count))
		},
	}
}

func newK8sPoolDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <cluster-id> <pool-id>",
		Short: "Delete a node pool (nodes are drained first)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to delete pool %s? [y/N]: ", args[1])
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
			if _, err := c.Kubernetes.DeleteNodePool(context.Background(), args[0], args[1]); err != nil {
				return err
			}
			return printActionMessage("Node pool deletion started.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newK8sUpgradeHACmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "upgrade-ha <cluster-id>",
		Short: "Upgrade to an HA control plane (3 masters + redundant gateway; cannot be reversed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Upgrade cluster %s to HA? This adds the flat monthly HA fee and cannot be reversed. [y/N]: ", args[0])
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
			if _, err := c.Kubernetes.UpgradeHA(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("HA upgrade started — the new masters and gateway join over a few minutes.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newK8sVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "List available Kubernetes versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			versions, _, err := c.Kubernetes.ListVersions(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(versions)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "VERSION", "DEFAULT")
			for _, v := range versions {
				id, version, def := "", "", ""
				if v.ID != nil {
					id = strconv.Itoa(*v.ID)
				}
				if v.DisplayName != nil {
					version = *v.DisplayName
				} else if v.Version != nil {
					version = *v.Version
				}
				if v.IsDefault != nil && *v.IsDefault {
					def = "yes"
				}
				t.AddRow(id, version, def)
			}
			t.Flush()
			return nil
		},
	}
}

func newK8sPlansCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plans",
		Short: "List worker node plans with pricing",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			plans, _, err := c.Kubernetes.ListNodePlans(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(plans)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "VCPU", "MEMORY", "SSD", "PRICE/MO")
			for _, p := range plans.Plans {
				t.AddRow(
					strconv.Itoa(p.ID),
					p.Name,
					strconv.Itoa(p.Vcpu),
					fmt.Sprintf("%d GiB", p.MemoryGib),
					fmt.Sprintf("%d GiB", p.SsdGib),
					fmt.Sprintf("$%.2f", p.PricePerMonth),
				)
			}
			t.Flush()
			return nil
		},
	}
}
