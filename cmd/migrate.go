/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"

	"github.com/mautops/approval-gin/internal/config"
	"github.com/mautops/approval-gin/internal/database"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long: `Run database migrations to create or update database schema.
This command will:
- Create all required tables if they don't exist
- Update table schemas if needed
- Create indexes for optimal query performance

The command uses the database configuration from the config file or environment variables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 加载配置
		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// 2. 连接数据库
		log.Printf("Connecting to database: %s@%s:%d/%s", 
			cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		db, err := database.Connect(cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect database: %w", err)
		}
		defer func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}()

		// 3. 执行迁移
		log.Println("Running database migrations...")
		if err := database.Migrate(db); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		log.Println("Database migrations completed successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// 添加配置标志
	migrateCmd.Flags().String("config", "", "Config file path (default: search in current directory, ./config, or $HOME/.approval-gin)")
}

