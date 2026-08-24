package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	var flagFunction string
	var flagExport bool

	cmd := &cobra.Command{
		Use:   "dev -- <command> [args…]",
		Short: "Run a local dev command with your function's env vars injected",
		Long: `Run your normal dev server with the function's environment variables
(including secrets and binding-managed vars) injected — production-identical
local dev without an emulator, because Raff handlers are standard servers.

  raff dev -- uvicorn main:app --reload
  raff dev -- npm start
  raff dev -- go run .

The function is resolved from raff.toml (name) or --function. PORT defaults
to 8080 when unset. Use --export to print shell export lines instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := flagFunction
			if ref == "" {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				tomlCfg, err := loadRaffToml(dir)
				if err != nil {
					return err
				}
				if tomlCfg != nil && tomlCfg.Name != "" {
					ref = tomlCfg.Name
				} else {
					ref = slugifyName(filepath.Base(dir))
				}
			}
			if ref == "" {
				return fmt.Errorf("could not resolve a function — pass --function or set name in raff.toml")
			}

			c, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			vars, _, err := c.Functions.ListEnvVars(ctx, ref, true)
			if err != nil {
				return fmt.Errorf("fetching env vars for %s: %w", ref, err)
			}

			if flagExport {
				for _, v := range vars {
					fmt.Printf("export %s=%s\n", v.Key, shellQuote(v.Value))
				}
				return nil
			}

			dashArgs := args
			if at := cmd.ArgsLenAtDash(); at >= 0 {
				dashArgs = args[at:]
			}
			if len(dashArgs) == 0 {
				return fmt.Errorf("no command given — usage: raff dev -- <command> [args…] (or --export)")
			}

			env := os.Environ()
			injected := make([]string, 0, len(vars))
			for _, v := range vars {
				env = append(env, v.Key+"="+v.Value)
				injected = append(injected, v.Key)
			}
			if os.Getenv("PORT") == "" {
				env = append(env, "PORT=8080")
			}
			if len(injected) > 0 {
				fmt.Fprintf(os.Stderr, "raff dev: injected %s from %s\n", strings.Join(injected, ", "), ref)
			} else {
				fmt.Fprintf(os.Stderr, "raff dev: %s has no env vars — running with local env only\n", ref)
			}

			child := exec.Command(dashArgs[0], dashArgs[1:]...)
			child.Env = env
			child.Stdin = os.Stdin
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			if err := child.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFunction, "function", "", "Function name/slug (default: raff.toml name or directory name)")
	cmd.Flags().BoolVar(&flagExport, "export", false, "Print `export KEY=value` lines instead of running a command")
	return cmd
}

// shellQuote single-quotes a value for safe `export` output.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
