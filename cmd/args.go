package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func usageError(cmd *cobra.Command, detail string) error {
	return fmt.Errorf("usage: %s\n\n%s", cmd.UseLine(), detail)
}

func exactArgs(n int, detail string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return usageError(cmd, detail)
		}
		return nil
	}
}

func maxArgs(n int, detail string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return usageError(cmd, detail)
		}
		return nil
	}
}

func rangeArgs(minCount, maxCount int, detail string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < minCount || len(args) > maxCount {
			return usageError(cmd, detail)
		}
		return nil
	}
}
