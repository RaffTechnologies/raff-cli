package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import functions from other platforms",
	}
	cmd.AddCommand(newImportLambdaCmd())
	return cmd
}

func newImportLambdaCmd() *cobra.Command {
	var flagRuntime string
	var flagHandler string
	var flagOut string
	var flagAsIs bool

	cmd := &cobra.Command{
		Use:   "lambda <handler-file>",
		Short: "Convert an AWS Lambda handler to a portable Raff function",
		Long: `Convert AWS Lambda handler code to a standard portable handler
(FastAPI / Web Fetch / net-http) — no Raff or AWS lock-in.

The conversion is stateless: nothing is created until you review the output
and run 'raff deploy' yourself. Every guess is flagged for review.

  raff import lambda lambda_function.py --runtime python3.12
  cd my-fn-raff && raff deploy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			runtime := flagRuntime
			if runtime == "" {
				switch filepath.Ext(args[0]) {
				case ".py":
					runtime = "python3.12"
				case ".js", ".mjs", ".ts":
					runtime = "nodejs20.x"
				case ".go":
					runtime = "go1.x"
				default:
					return fmt.Errorf("could not detect the Lambda runtime — pass --runtime (python3.12|nodejs20.x|go1.x)")
				}
			}

			// The AI rewrite takes a while — long client timeout.
			c, err := newClientWithTimeout(4 * time.Minute)
			if err != nil {
				return err
			}

			if flagAsIs {
				fmt.Println("Wrapping your handler with the run-as-is adapter (code unchanged)…")
			} else {
				fmt.Println("Converting — rewriting the handler usually takes 10–30 seconds…")
			}
			mode := "rewrite"
			if flagAsIs {
				mode = "adapter"
			}
			result, _, err := c.Functions.ConvertLambda(context.Background(), &raff.ConvertLambdaRequest{
				LambdaRuntime: runtime,
				Handler:       flagHandler,
				Mode:          mode,
				Files: []raff.SourceFile{
					{Name: filepath.Base(args[0]), Content: string(source)},
				},
			})
			if err != nil {
				return err
			}

			outDir := flagOut
			if outDir == "" {
				base := strings.TrimSuffix(filepath.Base(args[0]), filepath.Ext(args[0]))
				outDir = base + "-raff"
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			for _, f := range result.ConvertedFiles {
				if err := os.WriteFile(filepath.Join(outDir, f.Name), []byte(f.Content), 0o644); err != nil {
					return err
				}
				fmt.Printf("  wrote %s/%s\n", outDir, f.Name)
			}
			if err := os.WriteFile(filepath.Join(outDir, "raff.toml"), []byte(result.RaffToml), 0o644); err != nil {
				return err
			}
			fmt.Printf("  wrote %s/raff.toml\n", outDir)

			if len(result.Warnings) > 0 {
				fmt.Println("\nReview before deploying:")
				for _, w := range result.Warnings {
					fmt.Printf("  ! %s\n", w)
				}
			}
			if len(result.ProposedTriggers) > 0 {
				fmt.Println("\nSchedules detected — add as cron triggers in the dashboard:")
				for _, t := range result.ProposedTriggers {
					fmt.Printf("  → %s\n", t)
				}
			}
			if len(result.OutOfScope) > 0 {
				fmt.Println("\nNot migrated (and what to do instead):")
				for _, o := range result.OutOfScope {
					fmt.Printf("  · %s\n", o)
				}
			}
			fmt.Printf("\nNext: review the code, then\n  cd %s && raff deploy\n", outDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagRuntime, "runtime", "", "Lambda runtime (python3.12|nodejs20.x|go1.x; detected from extension)")
	cmd.Flags().StringVar(&flagHandler, "handler", "", "Lambda handler, e.g. lambda_function.handler")
	cmd.Flags().StringVar(&flagOut, "out", "", "Output directory (default <file>-raff)")
	cmd.Flags().BoolVar(&flagAsIs, "as-is", false, "Run the handler unchanged via the adapter shim (no AI rewrite)")
	return cmd
}
