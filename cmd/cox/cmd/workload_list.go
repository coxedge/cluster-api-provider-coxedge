package cmd

import (
	"context"
	"os"

	"github.com/erwinvaneyk/cobras"
	"github.com/olekukonko/tablewriter"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type WorkloadListOptions struct {
	*WorkloadOptions
	ShowWideInfo bool
}

func NewCmdWorkloadList(workloadOpts *WorkloadOptions) *cobra.Command {
	opts := &WorkloadListOptions{
		WorkloadOptions: workloadOpts,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display all Cox workloads",
		Run:   cobras.Run(opts),
	}

	cmd.Flags().BoolVar(&opts.ShowWideInfo, "wide", opts.ShowWideInfo, "Show more details about the workload.")

	return cmd
}

func (o *WorkloadListOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *WorkloadListOptions) Validate() error {
	return nil
}

func (o *WorkloadListOptions) Run(ctx context.Context) error {
	log := zap.S()
	client, err := createClientFromEnv()
	if err != nil {
		return err
	}

	log.Debug("Fetching all workloads")
	workloads, _, err := client.GetWorkloads()
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	if o.ShowWideInfo {
		table.SetHeader([]string{"ID", "NAME", "TYPE", "STATUS", "ANYCAST IP", "PUBLIC IP"})
	} else {
		table.SetHeader([]string{"ID", "NAME", "TYPE", "STATUS", "ANYCAST IP"})
	}
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetAutoWrapText(false)

	for _, workload := range workloads.Data {
		if o.ShowWideInfo {
			instances, _, err := client.GetInstances(workload.ID)
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
				instance.PublicIPAddress,
			})
		} else {
			table.Append([]string{
				workload.ID,
				workload.Name,
				workload.Type,
				workload.Status,
				workload.AnycastIPAddress,
			})
		}
	}
	table.Render()

	return nil
}
