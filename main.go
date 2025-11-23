/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
// @title           Approval Gin API
// @version         1.0
// @description     Approval workflow API server based on approval-kit
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token from Keycloak
package main

import "github.com/mautops/approval-gin/cmd"

func main() {
	cmd.Execute()
}
