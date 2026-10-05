package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "database",
		Aliases: []string{"db", "databases"},
		Short:   "Manage databases (PostgreSQL, MySQL, Valkey, ClickHouse, Kafka)",
	}
	cmd.AddCommand(newDBListCmd())
	cmd.AddCommand(newDBGetCmd())
	cmd.AddCommand(newDBCreateCmd())
	cmd.AddCommand(newDBDeleteCmd())
	cmd.AddCommand(newDBRenameCmd())
	cmd.AddCommand(newDBResizeCmd())
	cmd.AddCommand(newDBResumeCmd())
	cmd.AddCommand(newDBConnectionCmd())
	cmd.AddCommand(newDBRotatePasswordCmd())
	cmd.AddCommand(newDBQueryCmd())
	cmd.AddCommand(newDBBrowseCmd())
	cmd.AddCommand(newDBUsersCmd())
	cmd.AddCommand(newDBBackupsCmd())
	cmd.AddCommand(newDBRestoreCmd())
	cmd.AddCommand(newDBPublicCmd())
	cmd.AddCommand(newDBVPCCmd())
	cmd.AddCommand(newDBMetricsCmd())
	cmd.AddCommand(newDBLogsCmd())
	cmd.AddCommand(newDBSlowQueriesCmd())
	cmd.AddCommand(newDBExtensionsCmd())
	cmd.AddCommand(newDBParametersCmd())
	cmd.AddCommand(newDBEnginesCmd())
	cmd.AddCommand(newDBPlansCmd())
	return cmd
}

// printJSONIf prints v as JSON and reports true when --output json is set.
func printJSONIf(v any) bool {
	if outputFormat() != output.FormatJSON {
		return false
	}
	data, _ := json.Marshal(v)
	output.PrintJSON(data)
	return true
}

func dbStatus(d *raff.Database) string {
	if d.Status == nil {
		return ""
	}
	s := string(*d.Status)
	if msg := raff.StringValue(d.StatusMessage); msg != "" && *d.Status != raff.DatabaseStatusRunning {
		s += " (" + msg + ")"
	}
	return s
}

func dbEngine(d *raff.Database) string {
	if d.Engine == nil {
		return ""
	}
	e := string(*d.Engine)
	if v := raff.StringValue(d.EngineVersion); v != "" {
		e += " " + v
	}
	return e
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func newDBListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			dbs, _, err := c.Databases.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if printJSONIf(dbs) {
				return nil
			}
			t := output.NewTable("ID", "NAME", "ENGINE", "STATUS", "PLAN", "STORAGE", "PUBLIC", "PRICE/MO")
			for i := range dbs {
				d := &dbs[i]
				plan := strconv.Itoa(raff.IntValue(d.PlanID))
				if raff.BoolValue(d.IsFree) {
					plan = "free"
				}
				price := ""
				if d.MonthlyPrice != nil {
					price = fmt.Sprintf("$%.2f", *d.MonthlyPrice)
				}
				t.AddRow(raff.StringValue(d.DatabaseID), raff.StringValue(d.Name), dbEngine(d), dbStatus(d), plan,
					fmt.Sprintf("%d GB", raff.IntValue(d.StorageGb)), yesNo(raff.BoolValue(d.PublicAccess)), price)
			}
			t.Flush()
			return nil
		},
	}
}

func printDatabaseDetail(d *raff.Database) {
	pairs := [][2]string{
		{"ID", raff.StringValue(d.DatabaseID)},
		{"Name", raff.StringValue(d.Name)},
		{"Engine", dbEngine(d)},
		{"Status", dbStatus(d)},
		{"Plan", strconv.Itoa(raff.IntValue(d.PlanID))},
		{"Free tier", yesNo(raff.BoolValue(d.IsFree))},
		{"Storage", fmt.Sprintf("%d GB", raff.IntValue(d.StorageGb))},
		{"High availability", yesNo(raff.BoolValue(d.HaEnabled))},
		{"Read replicas", strconv.Itoa(raff.IntValue(d.ReplicaCount))},
	}
	if raff.StringValue(d.VpcID) != "" {
		pairs = append(pairs, [2]string{"Private host", fmt.Sprintf("%s:%d", raff.StringValue(d.ConnectionHost), raff.IntValue(d.ConnectionPort))})
	} else {
		pairs = append(pairs, [2]string{"Private host", "none (no VPC)"})
	}
	if raff.BoolValue(d.PublicAccess) {
		pairs = append(pairs, [2]string{"Public host", fmt.Sprintf("%s:%d", raff.StringValue(d.PublicDNSHostname), raff.IntValue(d.PublicPort))})
		if d.PublicAllowlist != nil && len(*d.PublicAllowlist) > 0 {
			pairs = append(pairs, [2]string{"Allowed sources", strings.Join(*d.PublicAllowlist, ", ")})
		}
	} else {
		pairs = append(pairs, [2]string{"Public access", "off"})
	}
	if v := raff.StringValue(d.VpcID); v != "" {
		pairs = append(pairs, [2]string{"VPC", v})
	}
	if d.MonthlyPrice != nil {
		pairs = append(pairs, [2]string{"Price", fmt.Sprintf("$%.2f/mo", *d.MonthlyPrice)})
	}
	if d.CreatedAt != nil {
		pairs = append(pairs, [2]string{"Created", d.CreatedAt.Format(time.RFC3339)})
	}
	output.PrintDetail(pairs)
}

func newDBGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <database-id>",
		Short: "Show a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			printDatabaseDetail(d)
			return nil
		},
	}
}

// freePlanID returns the engine's free-tier plan.
func freePlanID(ctx context.Context, c *raff.Client, engine raff.DatabaseEngine) (int, error) {
	plans, _, err := c.Databases.ListPlans(ctx, engine)
	if err != nil {
		return 0, err
	}
	for _, p := range plans.Plans {
		if raff.BoolValue(p.IsFreeTier) {
			return p.ID, nil
		}
	}
	return 0, fmt.Errorf("%s has no free plan; pick one with --plan (see 'raff database plans --engine %s')", engine, engine)
}

func randomDBName(engine raff.DatabaseEngine) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", engine, hex.EncodeToString(b))
}

func newDBCreateCmd() *cobra.Command {
	var (
		name     string
		engine   string
		version  string
		planID   int
		storage  int
		ha       bool
		replicas int
		vpcID    string
		public   bool
		private  bool
		noWait   bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a database (defaults: PostgreSQL on the free plan)",
		Long: `Create a database and wait until it is running.

With no flags this creates a free PostgreSQL database with a generated name,
turns on public access (TLS and a generated password; limit sources later
with 'raff database public enable --allow') and prints its connection string.
The free plan is one per account. With --vpc, or with --private, the database
is reachable only inside a VPC.`,
		Example: `  raff database create
  raff database create --name orders --engine mysql
  raff database create --name app --plan 5 --storage 50 --public`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			eng := raff.DatabaseEngine(engine)
			if planID == 0 {
				if planID, err = freePlanID(ctx, c, eng); err != nil {
					return err
				}
			}
			if name == "" {
				name = randomDBName(eng)
			}
			req := &raff.CreateDatabaseRequest{Name: name, Engine: &eng, PlanID: planID}
			if version != "" {
				req.EngineVersion = &version
			}
			if storage > 0 {
				req.StorageGb = &storage
			}
			if ha {
				req.HaEnabled = &ha
			}
			if replicas > 0 {
				req.ReplicaCount = &replicas
			}
			if vpcID != "" {
				id, err := uuid.Parse(vpcID)
				if err != nil {
					return fmt.Errorf("invalid --vpc %q: %w", vpcID, err)
				}
				req.VpcID = &id
			}
			if public || (vpcID == "" && !private) {
				req.PublicAccess = raff.Bool(true)
			}
			d, _, err := c.Databases.Create(ctx, req)
			if err != nil {
				return err
			}
			id := raff.StringValue(d.DatabaseID)
			if noWait {
				if printJSONIf(d) {
					return nil
				}
				return printActionMessage(fmt.Sprintf("Database %s (%s) is %s. Check progress with 'raff database get %s'.", id, name, dbStatus(d), id))
			}
			fmt.Fprintf(os.Stderr, "Creating %s (%s)...\n", name, id)
			waitCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
			if d, err = c.Databases.WaitForStatus(waitCtx, id, raff.DatabaseStatusRunning); err != nil {
				return err
			}
			conn, _, err := c.Databases.Connection(ctx, id, true)
			if err != nil {
				return err
			}
			if printJSONIf(map[string]any{"database": d, "connection": conn}) {
				return nil
			}
			fmt.Printf("Database %s (%s) is running.\n\n", id, name)
			switch {
			case raff.StringValue(conn.PublicConnectionURI) != "":
				fmt.Println(raff.StringValue(conn.PublicConnectionURI))
			case raff.StringValue(d.VpcID) != "":
				fmt.Println(raff.StringValue(conn.ConnectionURI))
			default:
				fmt.Printf("Not reachable yet: run 'raff database public enable %s' or 'raff database vpc connect %s <vpc-id>'.\n", id, id)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Database name (default: generated)")
	cmd.Flags().StringVar(&engine, "engine", "postgres", "Engine: postgres, mysql, valkey, clickhouse, kafka")
	cmd.Flags().StringVar(&version, "version", "", "Engine major version (default: latest, see 'raff database engines')")
	cmd.Flags().IntVar(&planID, "plan", 0, "Plan ID (default: the engine's free plan, see 'raff database plans')")
	cmd.Flags().IntVar(&storage, "storage", 0, "Storage in GB (default: the plan's included storage)")
	cmd.Flags().BoolVar(&ha, "ha", false, "High availability: a standby that takes over on failure")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Read replicas (PostgreSQL only)")
	cmd.Flags().StringVar(&vpcID, "vpc", "", "VPC ID for the private endpoint")
	cmd.Flags().BoolVar(&public, "public", false, "Turn on public access (the default without --vpc)")
	cmd.Flags().BoolVar(&private, "private", false, "No public access, even without --vpc")
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Return at once instead of waiting until the database is running")
	return cmd
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var answer string
	_, _ = fmt.Scanln(&answer)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

func newDBDeleteCmd() *cobra.Command {
	var force, wait bool
	cmd := &cobra.Command{
		Use:   "delete <database-id>",
		Short: "Delete a database and its backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force && !confirm(fmt.Sprintf("Delete database %s and all its backups? This cannot be undone.", args[0])) {
				fmt.Println("Aborted.")
				return nil
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Databases.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			if wait {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				if err := c.Databases.WaitForDeleted(ctx, args[0]); err != nil {
					return err
				}
				return printActionMessage(fmt.Sprintf("Database %s deleted.", args[0]))
			}
			return printActionMessage(fmt.Sprintf("Database %s is being deleted.", args[0]))
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip the confirmation prompt")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait until the database is gone")
	return cmd
}

func newDBRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <database-id> <new-name>",
		Short: "Rename a database (hostnames stay the same)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.Rename(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Database %s renamed to %s.", args[0], raff.StringValue(d.Name)))
		},
	}
}

