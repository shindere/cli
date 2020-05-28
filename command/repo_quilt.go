package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	repoCmd.AddCommand(repoQuiltCmd)
}

var repoQuiltCmd = &cobra.Command{
	Use:   "quilt",
	Short: "A unique piece of art derived from git history",
	RunE:  repoQuilt,
}

func repoQuilt(cmd *cobra.Command, args []string) error {
	fmt.Println("hi")
	return nil
}
