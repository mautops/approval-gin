/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "approval-gin",
	Short: "Approval workflow API server",
	Long: `Approval Gin is a REST API server for approval workflow management.
It provides complete API interfaces for approval templates and tasks,
based on the approval-kit core library.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 全局配置标志将在后续添加
}

// GetRootCmd 返回根命令（用于测试）
func GetRootCmd() *cobra.Command {
	return rootCmd
}