func newDBResizeCmd() *cobra.Command {
	var (
		planID   int
		storage  int
		ha       string
		replicas int
	)
	cmd := &cobra.Command{
		Use:   "resize <database-id>",
		Short: "Change plan, storage, high availability or read replicas",
		Example: `  raff database resize a1b2c3d4 --plan 6
  raff database resize a1b2c3d4 --storage 100
  raff database resize a1b2c3d4 --ha on --replicas 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			req := &raff.ScaleDatabaseRequest{}
			if planID > 0 {
				req.PlanID = &planID
			}
			if storage > 0 {
				req.StorageGb = &storage
			}
			switch ha {
			case "":
			case "on", "off":
				on := ha == "on"
				req.SetHa = raff.Bool(true)
				req.HasHa = &on
			default:
				return fmt.Errorf("--ha must be on or off")
			}
			// The API always applies replica_count, so keep the current count
			// unless --replicas was given.
			if cmd.Flags().Changed("replicas") {
				req.ReplicaCount = &replicas
			} else {
				cur, _, err := c.Databases.Get(ctx, args[0])
				if err != nil {
					return err
				}
				req.ReplicaCount = raff.Int(raff.IntValue(cur.ReplicaCount))
			}
			if req.PlanID == nil && req.StorageGb == nil && req.SetHa == nil && !cmd.Flags().Changed("replicas") {
				return fmt.Errorf("nothing to change: pass --plan, --storage, --ha or --replicas")
			}
			d, _, err := c.Databases.Scale(ctx, args[0], req)
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Database %s is %s.", args[0], dbStatus(d)))
		},
	}
	cmd.Flags().IntVar(&planID, "plan", 0, "New plan ID (same engine)")
	cmd.Flags().IntVar(&storage, "storage", 0, "New storage in GB (grow only)")
	cmd.Flags().StringVar(&ha, "ha", "", "High availability: on or off")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Read replica count (PostgreSQL only)")
	return cmd
}

func newDBResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <database-id>",
		Short: "Wake a free database paused after 7 idle days",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.Resume(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Database %s is waking up (%s).", args[0], dbStatus(d)))
		},
	}
}

func newDBConnectionCmd() *cobra.Command {
	var reveal, showCA bool
	cmd := &cobra.Command{
		Use:     "connection <database-id>",
		Aliases: []string{"conn"},
		Short:   "Show how to connect",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			conn, _, err := c.Databases.Connection(context.Background(), args[0], reveal)
			if err != nil {
				return err
			}
			if printJSONIf(conn) {
				return nil
			}
			password := "(hidden, add --reveal)"
			if reveal {
				password = raff.StringValue(conn.Password)
			}
			pairs := [][2]string{
				{"User", raff.StringValue(conn.Username)},
				{"Password", password},
				{"Database", raff.StringValue(conn.DatabaseName)},
			}
			if pub := raff.StringValue(conn.PublicHost); pub != "" {
				pairs = append(pairs,
					[2]string{"Public host", fmt.Sprintf("%s:%d", pub, raff.IntValue(conn.PublicPort))},
					[2]string{"Public URI", raff.StringValue(conn.PublicConnectionURI)})
			}
			pairs = append(pairs,
				[2]string{"Private host", fmt.Sprintf("%s:%d (inside the VPC)", raff.StringValue(conn.Host), raff.IntValue(conn.Port))},
				[2]string{"Private URI", raff.StringValue(conn.ConnectionURI)})
			if !strings.Contains(raff.StringValue(conn.Host), ".") {
				// No VPC: the private address is cluster-internal, not usable.
				pairs = pairs[:len(pairs)-2]
			}
			output.PrintDetail(pairs)
			// Kafka has no URI scheme and its clients must trust the CA.
			isKafka := !strings.Contains(raff.StringValue(conn.ConnectionURI), "://")
			if ca := raff.StringValue(conn.CaCert); ca != "" && (showCA || isKafka) {
				fmt.Println("\nCA certificate (save it and point your client at it):")
				fmt.Println(ca)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Include the password")
	cmd.Flags().BoolVar(&showCA, "ca", false, "Print the CA certificate (for sslmode=verify-full; always shown for Kafka)")
	return cmd
}

func newDBRotatePasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate-password <database-id>",
		Short: "Give the admin user a new password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Databases.RotateCredentials(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage(fmt.Sprintf("New password set. Show it with 'raff database connection %s --reveal'.", args[0]))
		},
	}
}

func printRows(columns []string, rows []raff.DatabaseRow) {
	t := output.NewTable(columns...)
	for _, r := range rows {
		cells := []string{}
		if r.Cells != nil {
			cells = append(cells, *r.Cells...)
		}
		if r.Nulls != nil {
			for i, isNull := range *r.Nulls {
				if isNull && i < len(cells) {
					cells[i] = "NULL"
				}
			}
		}
		t.AddRow(cells...)
	}
	t.Flush()
}

func newDBQueryCmd() *cobra.Command {
	var (
		write   bool
		maxRows int
	)
	cmd := &cobra.Command{
		Use:   "query <database-id> <statement>",
		Short: "Run one SQL statement (or one Valkey command)",
		Long: `Run one statement as the admin user and print the result.

Read-only unless --write is set. Returns at most 1,000 rows by default
(--max-rows, up to 5,000) and stops after 15 seconds.`,
		Example: `  raff database query a1b2c3d4 "SELECT now()"
  raff database query a1b2c3d4 --write "CREATE TABLE notes (id serial PRIMARY KEY, body text)"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.DatabaseQueryRequest{Statement: args[1], ReadOnly: raff.Bool(!write)}
			if maxRows > 0 {
				req.MaxRows = &maxRows
			}
			res, _, err := c.Databases.Query(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if printJSONIf(res) {
				return nil
			}
			if res.Columns != nil && len(*res.Columns) > 0 {
				rows := []raff.DatabaseRow{}
				if res.Rows != nil {
					rows = *res.Rows
				}
				printRows(*res.Columns, rows)
			}
			if res.Notices != nil {
				for _, n := range *res.Notices {
					fmt.Fprintln(os.Stderr, "NOTICE:", n)
				}
			}
			summary := raff.StringValue(res.Command)
			if raff.BoolValue(res.Truncated) {
				summary += " (more rows exist, raise --max-rows)"
			}
			if res.DurationMs != nil {
				summary += fmt.Sprintf(" in %.1f ms", *res.DurationMs)
			}
			fmt.Fprintln(os.Stderr, strings.TrimSpace(summary))
			return nil
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "Allow statements that change data or schema")
	cmd.Flags().IntVar(&maxRows, "max-rows", 0, "Most rows to return (default 1,000, up to 5,000)")
	return cmd
}

func newDBBrowseCmd() *cobra.Command {
	var (
		limit  int
		cursor string
		match  string
	)
	cmd := &cobra.Command{
		Use:   "browse <database-id> [schema [table]]",
		Short: "List schemas, tables, or a table's rows (Valkey: keys)",
		Example: `  raff database browse a1b2c3d4                 # schemas
  raff database browse a1b2c3d4 public          # tables in public
  raff database browse a1b2c3d4 public users    # rows of public.users`,
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			path := args[1:]
			req := &raff.DatabaseBrowseRequest{Path: &path}
			if limit > 0 {
				req.Limit = &limit
			}
			if cursor != "" {
				req.Cursor = &cursor
			}
			if match != "" {
				req.Match = &match
			}
			res, _, err := c.Databases.Browse(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if printJSONIf(res) {
				return nil
			}
			if res.Kind != nil && *res.Kind == "rows" {
				cols := []string{}
				if res.Columns != nil {
					cols = *res.Columns
				}
				rows := []raff.DatabaseRow{}
				if res.Rows != nil {
					rows = *res.Rows
				}
				printRows(cols, rows)
				if res.Meta != nil {
					for k, v := range *res.Meta {
						fmt.Fprintf(os.Stderr, "%s: %s\n", k, v)
					}
				}
			} else {
				t := output.NewTable("NAME", "KIND", "DETAIL")
				if res.Nodes != nil {
					for _, n := range *res.Nodes {
						t.AddRow(raff.StringValue(n.Name), raff.StringValue(n.Kind), raff.StringValue(n.Detail))
					}
				}
				t.Flush()
			}
			if next := raff.StringValue(res.NextCursor); next != "" {
				fmt.Fprintf(os.Stderr, "More available: add --cursor %s\n", next)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Items or rows per page")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor from the previous page")
	cmd.Flags().StringVar(&match, "match", "", "Valkey key pattern, e.g. user:*")
	return cmd
}

func newDBUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage database users",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list <database-id>",
		Short: "List database users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			users, _, err := c.Databases.ListUsers(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(users) {
				return nil
			}
			t := output.NewTable("NAME", "ACCESS", "SCOPE", "KIND")
			for _, u := range users {
				kind := ""
				if u.Kind != nil {
					kind = string(*u.Kind)
				}
				if kind == "system" {
					continue
				}
				t.AddRow(raff.StringValue(u.Name), raff.StringValue(u.Role), raff.StringValue(u.Scope), kind)
			}
			t.Flush()
			return nil
		},
	})
	var role string
	create := &cobra.Command{
		Use:   "create <database-id> <name>",
		Short: "Add a user and print its password",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r := raff.DatabaseUserRole(role)
			if r != raff.DatabaseUserReadOnly && r != raff.DatabaseUserReadWrite {
				return fmt.Errorf("--role must be readonly or readwrite")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			u, password, _, err := c.Databases.CreateUser(context.Background(), args[0], args[1], r)
			if err != nil {
				return err
			}
			if printJSONIf(map[string]any{"user": u, "password": password}) {
				return nil
			}
			output.PrintDetail([][2]string{{"User", raff.StringValue(u.Name)}, {"Access", raff.StringValue(u.Role)}, {"Password", password}})
			return nil
		},
	}
	create.Flags().StringVar(&role, "role", "readonly", "Access: readonly or readwrite")
	cmd.AddCommand(create)
	cmd.AddCommand(&cobra.Command{
		Use:   "password <database-id> <name>",
		Short: "Show a user's password",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			cred, _, err := c.Databases.UserCredential(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}
			if printJSONIf(cred) {
				return nil
			}
			output.PrintDetail([][2]string{{"User", cred.Username}, {"Password", cred.Password}})
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "rotate <database-id> <name>",
		Short: "Give a user a new password",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			password, _, err := c.Databases.RotateUser(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}
			if printJSONIf(map[string]string{"user": args[1], "password": password}) {
				return nil
			}
			output.PrintDetail([][2]string{{"User", args[1]}, {"Password", password}})
			return nil
		},
	})
	var force bool
	del := &cobra.Command{
		Use:   "delete <database-id> <name>",
		Short: "Remove a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force && !confirm(fmt.Sprintf("Remove user %s from database %s?", args[1], args[0])) {
				fmt.Println("Aborted.")
				return nil
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Databases.DeleteUser(context.Background(), args[0], args[1]); err != nil {
				return err
			}
			return printActionMessage(fmt.Sprintf("User %s removed.", args[1]))
		},
	}
	del.Flags().BoolVar(&force, "force", false, "Skip the confirmation prompt")
	cmd.AddCommand(del)
	return cmd
}

func newDBBackupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backups",
		Short: "List and create database backups",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list <database-id>",
		Short: "List backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			backups, _, err := c.Databases.ListBackups(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(backups) {
				return nil
			}
			t := output.NewTable("BACKUP ID", "TYPE", "STATUS", "TAKEN AT", "SIZE", "RESTORABLE")
			windowStart := ""
			for _, b := range backups {
				typ, status := "", ""
				if b.BackupType != nil {
					typ = string(*b.BackupType)
				}
				if b.Status != nil {
					status = string(*b.Status)
				}
				id := ""
				if b.ID != nil {
					id = b.ID.String()
				}
				t.AddRow(id, typ, status, raff.StringValue(b.BackupTimestamp),
					fmt.Sprintf("%.1f MB", float64(raff.IntValue(b.SizeBytes))/1e6), yesNo(raff.BoolValue(b.Restorable)))
				if w := raff.StringValue(b.RestoreWindowStart); w != "" {
					windowStart = w
				}
			}
			t.Flush()
			if windowStart != "" {
				fmt.Fprintf(os.Stderr, "\nPoint-in-time restore available from %s until now.\n", windowStart)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create <database-id>",
		Short: "Start a backup now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			b, _, err := c.Databases.CreateBackup(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(b) {
				return nil
			}
			id := ""
			if b.ID != nil {
				id = b.ID.String()
			}
			return printActionMessage(fmt.Sprintf("Backup %s started. Check it with 'raff database backups list %s'.", id, args[0]))
		},
	})
	return cmd
}

