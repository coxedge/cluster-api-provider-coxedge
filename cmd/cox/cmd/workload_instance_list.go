package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/erwinvaneyk/cobras"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var workloadInstanceListValidOutputFormats = []string{
	"table",
	"name",
}

type WorkloadInstanceListOptions struct {
	*WorkloadInstanceOptions
	OutputFormat string
	WorkloadID   string
}

func NewCmdWorkloadInstanceList(workloadOpts *WorkloadInstanceOptions) *cobra.Command {
	opts := &WorkloadInstanceListOptions{
		WorkloadInstanceOptions: workloadOpts,
		OutputFormat:            "table",
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display all Cox workloads",
		Run:   cobras.Run(opts),
	}

	cmd.Flags().StringVarP(&opts.WorkloadID, "workload", "w", opts.WorkloadID, "The workload of which to list the instances.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", opts.OutputFormat, "Output format. options: "+strings.Join(workloadInstanceListValidOutputFormats, ","))

	return cmd
}

func (o *WorkloadInstanceListOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *WorkloadInstanceListOptions) Validate() error {
	if len(o.WorkloadID) == 0 {
		return errors.New("workload ID required")
	}
	if !slices.Contains(workloadInstanceListValidOutputFormats, o.OutputFormat) {
		return fmt.Errorf("unknown output format: %s", o.OutputFormat)
	}

	return nil
}

func (o *WorkloadInstanceListOptions) Run(ctx context.Context) error {
	log := zap.S()
	client, err := createClientFromEnv()
	if err != nil {
		return err
	}

	log.Debug("Fetching all instances of the provided workload")
	instances, err := client.GetInstances(o.WorkloadID)
	if err != nil {
		return err
	}

	switch o.OutputFormat {
	case "name":
		for _, workload := range instances.Data {
			fmt.Println(workload.Name)
		}
	case "table":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "TYPE", "STATUS", "PUBLIC IP"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		for _, instance := range instances.Data {
			table.Append([]string{
				instance.ID,
				instance.Type,
				instance.Status,
				instance.PublicIPAddress,
			})
		}
		table.Render()
	default:
		return fmt.Errorf("unknown output format: %s", o.OutputFormat)
	}
	return nil
}
