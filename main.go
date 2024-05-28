package main

import (
	"log"
	"os"
	"strings"

	"github.com/lep13/AutoBuildGo/services/ecr"
	"github.com/lep13/AutoBuildGo/services/gitsetup"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <repo-name> [\"optional description\"]")
	}
	repoName := os.Args[1]
	description := "Created from a template via automated setup" // Default description if none provided

	if len(os.Args) > 2 {
		description = strings.Join(os.Args[2:], " ") // Combine all arguments after repoName as description
	}

	// Create ECR Repository
	if err := ecr.CreateRepo(repoName); err != nil {
		log.Fatalf("Failed to create ECR repository: %v", err)
	}

	// Prepare the HTTP client and command executor
	httpClient := &gitsetup.RealHttpClient{}
	commandExecutor := &gitsetup.DefaultCommandExecutor{} // Make sure this is defined and exported

	// Create Git Repository
	config := gitsetup.DefaultRepoConfig(repoName, description)
	if err := gitsetup.CreateGitRepository(httpClient, config, commandExecutor); err != nil {
		log.Fatalf("Failed to create Git repository: %v", err)
	}

	log.Println("ECR and Git repositories created successfully")
}