func newDBRestoreCmd() *cobra.Command {
	var (
		backupID string
		at       string
		inPlace  bool
		newName  string
		ha       bool
		force    bool
	)
	cmd := &cobra.Command{
		Use:   "restore <database-id>",
		Short: "Restore to a backup or point in time, into a new database or in place",
		Example: `  raff database restore a1b2c3d4 --time 2026-10-05T09:30:00Z
  raff database restore a1b2c3d4 --backup 9e1f5a7e-... --name orders-copy
  raff database restore a1b2c3d4 --in-place --time 2026-10-05T09:30:00Z`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &raff.RestoreDatabaseRequest{}
			if backupID != "" {
				id, err := uuid.Parse(backupID)
				if err != nil {
					return fmt.Errorf("invalid --backup %q: %w", backupID, err)
				}
				req.BackupID = &id
			}
			if at != "" {
				t, err := time.Parse(time.RFC3339, at)
				if err != nil {
					return fmt.Errorf("--time must be RFC 3339, e.g. 2026-10-05T09:30:00Z: %w", err)
				}
				req.PitrTimestamp = &t
			}
			if inPlace {
				if newName != "" || ha {
					return fmt.Errorf("--name and --ha apply only to a restore into a new database")
				}
				if !force && !confirm(fmt.Sprintf("Overwrite database %s? Data written after the restore point is lost.", args[0])) {
					fmt.Println("Aborted.")
					return nil
				}
				req.InPlace = raff.Bool(true)
			} else {
				req.CloneToNew = raff.Bool(true)
				if newName != "" {
					req.NewName = &newName
				}
				if ha {
					req.HaEnabled = &ha
				}
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			newID, _, err := c.Databases.Restore(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if printJSONIf(map[string]string{"database_id": args[0], "new_database_id": newID}) {
				return nil
			}
			if inPlace {
				return printActionMessage(fmt.Sprintf("Restoring %s in place. Follow it with 'raff database get %s'.", args[0], args[0]))
			}
			return printActionMessage(fmt.Sprintf("Restoring into new database %s. Follow it with 'raff database get %s'.", newID, newID))
		},
	}
	cmd.Flags().StringVar(&backupID, "backup", "", "Backup ID (see 'raff database backups list')")
	cmd.Flags().StringVar(&at, "time", "", "Point in time, RFC 3339 (PostgreSQL)")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Overwrite this database instead of creating a new one")
	cmd.Flags().StringVar(&newName, "name", "", "Name for the new database")
	cmd.Flags().BoolVar(&ha, "ha", false, "High availability on the new database")
	cmd.Flags().BoolVar(&force, "force", false, "Skip the confirmation prompt for --in-place")
	return cmd
}

func newDBPublicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "public",
		Short: "Turn public access on or off",
	}
	var allow []string
	enable := &cobra.Command{
		Use:   "enable <database-id>",
		Short: "Turn on public access",
		Example: `  raff database public enable a1b2c3d4
  raff database public enable a1b2c3d4 --allow 203.0.113.0/24,198.51.100.7/32`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var list []string
			if cmd.Flags().Changed("allow") {
				list = allow
				if list == nil {
					list = []string{}
				}
			}
			d, _, err := c.Databases.SetPublicAccess(context.Background(), args[0], true, list)
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Public access on: %s port %d.", raff.StringValue(d.PublicDNSHostname), raff.IntValue(d.PublicPort)))
		},
	}
	enable.Flags().StringSliceVar(&allow, "allow", nil, "Source CIDRs allowed to connect (empty = all)")
	cmd.AddCommand(enable)
	cmd.AddCommand(&cobra.Command{
		Use:   "disable <database-id>",
		Short: "Turn off public access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.SetPublicAccess(context.Background(), args[0], false, nil)
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Public access off for %s.", args[0]))
		},
	})
	return cmd
}

func newDBVPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "Connect a database to a VPC or disconnect it",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "connect <database-id> <vpc-id>",
		Short: "Make the database reachable inside a VPC",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.ConnectVPC(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Database %s connected to VPC %s.", args[0], args[1]))
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disconnect <database-id>",
		Short: "Detach the database from its VPC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.Databases.DisconnectVPC(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(d) {
				return nil
			}
			return printActionMessage(fmt.Sprintf("Database %s disconnected from its VPC.", args[0]))
		},
	})
	return cmd
}

func humanBytes(b int) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/(1<<20))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func newDBMetricsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "metrics <database-id>",
		Short: "Show CPU, memory, storage and connections",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			m, _, err := c.Databases.Metrics(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(m) {
				return nil
			}
			if raff.StringValue(m.MetricsUpdatedAt) == "" {
				fmt.Println("Not measured yet. Metrics refresh about every 60 seconds.")
				return nil
			}
			output.PrintDetail([][2]string{
				{"CPU", fmt.Sprintf("%d / %d millicores", raff.IntValue(m.CPUUsedMillicores), raff.IntValue(m.CPUCapacityMillicores))},
				{"Memory", fmt.Sprintf("%s / %s", humanBytes(raff.IntValue(m.MemUsedBytes)), humanBytes(raff.IntValue(m.MemCapacityBytes)))},
				{"Storage", fmt.Sprintf("%s / %s", humanBytes(raff.IntValue(m.StorageUsedBytes)), humanBytes(raff.IntValue(m.StorageCapacityBytes)))},
				{"Connections", fmt.Sprintf("%d / %d", raff.IntValue(m.ConnectionsActive), raff.IntValue(m.ConnectionsMax))},
				{"Replication lag", humanBytes(raff.IntValue(m.ReplicationLagBytes))},
				{"Measured at", raff.StringValue(m.MetricsUpdatedAt)},
			})
			return nil
		},
	}
}

