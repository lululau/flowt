package main

import (
	"aliyun-pipelines-tui/internal/api" // Import the api package
	"aliyun-pipelines-tui/internal/ui" // Local package for UI components
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	// Read environment variables
	accessKeyId := os.Getenv("ALICLOUD_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")
	orgId := os.Getenv("ALICLOUD_DEVOPS_ORG_ID")
	regionId := os.Getenv("ALICLOUD_REGION_ID")

	// Validate required environment variables
	if accessKeyId == "" {
		fmt.Fprintln(os.Stderr, "Error: ALICLOUD_ACCESS_KEY_ID environment variable is not set.")
		os.Exit(1)
	}
	if accessKeySecret == "" {
		fmt.Fprintln(os.Stderr, "Error: ALICLOUD_ACCESS_KEY_SECRET environment variable is not set.")
		os.Exit(1)
	}
	if orgId == "" {
		fmt.Fprintln(os.Stderr, "Error: ALICLOUD_DEVOPS_ORG_ID environment variable is not set.")
		os.Exit(1)
	}

	// Use default region if not set
	if regionId == "" {
		regionId = "cn-hangzhou" // Default region
	}

	// Initialize API client
	apiClient, err := api.NewClient(accessKeyId, accessKeySecret, regionId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing API client: %v\n", err)
		os.Exit(1)
	}

	// Set transparent background style
	tcell.StyleDefault = tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorDefault)

	// Initialize tview.Application
	app := tview.NewApplication()

	// Create the main view (Pages) using ui.NewMainView()
	mainPages := ui.NewMainView(app, apiClient, orgId) // Pass apiClient and orgId

	// Set up global input capture for 'q' and Ctrl+C to stop the application
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				app.Stop()
				return nil
			}
		}
		return event
	})

	// Set the root of the application and run
	if err := app.SetRoot(mainPages, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
