package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "bad"}

var otherCmd = &cobra.Command{Use: "other"}

func Execute() {
	fmt.Println(rootCmd, otherCmd)
}