func newDBLogsCmd() *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "logs <database-id>",
		Short: "Show recent engine log lines",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			lines, _, err := c.Databases.Logs(context.Background(), args[0], tail)
			if err != nil {
				return err
			}
			if printJSONIf(lines) {
				return nil
			}
			for _, l := range lines {
				fmt.Println(l)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 200, "Number of lines")
	return cmd
}

func newDBSlowQueriesCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "slow-queries <database-id>",
		Short: "Show the slowest recent queries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, _, err := c.Databases.SlowQueries(context.Background(), args[0], limit)
			if err != nil {
				return err
			}
			if printJSONIf(res) {
				return nil
			}
			if !res.Collecting {
				fmt.Fprintln(os.Stderr, "Query statistics are not being collected yet.")
			}
			t := output.NewTable("CALLS", "MEAN MS", "TOTAL MS", "QUERY")
			for _, q := range res.Queries {
				mean, total := float32(0), float32(0)
				if q.MeanMs != nil {
					mean = *q.MeanMs
				}
				if q.TotalMs != nil {
					total = *q.TotalMs
				}
				query := strings.Join(strings.Fields(raff.StringValue(q.Query)), " ")
				if len(query) > 100 {
					query = query[:97] + "..."
				}
				t.AddRow(strconv.Itoa(raff.IntValue(q.Calls)), fmt.Sprintf("%.1f", mean), fmt.Sprintf("%.1f", total), query)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of queries")
	return cmd
}

func newDBExtensionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extensions",
		Short: "List, enable or disable PostgreSQL extensions",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list <database-id>",
		Short: "List available extensions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			exts, _, err := c.Databases.ListExtensions(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(exts) {
				return nil
			}
			t := output.NewTable("NAME", "INSTALLED", "VERSION", "DESCRIPTION")
			for _, e := range exts {
				t.AddRow(raff.StringValue(e.Name), yesNo(raff.BoolValue(e.Installed)), raff.StringValue(e.Version), raff.StringValue(e.Description))
			}
			t.Flush()
			return nil
		},
	})
	for _, on := range []bool{true, false} {
		use, short := "enable", "Install an extension"
		if !on {
			use, short = "disable", "Remove an extension"
		}
		cmd.AddCommand(&cobra.Command{
			Use:   use + " <database-id> <extension>",
			Short: short,
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				c, err := newClient()
				if err != nil {
					return err
				}
				if _, err := c.Databases.SetExtension(context.Background(), args[0], args[1], on); err != nil {
					return err
				}
				return printActionMessage(fmt.Sprintf("Extension %s %sd.", args[1], use))
			},
		})
	}
	return cmd
}

func newDBParametersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "parameters <database-id>",
		Short: "Show the engine settings in effect",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			params, _, err := c.Databases.ListParameters(context.Background(), args[0])
			if err != nil {
				return err
			}
			if printJSONIf(params) {
				return nil
			}
			t := output.NewTable("NAME", "VALUE", "UNIT", "DEFAULT", "SOURCE")
			for _, p := range params {
				source := ""
				if p.Source != nil {
					source = string(*p.Source)
				}
				t.AddRow(raff.StringValue(p.Name), raff.StringValue(p.Value), raff.StringValue(p.Unit), raff.StringValue(p.DefaultValue), source)
			}
			t.Flush()
			return nil
		},
	}
}

func newDBEnginesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "engines",
		Short: "List engines and versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			engines, _, err := c.Databases.ListEngines(context.Background())
			if err != nil {
				return err
			}
			if printJSONIf(engines) {
				return nil
			}
			t := output.NewTable("ENGINE", "NAME", "VERSIONS", "AVAILABLE")
			for _, e := range engines {
				eng, versions := "", ""
				if e.Engine != nil {
					eng = string(*e.Engine)
				}
				if e.Versions != nil {
					versions = strings.Join(*e.Versions, ", ")
				}
				t.AddRow(eng, raff.StringValue(e.DisplayName), versions, yesNo(raff.BoolValue(e.Available)))
			}
			t.Flush()
			return nil
		},
	}
}

func newDBPlansCmd() *cobra.Command {
	var engine string
	cmd := &cobra.Command{
		Use:   "plans",
		Short: "List plans with prices",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			plans, _, err := c.Databases.ListPlans(context.Background(), raff.DatabaseEngine(engine))
			if err != nil {
				return err
			}
			if printJSONIf(plans) {
				return nil
			}
			t := output.NewTable("ID", "ENGINE", "NAME", "VCPU", "MEMORY", "STORAGE", "PRICE/MO")
			for _, p := range plans.Plans {
				mem := fmt.Sprintf("%d GiB", p.MemoryGib)
				if mib := raff.IntValue(p.MemoryMib); mib > 0 {
					mem = fmt.Sprintf("%d MiB", mib)
				}
				price := fmt.Sprintf("$%.2f", p.PricePerMonth)
				if raff.BoolValue(p.IsFreeTier) {
					price = "free"
				}
				t.AddRow(strconv.Itoa(p.ID), string(p.Engine), p.Name, strconv.Itoa(p.Vcpu), mem, fmt.Sprintf("%d GiB", p.StorageGib), price)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&engine, "engine", "", "Only this engine")
	return cmd
}
