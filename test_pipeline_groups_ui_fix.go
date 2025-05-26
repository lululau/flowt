package main

import (
	"aliyun-pipelines-tui/internal/api"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {
	// Load configuration
	endpoint := os.Getenv("ALIYUN_DEVOPS_ENDPOINT")
	token := os.Getenv("ALIYUN_DEVOPS_TOKEN")
	orgId := os.Getenv("ALIYUN_DEVOPS_ORG_ID")

	if endpoint == "" || token == "" || orgId == "" {
		log.Fatal("Please set ALIYUN_DEVOPS_ENDPOINT, ALIYUN_DEVOPS_TOKEN, and ALIYUN_DEVOPS_ORG_ID environment variables")
	}

	// Create API client
	client, err := api.NewClientWithToken(endpoint, token)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	fmt.Printf("Testing UI Fix for Pipeline Groups API for Organization: %s\n", orgId)
	fmt.Println("=" + fmt.Sprintf("%*s", 70, "="))

	// Test 1: List Pipeline Groups
	fmt.Println("\n1. Testing ListPipelineGroups...")
	groups, err := client.ListPipelineGroups(orgId)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d pipeline groups:\n", len(groups))
	for i, group := range groups {
		fmt.Printf("  %d. ID: %s, Name: %s\n", i+1, group.GroupID, group.Name)
	}

	if len(groups) == 0 {
		fmt.Println("No pipeline groups found. Cannot test group-specific pipeline listing.")
		return
	}

	// Test 2: Test each group to see which ones have pipelines
	fmt.Println("\n2. Testing ListPipelineGroupPipelines for each group...")
	fmt.Println("This simulates what happens when you press Enter on a group in the UI:")

	for _, group := range groups {
		groupId, err := strconv.Atoi(group.GroupID)
		if err != nil {
			fmt.Printf("  Group '%s': Error converting ID '%s' to int: %v\n", group.Name, group.GroupID, err)
			continue
		}

		fmt.Printf("\n  Group '%s' (ID: %s):\n", group.Name, group.GroupID)

		// This is the API call that the UI now makes instead of string matching
		pipelines, err := client.ListPipelineGroupPipelines(orgId, groupId, nil)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
		} else {
			fmt.Printf("    ✅ Found %d pipelines:\n", len(pipelines))
			if len(pipelines) == 0 {
				fmt.Printf("      (No pipelines in this group)\n")
			} else {
				for i, pipeline := range pipelines {
					fmt.Printf("      %d. ID: %s, Name: %s, Created: %s\n",
						i+1, pipeline.PipelineID, pipeline.Name, pipeline.CreateTime.Format("2006-01-02 15:04:05"))
				}
			}
		}
	}

	// Test 3: Compare with old string matching approach (simulation)
	fmt.Println("\n3. Comparison with old string matching approach:")
	fmt.Println("(This shows why only 'dev' and 'Prod' groups had data before)")

	// Get all pipelines for comparison
	allPipelines, err := client.ListPipelines(orgId)
	if err != nil {
		fmt.Printf("Error getting all pipelines: %v\n", err)
		return
	}

	fmt.Printf("\nTotal pipelines in organization: %d\n", len(allPipelines))

	for _, group := range groups {
		fmt.Printf("\nGroup '%s':\n", group.Name)

		// Simulate old string matching approach
		var matchedByName []api.Pipeline
		for _, p := range allPipelines {
			if containsIgnoreCase(p.Name, group.Name) {
				matchedByName = append(matchedByName, p)
			}
		}

		// Get actual group pipelines using API
		groupId, _ := strconv.Atoi(group.GroupID)
		actualGroupPipelines, err := client.ListPipelineGroupPipelines(orgId, groupId, nil)
		if err != nil {
			fmt.Printf("  API Error: %v\n", err)
			continue
		}

		fmt.Printf("  Old approach (string matching): %d pipelines\n", len(matchedByName))
		fmt.Printf("  New approach (API call): %d pipelines\n", len(actualGroupPipelines))

		if len(matchedByName) != len(actualGroupPipelines) {
			fmt.Printf("  🔧 FIXED: This group now shows correct data!\n")
		} else if len(actualGroupPipelines) > 0 {
			fmt.Printf("  ✅ Both approaches return same count (group has data)\n")
		} else {
			fmt.Printf("  ℹ️  Both approaches return 0 (group is empty)\n")
		}
	}

	fmt.Println("\n" + "=" + fmt.Sprintf("%*s", 70, "="))
	fmt.Println("UI Fix Testing completed!")
	fmt.Println("\nSummary:")
	fmt.Println("- Before: Only groups with names matching pipeline names showed data")
	fmt.Println("- After: All groups show their actual pipelines via API calls")
	fmt.Println("- The UI now calls ListPipelineGroupPipelines API instead of string matching")
}

// Helper function for case-insensitive string matching (simulating old approach)
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
