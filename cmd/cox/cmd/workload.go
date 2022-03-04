package cmd

import (
	"github.com/spf13/cobra"
)

type WorkloadOptions struct {
	*RootOptions
}

func NewCmdWorkload(rootOptions *RootOptions) *cobra.Command {
	opts := &WorkloadOptions{
		RootOptions: rootOptions,
	}

	cmd := &cobra.Command{
		Use:     "workload",
		Short:   "Interact with Cox workloads.",
		Aliases: []string{"workloads"},
	}

	cmd.AddCommand(NewCmdWorkloadList(opts))
	cmd.AddCommand(NewCmdWorkloadDelete(opts))

	return cmd
}
