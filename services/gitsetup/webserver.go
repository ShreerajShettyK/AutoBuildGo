package gitsetup

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lep13/AutoBuildGo/services/ecr"
)

// Wrapper variables for external dependencies
var (
	CreateECRClientFunc  = ecr.CreateECRClient
	CreateRepoFunc       = ecr.CreateRepo
	NewGitClientFunc     = NewGitClient
	LoadEnvFunc          = LoadEnv
	CloneAndPushRepoFunc = CloneAndPushRepo
)

type RepoRequest struct {
	RepoName    string `json:"repo_name"`
	Description string `json:"description"`
}

func HandleWebServer() {
	http.HandleFunc("/create-repo", CreateRepoHandler)
	log.Println("Server is starting on :8082...")
	err := http.ListenAndServe(":8082", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func CreateRepoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("CreateRepoHandler invoked")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.RepoName == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	description := req.Description
	if description == "" {
		description = "Created from a template via automated setup"
	}

	// Use the wrapper function to create ECR client
	ecrClient, err := CreateECRClientFunc()
	if err != nil {
		http.Error(w, "Failed to create ECR client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Use the wrapper function to create ECR Repository
	if err := CreateRepoFunc(req.RepoName, ecrClient); err != nil {
		http.Error(w, "Failed to create ECR repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Use the wrapper function to load environment variables
	LoadEnvFunc()

	// Use the wrapper function to create Git Repository
	config := DefaultRepoConfig(req.RepoName, description)
	gitClient := NewGitClientFunc() // Create an instance of GitClient

	if err := gitClient.CreateGitRepository(config); err != nil {
		http.Error(w, "Failed to create Git repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 20 second time delay
	time.Sleep(20 * time.Second)

	// Use the wrapper function to clone and push the repository
	if err := CloneAndPushRepoFunc(req.RepoName); err != nil {
		http.Error(w, "Failed to clone and push repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ECR and Git repositories created successfully"))
}