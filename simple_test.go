package main

import (
	"aliyun-pipelines-tui/internal/api"
	"fmt"
	"log"
	"os"
)

func main() {
	// Enable debug mode
	os.Setenv("FLOWT_DEBUG", "1")

	// Use hardcoded values for testing
	endpoint := "openapi-rdc.aliyuncs.com"
	token := "pt-qIvFwQHJT9vmIxs7ZQJh6VGf_0147c6e0-0229-4b94-ac80-75c1dee84ccd"
	orgId := "64cb1574fb4dfc705becd94b"

	// Create API client
	client, err := api.NewClientWithToken(endpoint, token)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	fmt.Println("Testing RUNNING+WAITING pipeline filtering...")

	// Test status filtering
	statusList := []string{"RUNNING", "WAITING"}
	pipelines, err := client.ListPipelinesWithStatus(orgId, statusList)
	if err != nil {
		log.Fatalf("Error getting filtered pipelines: %v", err)
	}

	fmt.Printf("Found %d RUNNING+WAITING pipelines:\n", len(pipelines))
	for _, p := range pipelines {
		fmt.Printf("  - ID: %s, Name: %s, Status: %s, LastRunStatus: %s\n",
			p.PipelineID, p.Name, p.Status, p.LastRunStatus)
	}

	if len(pipelines) == 0 {
		fmt.Println("No RUNNING+WAITING pipelines found. This might be the issue!")

		// Let's also test getting all pipelines
		fmt.Println("\nTesting all pipelines...")
		allPipelines, err := client.ListPipelines(orgId)
		if err != nil {
			log.Fatalf("Error getting all pipelines: %v", err)
		}

		fmt.Printf("Found %d total pipelines:\n", len(allPipelines))
		for i, p := range allPipelines {
			if i < 5 { // Show first 5
				fmt.Printf("  - ID: %s, Name: %s, Status: %s, LastRunStatus: %s\n",
					p.PipelineID, p.Name, p.Status, p.LastRunStatus)
			}
		}
		if len(allPipelines) > 5 {
			fmt.Printf("  ... and %d more\n", len(allPipelines)-5)
		}
	}
}
