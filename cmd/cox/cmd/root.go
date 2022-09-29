package cmd

import (
	"context"
	"net/http"
	"os"

	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge/scope"
	"github.com/erwinvaneyk/cobras"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type RootOptions struct {
	Debug bool
}

func NewCmdRoot() *cobra.Command {
	opts := &RootOptions{
		Debug: os.Getenv("COX_DEBUG") != "",
	}

	cmd := &cobra.Command{
		Use:              "cox",
		Short:            "CLI for interacting with Cox services",
		PersistentPreRun: cobras.Run(opts),
	}

	cmd.PersistentFlags().BoolVar(&opts.Debug, "debug", opts.Debug, "More logs. [COX_DEBUG]")

	cmd.AddCommand(NewCmdWorkload(opts))

	return cmd
}

func (o *RootOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *RootOptions) Validate() error {
	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	// Configure the logging and its verbosity
	setupLogging(o.Debug)

	return nil
}

func Execute() {
	if err := NewCmdRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func setupLogging(debug bool) {
	zapCfg := zap.NewDevelopmentConfig()
	if !debug {
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := zapCfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func createClientFromEnv() (*coxedge.Client, error) {
	creds, err := scope.ParseFromEnv()
	if err != nil {
		return nil, err
	}
	return coxedge.NewClient(creds.CoxAPIBaseURL, creds.CoxService, creds.CoxEnvironment, creds.CoxAPIKey, creds.CoxOrganization, http.DefaultClient)
}
