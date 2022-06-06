package cmd

import (
	"context"

	"github.com/erwinvaneyk/cobras"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type WorkloadDeleteOptions struct {
	*WorkloadOptions
	workloadID []string
}

func NewCmdWorkloadDelete(workloadOpts *WorkloadOptions) *cobra.Command {
	opts := &WorkloadDeleteOptions{
		WorkloadOptions: workloadOpts,
	}

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a Cox workload",
		Run:   cobras.Run(opts),
		Args:  cobra.MinimumNArgs(1),
	}

	return cmd
}

func (o *WorkloadDeleteOptions) Complete(cmd *cobra.Command, args []string) error {
	o.workloadID = args
	return nil
}

func (o *WorkloadDeleteOptions) Validate() error {
	return nil
}

func (o *WorkloadDeleteOptions) Run(ctx context.Context) error {
	log := zap.S()
	client, err := createClientFromEnv()
	if err != nil {
		return err
	}

	for _, workloadID := range o.workloadID {
		_, err := client.DeleteWorkload(workloadID)
		if err != nil {
			log.Errorf("Failed to delete workload '%s': %v", workloadID, err)
			continue
		}
		log.Infof("Deleted workload '%s'", workloadID)
	}

	return nil
}
