package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/erwinvaneyk/cobras"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var workloadListValidOutputFormats = []string{
	"table",
	"wide",
	"name",
}

type WorkloadListOptions struct {
	*WorkloadOptions
	OutputFormat string
}

func NewCmdWorkloadList(workloadOpts *WorkloadOptions) *cobra.Command {
	opts := &WorkloadListOptions{
		WorkloadOptions: workloadOpts,
		OutputFormat:    "table",
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display all Cox workloads",
		Run:   cobras.Run(opts),
	}

	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", opts.OutputFormat, "Output format. options: "+strings.Join(workloadListValidOutputFormats, ","))

	return cmd
}

func (o *WorkloadListOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *WorkloadListOptions) Validate() error {
	if !slices.Contains(workloadListValidOutputFormats, o.OutputFormat) {
		return fmt.Errorf("unknown output format: %s", o.OutputFormat)
	}
	return nil
}

func (o *WorkloadListOptions) Run(ctx context.Context) error {
	log := zap.S()
	client, err := createClientFromEnv()
	if err != nil {
		return err
	}

	log.Debug("Fetching all workloads")
	workloads, err := client.GetWorkloads()
	if err != nil {
		return err
	}

	switch o.OutputFormat {
	case "name":
		for _, workload := range workloads.Data {
			fmt.Println(workload.Name)
		}
	case "wide":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "NAME", "TYPE", "STATUS", "ANYCAST IP", "INSTANCE STATUS", "PUBLIC IP"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		for _, workload := range workloads.Data {
			instances, err := client.GetInstances(workload.ID)
			if err != nil {
				return err
			}
			// Just assume a single instance for now, because we do not use multiple instances
			var instance coxedge.InstanceData
			if len(instances.Data) > 0 {
				instance = instances.Data[0]
			}

			table.Append([]string{
				workload.ID,
				workload.Name,
				workload.Type,
				workload.Status,
				workload.AnycastIPAddress,
				instance.Status,
				instance.PublicIPAddress,
			})
		}
		table.Render()
	case "table":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "NAME", "TYPE", "STATUS", "ANYCAST IP"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		for _, workload := range workloads.Data {
			table.Append([]string{
				workload.ID,
				workload.Name,
				workload.Type,
				workload.Status,
				workload.AnycastIPAddress,
			})
		}
		table.Render()
	default:
		return fmt.Errorf("unknown output format: %s", o.OutputFormat)
	}
	return nil
}
