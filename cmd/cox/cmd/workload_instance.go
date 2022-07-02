package cmd

import (
	"github.com/spf13/cobra"
)

type WorkloadInstanceOptions struct {
	*WorkloadOptions
}

func NewCmdWorkloadInstance(workloadOptions *WorkloadOptions) *cobra.Command {
	opts := &WorkloadInstanceOptions{
		WorkloadOptions: workloadOptions,
	}

	cmd := &cobra.Command{
		Use:     "instance",
		Short:   "Interact with Cox workload instances.",
		Aliases: []string{"instances"},
	}

	cmd.AddCommand(NewCmdWorkloadInstanceList(opts))

	return cmd
}
