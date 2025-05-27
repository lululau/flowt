package ui

import (
	"aliyun-pipelines-tui/internal/api"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// fuzzyMatch performs fuzzy matching between query and text
// Returns true if all characters in query appear in text in order (case-insensitive)
func fuzzyMatch(query, text string) bool {
	if query == "" {
		return true
	}

	query = strings.ToLower(query)
	text = strings.ToLower(text)

	queryIndex := 0
	for _, char := range text {
		if queryIndex < len(query) && rune(query[queryIndex]) == char {
			queryIndex++
		}
	}

	return queryIndex == len(query)
}

// SetGlobalConfig sets the global editor and pager commands
func SetGlobalConfig(editorCmd, pagerCmd string) {
	globalEditorCmd = editorCmd
	globalPagerCmd = pagerCmd
}

// OpenInEditor opens the given text content in the configured editor
func OpenInEditor(content string, app *tview.Application) error {
	if globalEditorCmd == "" {
		return fmt.Errorf("no editor configured")
	}

	// Create a temporary file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("flowt_logs_%d.txt", time.Now().Unix()))

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Use app.Suspend with proper terminal restoration
	var cmdErr error
	app.Suspend(func() {
		// Parse editor command (might have arguments)
		cmdParts := strings.Fields(globalEditorCmd)
		if len(cmdParts) == 0 {
			cmdErr = fmt.Errorf("invalid editor command")
			return
		}

		// Add the temporary file as the last argument
		cmdParts = append(cmdParts, tmpFile)

		// Open in editor with proper terminal handling
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run editor and wait for it to complete
		cmdErr = cmd.Run()

		// Reset terminal after editor exits
		resetCmd := exec.Command("reset")
		resetCmd.Stdout = os.Stdout
		resetCmd.Stderr = os.Stderr
		resetCmd.Run()
	})

	// Clean up temp file after editor closes
	os.Remove(tmpFile)

	if cmdErr != nil {
		return fmt.Errorf("editor command failed: %w", cmdErr)
	}

	return nil
}

// OpenInPager opens the given text content in the configured pager
func OpenInPager(content string, app *tview.Application) error {
	if globalPagerCmd == "" {
		return fmt.Errorf("no pager configured")
	}

	// Create a temporary file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("flowt_logs_%d.txt", time.Now().Unix()))

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Use app.Suspend with proper terminal restoration
	var cmdErr error
	app.Suspend(func() {
		// Parse pager command (might have arguments)
		cmdParts := strings.Fields(globalPagerCmd)
		if len(cmdParts) == 0 {
			cmdErr = fmt.Errorf("invalid pager command")
			return
		}

		// Add the temporary file as the last argument
		cmdParts = append(cmdParts, tmpFile)

		// Open in pager with proper terminal handling
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run pager and wait for it to complete
		cmdErr = cmd.Run()

		// Reset terminal after pager exits
		resetCmd := exec.Command("reset")
		resetCmd.Stdout = os.Stdout
		resetCmd.Stderr = os.Stderr
		resetCmd.Run()
	})

	// Clean up temp file after pager closes
	os.Remove(tmpFile)

	if cmdErr != nil {
		return fmt.Errorf("pager command failed: %w", cmdErr)
	}

	return nil
}

var (
	allPipelines      []api.Pipeline
	allPipelineGroups []api.PipelineGroup
	currentViewMode   string // "all_pipelines", "group_list", "pipelines_in_group"
	selectedGroupID   string
	selectedGroupName string

	currentSearchQuery      string
	currentGroupSearchQuery string // New: search query for groups

	// Status filtering
	showOnlyRunningWaiting bool // Toggle between all pipelines and RUNNING+WAITING only

	// Global configuration for editor and pager
	globalEditorCmd string
	globalPagerCmd  string

	// Maps to store references for table rows
	pipelineRowMap = make(map[int]*api.Pipeline)
	groupRowMap    = make(map[int]*api.PipelineGroup)

	// New state variables for current run and log view
	currentRunID            string
	currentPipelineIDForRun string
	currentPipelineName     string
	currentRunStatus        string // Current run status for status bar
	isLogViewActive         bool
	isRunHistoryActive      bool
	logViewTextView         *tview.TextView
	logStatusBar            *tview.TextView    // Status bar for log view
	logPage                 *tview.Flex        // Flex layout for the log page
	runHistoryTable         *tview.Table       // Table for pipeline run history
	runHistoryPage          *tview.Flex        // Flex layout for the run history page
	pipelineTableGlobal     *tview.Table       // To allow focus from modal
	groupTableGlobal        *tview.Table       // For group list table
	groupSearchInputGlobal  *tview.InputField  // New: search input for groups
	mainPagesGlobal         *tview.Pages       // To allow modal to be added/removed
	appGlobal               *tview.Application // For setting focus from modal

	// Maps to store run history references
	runHistoryRowMap = make(map[int]*api.PipelineRun)

	// Pagination state for run history
	currentRunHistoryPage = 1
	runHistoryPerPage     = 30
	totalRunHistoryPages  = 1
	currentRunHistoryData []api.PipelineRun
)

// ShowModal displays a modal dialog.
func ShowModal(title, text string, buttons []string, doneFunc func(buttonIndex int, buttonLabel string)) {
	if mainPagesGlobal == nil || appGlobal == nil {
		// Should not happen if app is initialized properly
		return
	}
	modal := tview.NewModal()
	modal.SetText(text)
	modal.SetTitle(title)
	modal.AddButtons(buttons)

	// Set transparent background for modal
	modal.SetBackgroundColor(tcell.ColorDefault)
	modal.SetTextColor(tcell.ColorWhite)
	modal.SetButtonBackgroundColor(tcell.ColorDefault)
	modal.SetButtonTextColor(tcell.ColorWhite)
	modal.SetBorderColor(tcell.ColorWhite)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		HideModal() // Hide modal first
		if doneFunc != nil {
			doneFunc(buttonIndex, buttonLabel)
		}
	})
	mainPagesGlobal.AddPage("modal", modal, true, true)
	appGlobal.SetFocus(modal)
}

// HideModal removes the modal dialog.
func HideModal() {
	if mainPagesGlobal == nil || appGlobal == nil {
		// Should not happen
		return
	}
	mainPagesGlobal.RemovePage("modal")
	// Try to restore focus to a sensible default, like the pipeline table
	if pipelineTableGlobal != nil && (currentViewMode == "all_pipelines" || currentViewMode == "pipelines_in_group") {
		appGlobal.SetFocus(pipelineTableGlobal)
	} else if currentViewMode == "group_list" && groupTableGlobal != nil {
		appGlobal.SetFocus(groupTableGlobal)
	}
}

// formatTime formats time for display in table
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// getStatusColor returns color for status display
func getStatusColor(status string) tcell.Color {
	switch strings.ToUpper(status) {
	case "SUCCESS":
		return tcell.ColorGreen
	case "RUNNING":
		return tcell.ColorYellow
	case "FAILED":
		return tcell.ColorRed
	case "CANCELED":
		return tcell.ColorOrange
	default:
		return tcell.ColorWhite
	}
}

// updateLogStatusBar updates the status bar in the log view
func updateLogStatusBar() {
	if logStatusBar == nil {
		return
	}

	var statusText string
	var statusColor tcell.Color

	switch strings.ToUpper(currentRunStatus) {
	case "RUNNING":
		statusText = fmt.Sprintf("Status: [green]%s[-] | Auto-refresh: ON", currentRunStatus)
		statusColor = tcell.ColorDefault
	case "SUCCESS":
		statusText = fmt.Sprintf("Status: [white]%s[-] | Auto-refresh: %s", currentRunStatus, getAutoRefreshStatus())
		statusColor = tcell.ColorDefault
	case "FAILED":
		statusText = fmt.Sprintf("Status: [red]%s[-] | Auto-refresh: %s", currentRunStatus, getAutoRefreshStatus())
		statusColor = tcell.ColorDefault
	case "CANCELED":
		statusText = fmt.Sprintf("Status: [gray]%s[-] | Auto-refresh: %s", currentRunStatus, getAutoRefreshStatus())
		statusColor = tcell.ColorDefault
	default:
		statusText = fmt.Sprintf("Status: [white]%s[-] | Auto-refresh: ON", currentRunStatus)
		statusColor = tcell.ColorDefault
	}

	logStatusBar.SetText(statusText)
	logStatusBar.SetTextColor(statusColor)
}

// getAutoRefreshStatus returns the current auto-refresh status text
func getAutoRefreshStatus() string {
	if pipelineFinished {
		remaining := 3 - finishedRefreshCount
		if remaining > 0 {
			return fmt.Sprintf("ON (%d more)", remaining)
		} else {
			return "OFF"
		}
	}
	return "ON"
}

// updatePipelineTable filters and updates the pipeline table widget.
func updatePipelineTable(table *tview.Table, app *tview.Application, _ *tview.InputField, apiClient *api.Client, orgId string) {
	table.Clear()
	pipelineTableGlobal = table // Update global reference

	var title string
	if currentViewMode == "pipelines_in_group" {
		if showOnlyRunningWaiting {
			title = fmt.Sprintf("Pipelines in '%s' (RUNNING+WAITING)", selectedGroupName)
		} else {
			title = fmt.Sprintf("Pipelines in '%s'", selectedGroupName)
		}
	} else {
		if showOnlyRunningWaiting {
			title = "Pipelines (RUNNING+WAITING)"
		} else {
			title = "All Pipelines"
		}
	}
	table.SetTitle(title)

	// Set table headers - only ID and Name
	headers := []string{"ID", "Pipeline Name"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(0, col, cell)
	}

	// 1. Get pipelines based on current view mode and status filter
	var tempFilteredByGroup []api.Pipeline
	if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
		// Use the correct API to get pipelines in the selected group
		groupIdInt := 0
		if _, err := fmt.Sscanf(selectedGroupID, "%d", &groupIdInt); err != nil {
			// Show error if group ID is invalid
			cell := tview.NewTableCell(fmt.Sprintf("Error: Invalid group ID '%s'", selectedGroupID)).
				SetTextColor(tcell.ColorRed).
				SetAlign(tview.AlignCenter)
			table.SetCell(1, 0, cell)
			table.SetCell(1, 1, tview.NewTableCell(""))
			return
		}

		// Call the ListPipelineGroupPipelines API
		groupPipelines, err := apiClient.ListPipelineGroupPipelines(orgId, groupIdInt, nil)
		if err != nil {
			// Show error message
			cell := tview.NewTableCell(fmt.Sprintf("Error fetching group pipelines: %v", err)).
				SetTextColor(tcell.ColorRed).
				SetAlign(tview.AlignCenter)
			table.SetCell(1, 0, cell)
			table.SetCell(1, 1, tview.NewTableCell(""))
			return
		}
		tempFilteredByGroup = groupPipelines
	} else {
		// Use all pipelines for "all_pipelines" view, with status filtering if enabled
		if showOnlyRunningWaiting {
			// Fetch pipelines with status filter
			statusList := []string{"RUNNING", "WAITING"}
			filteredPipelines, err := apiClient.ListPipelinesWithStatus(orgId, statusList)
			if err != nil {
				// Show error message
				cell := tview.NewTableCell(fmt.Sprintf("Error fetching filtered pipelines: %v", err)).
					SetTextColor(tcell.ColorRed).
					SetAlign(tview.AlignCenter)
				table.SetCell(1, 0, cell)
				table.SetCell(1, 1, tview.NewTableCell(""))
				return
			}
			tempFilteredByGroup = filteredPipelines
		} else {
			// Use cached all pipelines
			tempFilteredByGroup = append(tempFilteredByGroup, allPipelines...)
		}
	}

	// 2. Filter by search query (fuzzy search)
	tempFilteredBySearch := make([]api.Pipeline, 0)
	if currentSearchQuery != "" {
		for _, p := range tempFilteredByGroup {
			if fuzzyMatch(currentSearchQuery, p.Name) || fuzzyMatch(currentSearchQuery, p.PipelineID) {
				tempFilteredBySearch = append(tempFilteredBySearch, p)
			}
		}
	} else {
		tempFilteredBySearch = append(tempFilteredBySearch, tempFilteredByGroup...)
	}

	// Final filtered pipelines (no status filtering)
	finalFilteredPipelines := tempFilteredBySearch

	// Clear the pipeline row map
	pipelineRowMap = make(map[int]*api.Pipeline)

	// Populate the table
	if len(finalFilteredPipelines) == 0 {
		// Show "no data" message
		cell := tview.NewTableCell("No pipelines match filters.").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter)
		table.SetCell(1, 0, cell)
		table.SetCell(1, 1, tview.NewTableCell(""))
	} else {
		for i, p := range finalFilteredPipelines {
			pipelineCopy := p // Important: capture range variable for reference
			row := i + 1      // +1 because row 0 is header

			// Store the pipeline object in our map
			pipelineRowMap[row] = &pipelineCopy

			// Pipeline ID (left column)
			idCell := tview.NewTableCell(pipelineCopy.PipelineID).
				SetTextColor(tcell.ColorLightBlue).
				SetAlign(tview.AlignLeft).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(row, 0, idCell)

			// Pipeline Name (right column)
			nameCell := tview.NewTableCell(pipelineCopy.Name).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(row, 1, nameCell)
		}
	}

	// Set column widths to fill the screen
	table.SetFixed(1, 0) // Fix header row
	if table.GetRowCount() > 1 {
		table.Select(1, 0) // Select first data row
	}
}

// updateGroupTable updates the group list table
func updateGroupTable(table *tview.Table, app *tview.Application) {
	table.Clear()
	groupTableGlobal = table // Update global reference

	table.SetTitle("Pipeline Groups")

	// Set table headers
	headers := []string{"Group Name", "Group ID"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(0, col, cell)
	}

	// Clear the group row map
	groupRowMap = make(map[int]*api.PipelineGroup)

	// Filter groups by search query (fuzzy search)
	filteredGroups := make([]api.PipelineGroup, 0)
	if currentGroupSearchQuery != "" {
		for _, g := range allPipelineGroups {
			if fuzzyMatch(currentGroupSearchQuery, g.Name) || fuzzyMatch(currentGroupSearchQuery, g.GroupID) {
				filteredGroups = append(filteredGroups, g)
			}
		}
	} else {
		filteredGroups = append(filteredGroups, allPipelineGroups...)
	}

	// Populate the table
	if len(filteredGroups) == 0 {
		// Show "no data" message
		cell := tview.NewTableCell("No pipeline groups match filters.").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter)
		table.SetCell(1, 0, cell)
		table.SetCell(1, 1, tview.NewTableCell(""))
	} else {
		for i, g := range filteredGroups {
			groupCopy := g // Important: capture range variable for reference
			row := i + 1   // +1 because row 0 is header

			// Store the group object in our map
			groupRowMap[row] = &groupCopy

			// Group Name
			nameCell := tview.NewTableCell(groupCopy.Name).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(row, 0, nameCell)

			// Group ID
			idCell := tview.NewTableCell(groupCopy.GroupID).
				SetTextColor(tcell.ColorLightBlue).
				SetAlign(tview.AlignLeft).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(row, 1, idCell)
		}
	}

	// Set column widths and selection
	table.SetFixed(1, 0) // Fix header row
	if table.GetRowCount() > 1 {
		table.Select(1, 0) // Select first data row
	}
}

// showRunPipelineDialog shows a dialog to collect branch information and run the pipeline
func showRunPipelineDialog(selectedPipeline *api.Pipeline, app *tview.Application, apiClient *api.Client, orgId string) {
	// First, try to get the latest run to pre-fill branch information and extract repository URLs
	var defaultBranch string = "master" // Default branch name
	var repositoryURLs map[string]string = make(map[string]string)

	// Try to get latest run information to extract branch and repository information from previous run
	go func() {
		latestRunInfo, err := apiClient.GetLatestPipelineRunInfo(orgId, selectedPipeline.PipelineID)
		if err == nil && latestRunInfo != nil && len(latestRunInfo.RepositoryURLs) > 0 {
			// Use repository information from the latest run
			repositoryURLs = latestRunInfo.RepositoryURLs
			// Use the first repository's branch as default
			for _, branch := range latestRunInfo.RepositoryURLs {
				defaultBranch = branch
				break
			}
		}

		app.QueueUpdateDraw(func() {
			showBranchInputDialog(selectedPipeline, app, apiClient, orgId, defaultBranch, repositoryURLs)
		})
	}()
}

// showBranchInputDialog shows an input dialog for branch selection
func showBranchInputDialog(selectedPipeline *api.Pipeline, app *tview.Application, apiClient *api.Client, orgId, defaultBranch string, repositoryURLs map[string]string) {
	// Create a form for branch input
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Run Pipeline: %s", selectedPipeline.Name))
	form.SetBackgroundColor(tcell.ColorDefault)

	// Add branch input field
	branchInput := ""
	form.AddInputField("Branch Name:", defaultBranch, 30, nil, func(text string) {
		branchInput = text
	})

	// Add buttons
	form.AddButton("Run", func() {
		if branchInput == "" {
			branchInput = defaultBranch
		}

		// Hide the form
		mainPagesGlobal.RemovePage("branch_input")

		// Prepare parameters for the pipeline run using the correct format
		// Build runningBranchs map with repository URLs from latest run
		runningBranchs := make(map[string]string)

		if len(repositoryURLs) > 0 {
			// Use repository URLs from the latest run
			for repoUrl := range repositoryURLs {
				runningBranchs[repoUrl] = branchInput
			}
		} else {
			// Fallback: use a placeholder repository URL
			// This should be replaced with actual repository detection logic
			runningBranchs["https://gitlab.example.com/default/repo.git"] = branchInput
		}

		// Convert runningBranchs to JSON string
		runningBranchsJSON, err := json.Marshal(runningBranchs)
		if err != nil {
			ShowModal("Error", fmt.Sprintf("Failed to prepare parameters: %v", err), []string{"OK"}, nil)
			return
		}

		params := map[string]string{
			"runningBranchs": string(runningBranchsJSON),
		}

		// Run the pipeline
		runPipelineWithBranch(selectedPipeline, app, apiClient, orgId, params, repositoryURLs)
	})

	form.AddButton("Cancel", func() {
		mainPagesGlobal.RemovePage("branch_input")
		app.SetFocus(pipelineTableGlobal)
	})

	// Set form styling
	form.SetButtonBackgroundColor(tcell.ColorDefault)
	form.SetButtonTextColor(tcell.ColorWhite)
	form.SetFieldBackgroundColor(tcell.ColorDefault)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetLabelColor(tcell.ColorWhite)

	// Add the form to pages and show it
	mainPagesGlobal.AddPage("branch_input", form, true, true)
	app.SetFocus(form)
}

// runPipelineWithBranch executes the pipeline with the specified branch parameters
func runPipelineWithBranch(selectedPipeline *api.Pipeline, app *tview.Application, apiClient *api.Client, orgId string, params map[string]string, repositoryURLs map[string]string) {
	currentPipelineIDForRun = selectedPipeline.PipelineID

	go func() { // Run in goroutine to avoid blocking UI
		// Extract branch name and repository info for display
		var branchInfo string
		var repoInfo string

		if runningBranchsParam, ok := params["runningBranchs"]; ok {
			var runningBranchs map[string]string
			if err := json.Unmarshal([]byte(runningBranchsParam), &runningBranchs); err == nil {
				if len(runningBranchs) > 0 {
					for repoUrl, branch := range runningBranchs {
						branchInfo = branch
						repoInfo = repoUrl
						break // Use the first repository for display
					}
				}
			}
		}

		if branchInfo == "" {
			branchInfo = "master"
		}

		app.QueueUpdateDraw(func() {
			logText := fmt.Sprintf("Initiating pipeline run for '%s'...\nBranch: %s\n", selectedPipeline.Name, branchInfo)
			if repoInfo != "" {
				logText += fmt.Sprintf("Repository: %s\n", repoInfo)
			}
			if logViewTextView != nil {
				logViewTextView.SetText(logText)
				mainPagesGlobal.SwitchToPage("logs")
				app.SetFocus(logViewTextView)
			}
		})

		runResponse, err := apiClient.RunPipeline(orgId, selectedPipeline.PipelineID, params)
		if err != nil {
			app.QueueUpdateDraw(func() {
				ShowModal("Error", fmt.Sprintf("Failed to run pipeline: %v", err), []string{"OK"}, nil)
			})
			return
		}
		currentRunID = runResponse.RunID
		currentRunStatus = "RUNNING" // New runs start as RUNNING
		isLogViewActive = true

		app.QueueUpdateDraw(func() {
			logText := fmt.Sprintf("Pipeline '%s' triggered successfully!\nRun ID: %s\nBranch: %s\n",
				selectedPipeline.Name, currentRunID, branchInfo)
			if repoInfo != "" {
				logText += fmt.Sprintf("Repository: %s\n", repoInfo)
			}
			logText += "Fetching run details...\n"
			if logViewTextView != nil {
				logViewTextView.SetText(logText)
				logViewTextView.ScrollToEnd()
			}
			// Update status bar for new run
			updateLogStatusBar()
		})

		// Start automatic log fetching and refreshing every 5 seconds
		startLogAutoRefresh(app, apiClient, orgId, selectedPipeline.Name, branchInfo, repoInfo)
	}()
}

// Global variables for log auto-refresh control
var (
	logRefreshTicker *time.Ticker
	logRefreshStop   chan bool
	// Variables for delayed auto-refresh stop
	finishedRefreshCount int  // Count of refreshes after pipeline finished
	pipelineFinished     bool // Whether pipeline has finished
)

// startLogAutoRefresh starts automatic log fetching and refreshing every 5 seconds
func startLogAutoRefresh(app *tview.Application, apiClient *api.Client, orgId, pipelineName, branchInfo, repoInfo string) {
	// Stop any existing refresh ticker
	stopLogAutoRefresh()

	// Reset delayed stop state
	finishedRefreshCount = 0
	pipelineFinished = false

	// Create new ticker and stop channel
	logRefreshTicker = time.NewTicker(5 * time.Second)
	logRefreshStop = make(chan bool, 1)

	// Start the refresh goroutine
	go func() {
		// Capture the channels locally to avoid race conditions
		ticker := logRefreshTicker
		stopChan := logRefreshStop

		// Defer cleanup to ensure resources are properly released
		defer func() {
			if ticker != nil {
				ticker.Stop()
			}
			// Close the stop channel if it's still open
			if stopChan != nil {
				// Check if channel is still open before closing
				select {
				case <-stopChan:
					// Channel already received a value, safe to close
				default:
					// Channel is empty, close it
					close(stopChan)
				}
			}
		}()

		// Initial log fetch
		fetchAndDisplayLogs(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)

		for {
			select {
			case <-ticker.C:
				// Only refresh if log view is still active
				if isLogViewActive {
					fetchAndDisplayLogs(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)
				} else {
					// Stop refreshing if log view is no longer active
					return
				}
			case <-stopChan:
				return
			}
		}
	}()
}

// stopLogAutoRefresh stops the automatic log refresh
func stopLogAutoRefresh() {
	// Stop ticker first
	if logRefreshTicker != nil {
		logRefreshTicker.Stop()
		logRefreshTicker = nil
	}

	// Send stop signal to goroutine if channel exists
	if logRefreshStop != nil {
		// Send stop signal in a non-blocking way
		select {
		case logRefreshStop <- true:
			// Signal sent successfully
		default:
			// Channel might be full or closed, that's ok
		}
		// Set to nil to prevent further access
		logRefreshStop = nil
	}
}

// fetchAndDisplayLogs fetches and displays the current logs for the running pipeline
func fetchAndDisplayLogs(app *tview.Application, apiClient *api.Client, orgId, pipelineName, branchInfo, repoInfo string) {
	if currentRunID == "" || currentPipelineIDForRun == "" {
		return
	}

	// Check if app is still valid
	if app == nil {
		return
	}

	// Fetch complete logs (this will internally get run details as well)
	logs, err := apiClient.GetPipelineRunLogs(orgId, currentPipelineIDForRun, currentRunID)

	// Use a safe update mechanism
	app.QueueUpdateDraw(func() {
		// Double-check that we're still in log view mode
		if !isLogViewActive {
			return
		}

		// Check if logViewTextView is still valid before updating
		if logViewTextView == nil {
			return
		}

		// Extract status from logs for status tracking
		var extractedStatus string = "RUNNING" // Default status
		if logs != "" {
			if strings.Contains(logs, "Status: SUCCESS") {
				extractedStatus = "SUCCESS"
			} else if strings.Contains(logs, "Status: FAILED") {
				extractedStatus = "FAILED"
			} else if strings.Contains(logs, "Status: CANCELED") {
				extractedStatus = "CANCELED"
			} else if strings.Contains(logs, "Status: RUNNING") {
				extractedStatus = "RUNNING"
			}
		}
		currentRunStatus = extractedStatus

		// Update status bar
		updateLogStatusBar()

		// Build the complete log display
		var logText strings.Builder

		// Header information
		logText.WriteString(fmt.Sprintf("Pipeline: %s\n", pipelineName))
		logText.WriteString(fmt.Sprintf("Run ID: %s\n", currentRunID))
		logText.WriteString(fmt.Sprintf("Branch: %s\n", branchInfo))
		if repoInfo != "" {
			logText.WriteString(fmt.Sprintf("Repository: %s\n", repoInfo))
		}

		// Add refresh timestamp
		logText.WriteString(fmt.Sprintf("Last Updated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
		logText.WriteString(strings.Repeat("=", 80) + "\n\n")

		// Add logs content
		if err != nil {
			logText.WriteString(fmt.Sprintf("Error fetching logs: %v\n\n", err))
			logText.WriteString("Note: Log fetching may require additional parameters or the pipeline may still be initializing.\n")
		} else if logs == "" {
			logText.WriteString("No logs available yet. The pipeline may still be starting...\n")
		} else {
			logText.WriteString(logs)
		}

		// Add footer with instructions
		logText.WriteString("\n" + strings.Repeat("=", 80) + "\n")
		logText.WriteString("Auto-refreshing every 5 seconds. Press 'r' to refresh manually, 'q' to return, 'e' to edit in editor, 'v' to view in pager.\n")

		// Handle delayed auto-refresh stop logic
		if extractedStatus == "SUCCESS" || extractedStatus == "FAILED" || extractedStatus == "CANCELED" {
			if !pipelineFinished {
				// Pipeline just finished
				pipelineFinished = true
				finishedRefreshCount = 0
			} else {
				// Pipeline was already finished, increment counter
				finishedRefreshCount++
				if finishedRefreshCount >= 3 {
					// Stop auto-refresh after 3 additional refreshes
					stopLogAutoRefresh()
				}
			}
		}

		// Final check before updating UI
		if logViewTextView != nil {
			logViewTextView.SetText(logText.String())
			logViewTextView.ScrollToEnd()
		}
	})
}

// updateRunHistoryTable updates the run history table for a specific pipeline
func updateRunHistoryTable(table *tview.Table, app *tview.Application, apiClient *api.Client, orgId, pipelineId, pipelineName string) {
	table.Clear()

	// Update title with pagination info
	title := fmt.Sprintf("Run History - %s (Page %d/%d) [/] to navigate, 0 to go to first page",
		pipelineName, currentRunHistoryPage, totalRunHistoryPages)
	table.SetTitle(title)

	// Set table headers
	headers := []string{"#", "Status", "Trigger", "Start Time", "Finish Time", "Duration"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(0, col, cell)
	}

	// Clear the run history row map
	runHistoryRowMap = make(map[int]*api.PipelineRun)

	// Fetch pipeline runs for current page
	runs, err := apiClient.ListPipelineRuns(orgId, pipelineId)
	if err != nil {
		// Show error message
		cell := tview.NewTableCell(fmt.Sprintf("Error fetching runs: %v", err)).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(1, 0, cell)
		for i := 1; i < len(headers); i++ {
			table.SetCell(1, i, tview.NewTableCell("").SetBackgroundColor(tcell.ColorDefault))
		}
		return
	}

	// Store all runs data for pagination
	currentRunHistoryData = runs

	// Calculate pagination
	totalRuns := len(runs)
	if totalRuns == 0 {
		totalRunHistoryPages = 1
	} else {
		totalRunHistoryPages = (totalRuns + runHistoryPerPage - 1) / runHistoryPerPage
	}

	// Ensure current page is valid
	if currentRunHistoryPage > totalRunHistoryPages {
		currentRunHistoryPage = totalRunHistoryPages
	}
	if currentRunHistoryPage < 1 {
		currentRunHistoryPage = 1
	}

	// Update title with correct pagination info
	title = fmt.Sprintf("Run History - %s (Page %d/%d) [/] to navigate, 0 to go to first page",
		pipelineName, currentRunHistoryPage, totalRunHistoryPages)
	table.SetTitle(title)

	if totalRuns == 0 {
		// Show "no data" message
		cell := tview.NewTableCell("No run history found.").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(1, 0, cell)
		for i := 1; i < len(headers); i++ {
			table.SetCell(1, i, tview.NewTableCell("").SetBackgroundColor(tcell.ColorDefault))
		}
		return
	}

	// Calculate start and end indices for current page
	startIdx := (currentRunHistoryPage - 1) * runHistoryPerPage
	endIdx := startIdx + runHistoryPerPage
	if endIdx > totalRuns {
		endIdx = totalRuns
	}

	// Get runs for current page
	pageRuns := runs[startIdx:endIdx]

	// Populate the table with runs
	for i, run := range pageRuns {
		runCopy := run // Important: capture range variable for reference
		row := i + 1   // +1 because row 0 is header

		// Store the run object in our map
		runHistoryRowMap[row] = &runCopy

		// Run number (reverse order, latest first) - adjust for pagination
		globalRunIndex := startIdx + i
		runNumCell := tview.NewTableCell(fmt.Sprintf("#%d", totalRuns-globalRunIndex)).
			SetTextColor(tcell.ColorLightBlue).
			SetAlign(tview.AlignCenter).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(1) // Minimal width
		table.SetCell(row, 0, runNumCell)

		// Status - make it more compact
		statusCell := tview.NewTableCell(runCopy.Status).
			SetTextColor(getStatusColor(runCopy.Status)).
			SetAlign(tview.AlignCenter).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(2) // Small width
		table.SetCell(row, 1, statusCell)

		// Trigger Mode - compact display
		triggerDisplay := runCopy.TriggerMode
		if len(triggerDisplay) > 10 {
			triggerDisplay = triggerDisplay[:10] + "..."
		}
		triggerCell := tview.NewTableCell(triggerDisplay).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(2) // Small width
		table.SetCell(row, 2, triggerCell)

		// Start Time - more space for timestamps
		startTimeCell := tview.NewTableCell(formatTime(runCopy.StartTime)).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(3) // More width for timestamps
		table.SetCell(row, 3, startTimeCell)

		// Finish Time - more space for timestamps
		finishTimeCell := tview.NewTableCell(formatTime(runCopy.FinishTime)).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(3) // More width for timestamps
		table.SetCell(row, 4, finishTimeCell)

		// Duration - compact
		var duration string
		if !runCopy.StartTime.IsZero() && !runCopy.FinishTime.IsZero() {
			dur := runCopy.FinishTime.Sub(runCopy.StartTime)
			if dur > time.Hour {
				duration = fmt.Sprintf("%.1fh", dur.Hours())
			} else if dur > time.Minute {
				duration = fmt.Sprintf("%.1fm", dur.Minutes())
			} else {
				duration = fmt.Sprintf("%.0fs", dur.Seconds())
			}
		} else if !runCopy.StartTime.IsZero() {
			// Running or incomplete
			duration = "Running..."
		} else {
			duration = "-"
		}
		durationCell := tview.NewTableCell(duration).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignRight).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(1) // Minimal width
		table.SetCell(row, 5, durationCell)
	}

	// Set column widths to fill the screen
	table.SetFixed(1, 0) // Fix header row
	if table.GetRowCount() > 1 {
		table.Select(1, 0) // Select first data row
	}
}

// NewMainView creates the main layout for the application.
func NewMainView(app *tview.Application, apiClient *api.Client, orgId string) tview.Primitive {
	// Force default background color for primitives to handle potential InputField empty background issue
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	// Initialize global references for modal helpers
	appGlobal = app

	currentViewMode = "all_pipelines"
	currentSearchQuery = ""
	currentGroupSearchQuery = ""
	selectedGroupID = ""
	selectedGroupName = ""
	showOnlyRunningWaiting = false
	isLogViewActive = false
	isRunHistoryActive = false

	var fetchErrPipelines error
	allPipelines, fetchErrPipelines = apiClient.ListPipelines(orgId)

	var fetchErrGroups error
	allPipelineGroups, fetchErrGroups = apiClient.ListPipelineGroups(orgId)

	// UI Elements
	pipelineTable := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	pipelineTable.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
	// Enable table to receive focus and handle input
	pipelineTable.SetSelectable(true, false)
	pipelineTableGlobal = pipelineTable // Set global reference

	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetPlaceholder("Pipeline name/ID (/ to focus)...").
		SetFieldWidth(0)
	searchInput.SetBackgroundColor(tcell.ColorDefault)      // Overall background of the box
	searchInput.SetFieldBackgroundColor(tcell.ColorDefault) // Background of the text entry area
	searchInput.SetLabelColor(tcell.ColorWhite)             // Color of the "Search: " label
	searchInput.SetFieldTextColor(tcell.ColorWhite)         // Color of the text as you type
	searchInput.SetPlaceholderTextColor(tcell.ColorGray)    // Color of the placeholder text

	// Explicitly set the style for the field itself
	fieldStyle := tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorWhite)
	searchInput.SetFieldStyle(fieldStyle)

	// Help info
	helpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=run history, r=run, a=toggle filter, Ctrl+G=groups, /=search, q=back, Q=quit").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGray)
	helpInfo.SetBackgroundColor(tcell.ColorDefault)

	pipelineListFlexView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 1, 1, false).
		AddItem(pipelineTable, 0, 1, true).
		AddItem(helpInfo, 1, 1, false)

	groupTable := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	groupTable.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
	// Enable table to receive focus and handle input
	groupTable.SetSelectable(true, false)
	groupTableGlobal = groupTable

	// Group search input
	groupSearchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetPlaceholder("Group name/ID (/ to focus)...").
		SetFieldWidth(0)
	groupSearchInput.SetBackgroundColor(tcell.ColorDefault)      // Overall background of the box
	groupSearchInput.SetFieldBackgroundColor(tcell.ColorDefault) // Background of the text entry area
	groupSearchInput.SetLabelColor(tcell.ColorWhite)             // Color of the "Search: " label
	groupSearchInput.SetFieldTextColor(tcell.ColorWhite)         // Color of the text as you type
	groupSearchInput.SetPlaceholderTextColor(tcell.ColorGray)    // Color of the placeholder text
	groupSearchInputGlobal = groupSearchInput

	// Explicitly set the style for the field itself
	groupSearchInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorWhite))

	// Group help info
	groupHelpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=select group, /=search, q=back to all pipelines, Q=quit").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGray)
	groupHelpInfo.SetBackgroundColor(tcell.ColorDefault)

	groupListFlexView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(groupSearchInput, 1, 1, false).
		AddItem(groupTable, 0, 1, true).
		AddItem(groupHelpInfo, 1, 1, false)

	// Log View elements
	logViewTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() }) // Redraw on text change for scrolling
	logViewTextView.SetBorder(true).SetTitle("Logs").SetBackgroundColor(tcell.ColorDefault)

	// Status bar for log view
	logStatusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("Status: [white]UNKNOWN[-] | Auto-refresh: ON").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorWhite)
	logStatusBar.SetBackgroundColor(tcell.ColorDefault)

	// Create log page with status bar
	logPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logViewTextView, 0, 1, true). // TextView takes most space, is focus target
		AddItem(logStatusBar, 1, 1, false)    // Status bar takes 1 line, not focusable

	// Run History View elements
	runHistoryTable = tview.NewTable().SetBorders(false).SetSelectable(true, false)
	runHistoryTable.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
	// Enable table to receive focus and handle input
	runHistoryTable.SetSelectable(true, false)

	// Run history help info
	runHistoryHelpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=view logs, r=run pipeline, [/]=prev/next page, 0=first page, q=back to pipelines, Q=quit").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGray)
	runHistoryHelpInfo.SetBackgroundColor(tcell.ColorDefault)

	runHistoryPage = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(runHistoryTable, 0, 1, true).
		AddItem(runHistoryHelpInfo, 1, 1, false)

	// Main pages
	mainPages := tview.NewPages().
		AddPage("pipelines", pipelineListFlexView, true, true).
		AddPage("groups", groupListFlexView, true, false).
		AddPage("run_history", runHistoryPage, true, false).
		AddPage("logs", logPage, true, false) // Log page, initially not visible
	mainPagesGlobal = mainPages // Set global reference for modals

	// Initial population of the pipeline table
	if fetchErrPipelines != nil {
		pipelineTable.Clear()
		cell := tview.NewTableCell(fmt.Sprintf("Error fetching pipelines: %v", fetchErrPipelines)).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter)
		pipelineTable.SetCell(0, 0, cell)

		if os.Getenv("FLOWT_DEBUG") == "1" {
			fmt.Printf("UI Error: %s\n", fetchErrPipelines)
		}
	} else {
		updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
	}

	// Initial population of the group table
	if fetchErrGroups != nil {
		groupTable.Clear()
		cell := tview.NewTableCell(fmt.Sprintf("Error fetching groups: %v", fetchErrGroups)).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter)
		groupTable.SetCell(0, 0, cell)

		if os.Getenv("FLOWT_DEBUG") == "1" {
			fmt.Printf("UI Error: %s\n", fetchErrGroups)
		}
	} else {
		updateGroupTable(groupTable, app)
	}

	// --- Event Handlers for pipelineTable ---
	pipelineTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentRow, _ := pipelineTable.GetSelection()
		rowCount := pipelineTable.GetRowCount()

		switch event.Rune() {
		case 'j':
			if rowCount > 1 {
				newRow := currentRow + 1
				if newRow >= rowCount {
					newRow = 1 // Skip header row
				}
				pipelineTable.Select(newRow, 0)
			}
			return nil
		case 'k':
			if rowCount > 1 {
				newRow := currentRow - 1
				if newRow < 1 {
					newRow = rowCount - 1
				}
				pipelineTable.Select(newRow, 0)
			}
			return nil
		case 'q':
			if currentViewMode == "pipelines_in_group" {
				currentViewMode = "group_list"
				mainPages.SwitchToPage("groups")
				app.SetFocus(groupTable)
				return nil
			}
			// If search is active, clear search and focus table. Otherwise, do nothing.
			if currentSearchQuery != "" {
				currentSearchQuery = ""
				searchInput.SetText("")
				updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
				// app.SetFocus(pipelineTable) // Already focused or will be by searchInput.SetDoneFunc
			}
			return nil
		case 'r': // Run pipeline
			if rowCount > 1 && currentRow > 0 {
				if selectedPipeline, ok := pipelineRowMap[currentRow]; ok && selectedPipeline != nil {
					showRunPipelineDialog(selectedPipeline, app, apiClient, orgId)
				}
			}
			return nil
		case 'a': // Toggle status filter
			showOnlyRunningWaiting = !showOnlyRunningWaiting
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			return nil
		}
		switch event.Key() {
		case tcell.KeyEnter:
			if rowCount > 1 && currentRow > 0 {
				if selectedPipeline, ok := pipelineRowMap[currentRow]; ok && selectedPipeline != nil {
					currentPipelineIDForRun = selectedPipeline.PipelineID
					currentPipelineName = selectedPipeline.Name
					isRunHistoryActive = true

					// Reset pagination state when entering run history
					currentRunHistoryPage = 1
					totalRunHistoryPages = 1

					// Update run history table and switch to it
					updateRunHistoryTable(runHistoryTable, app, apiClient, orgId, selectedPipeline.PipelineID, selectedPipeline.Name)
					mainPages.SwitchToPage("run_history")
					app.SetFocus(runHistoryTable)
				}
			}
			return nil
		case tcell.KeyEscape:
			if currentViewMode == "pipelines_in_group" {
				currentViewMode = "group_list"
				mainPages.SwitchToPage("groups")
				app.SetFocus(groupTable)
				return nil
			}
		}
		return event
	})

	// --- Event Handlers for searchInput ---
	searchInput.SetChangedFunc(func(text string) {
		currentSearchQuery = text
		updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
	})
	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyDown || key == tcell.KeyUp {
			app.SetFocus(pipelineTable)
		} else if key == tcell.KeyEscape {
			currentSearchQuery = ""
			searchInput.SetText("")
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			app.SetFocus(pipelineTable)
		}
	})

	// Add input capture for search input to handle 'q' key
	searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Allow 'q' to be typed in search input.
		// For other keys, let them be processed by SetDoneFunc or propagate.
		return event
	})

	// --- Event Handlers for groupSearchInput ---
	groupSearchInput.SetChangedFunc(func(text string) {
		currentGroupSearchQuery = text
		updateGroupTable(groupTable, app)
	})
	groupSearchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyDown || key == tcell.KeyUp {
			app.SetFocus(groupTable)
		} else if key == tcell.KeyEscape {
			currentGroupSearchQuery = ""
			groupSearchInput.SetText("")
			updateGroupTable(groupTable, app)
			app.SetFocus(groupTable)
		}
	})

	// Add input capture for group search input to handle 'q' key
	groupSearchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Allow 'q' to be typed in group search input.
		// For other keys, let them be processed by SetDoneFunc or propagate.
		return event
	})

	// --- Event Handlers for groupTable ---
	groupTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentRow, _ := groupTable.GetSelection()
		rowCount := groupTable.GetRowCount()
		switch event.Rune() {
		case 'j':
			if rowCount > 1 {
				newRow := currentRow + 1
				if newRow >= rowCount {
					newRow = 1 // Skip header row
				}
				groupTable.Select(newRow, 0)
			}
			return nil
		case 'k':
			if rowCount > 1 {
				newRow := currentRow - 1
				if newRow < 1 {
					newRow = rowCount - 1
				}
				groupTable.Select(newRow, 0)
			}
			return nil
		case 'q':
			// Back to pipelines view
			currentViewMode = "all_pipelines"
			selectedGroupID = ""
			selectedGroupName = ""
			currentGroupSearchQuery = ""
			groupSearchInput.SetText("")
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineTable)
			return nil
		case '/':
			// Focus group search input
			app.SetFocus(groupSearchInput)
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			if rowCount > 1 && currentRow > 0 {
				if selectedGroup, ok := groupRowMap[currentRow]; ok && selectedGroup != nil {
					selectedGroupID = selectedGroup.GroupID
					selectedGroupName = selectedGroup.Name
					currentViewMode = "pipelines_in_group"
					currentSearchQuery = ""
					searchInput.SetText("")
					updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
					mainPages.SwitchToPage("pipelines")
					app.SetFocus(pipelineTable)
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			currentViewMode = "all_pipelines"
			selectedGroupID = ""
			selectedGroupName = ""
			currentGroupSearchQuery = ""
			groupSearchInput.SetText("")
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineTable)
			return nil
		}
		return event
	})

	// --- Event Handlers for runHistoryTable ---
	runHistoryTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentRow, _ := runHistoryTable.GetSelection()
		rowCount := runHistoryTable.GetRowCount()

		switch event.Rune() {
		case 'j':
			if rowCount > 1 {
				newRow := currentRow + 1
				if newRow >= rowCount {
					newRow = 1 // Skip header row
				}
				runHistoryTable.Select(newRow, 0)
			}
			return nil
		case 'k':
			if rowCount > 1 {
				newRow := currentRow - 1
				if newRow < 1 {
					newRow = rowCount - 1
				}
				runHistoryTable.Select(newRow, 0)
			}
			return nil
		case 'q':
			// Back to pipelines view
			isRunHistoryActive = false
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineTable)
			return nil
		case '[':
			// Previous page
			if currentRunHistoryPage > 1 {
				currentRunHistoryPage--
				updateRunHistoryTable(runHistoryTable, app, apiClient, orgId, currentPipelineIDForRun, currentPipelineName)
				if runHistoryTable.GetRowCount() > 1 {
					runHistoryTable.Select(1, 0) // Select first data row
				}
			}
			return nil
		case ']':
			// Next page
			if currentRunHistoryPage < totalRunHistoryPages {
				currentRunHistoryPage++
				updateRunHistoryTable(runHistoryTable, app, apiClient, orgId, currentPipelineIDForRun, currentPipelineName)
				if runHistoryTable.GetRowCount() > 1 {
					runHistoryTable.Select(1, 0) // Select first data row
				}
			}
			return nil
		case '0':
			// Go to first page
			if currentRunHistoryPage != 1 {
				currentRunHistoryPage = 1
				updateRunHistoryTable(runHistoryTable, app, apiClient, orgId, currentPipelineIDForRun, currentPipelineName)
				if runHistoryTable.GetRowCount() > 1 {
					runHistoryTable.Select(1, 0) // Select first data row
				}
			}
			return nil
		case 'r': // Run pipeline
			// Find the pipeline object for the current pipeline
			var selectedPipeline *api.Pipeline
			for _, p := range allPipelines {
				if p.PipelineID == currentPipelineIDForRun {
					selectedPipeline = &p
					break
				}
			}
			if selectedPipeline != nil {
				showRunPipelineDialog(selectedPipeline, app, apiClient, orgId)
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyEnter:
			if rowCount > 1 && currentRow > 0 {
				if selectedRun, ok := runHistoryRowMap[currentRow]; ok && selectedRun != nil {
					currentRunID = selectedRun.RunID
					currentRunStatus = selectedRun.Status // Initialize status from selected run
					isLogViewActive = true

					// Switch to log view and start auto-refresh for historical runs
					go func() {
						app.QueueUpdateDraw(func() {
							if logViewTextView != nil {
								logViewTextView.SetText(fmt.Sprintf("Fetching logs for run %s...", currentRunID))
								// Update status bar immediately
								updateLogStatusBar()
								mainPages.SwitchToPage("logs")
								app.SetFocus(logViewTextView)
							}
						})

						// Use the run data we already have from the table to avoid duplicate API calls
						pipelineName := currentPipelineName
						branchInfo := "N/A" // Historical runs don't have branch info readily available
						repoInfo := ""

						// Start auto-refresh for this historical run (but only refresh once for completed runs)
						if selectedRun.Status == "RUNNING" || selectedRun.Status == "QUEUED" {
							// Only auto-refresh for running pipelines
							startLogAutoRefresh(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)
						} else {
							// For completed runs, just fetch once
							fetchAndDisplayLogs(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)
						}
					}()
				}
			}
			return nil
		case tcell.KeyEscape:
			isRunHistoryActive = false
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineTable)
			return nil
		}
		return event
	})

	// --- Event Handlers for logViewTextView ---
	logViewTextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'r':
			// Manual refresh
			go func() {
				// Get current context for refresh
				pipelineName := currentPipelineName
				branchInfo := "N/A"
				repoInfo := ""

				// Perform manual refresh
				fetchAndDisplayLogs(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)
			}()
			return nil
		case 'e':
			// Open logs in editor
			if logViewTextView != nil {
				logContent := logViewTextView.GetText(false)
				if logContent != "" {
					err := OpenInEditor(logContent, app)
					if err != nil {
						ShowModal("Error", fmt.Sprintf("Failed to open editor: %v", err), []string{"OK"}, nil)
					}
				}
			}
			return nil
		case 'v':
			// Open logs in pager
			if logViewTextView != nil {
				logContent := logViewTextView.GetText(false)
				if logContent != "" {
					err := OpenInPager(logContent, app)
					if err != nil {
						ShowModal("Error", fmt.Sprintf("Failed to open pager: %v", err), []string{"OK"}, nil)
					}
				}
			}
			return nil
		case 'b', 'q':
			isLogViewActive = false
			// Stop auto-refresh when leaving log view
			stopLogAutoRefresh()
			if isRunHistoryActive {
				// Return to run history if we came from there
				mainPages.SwitchToPage("run_history")
				app.SetFocus(runHistoryTable)
			} else {
				// Return to pipelines
				mainPages.SwitchToPage("pipelines")
				app.SetFocus(pipelineTable)
			}
			return nil
		}

		if event.Key() == tcell.KeyEscape {
			isLogViewActive = false
			// Stop auto-refresh when leaving log view
			stopLogAutoRefresh()
			if isRunHistoryActive {
				// Return to run history if we came from there
				mainPages.SwitchToPage("run_history")
				app.SetFocus(runHistoryTable)
			} else {
				// Return to pipelines
				mainPages.SwitchToPage("pipelines")
				app.SetFocus(pipelineTable)
			}
			return nil
		}
		// Allow default scrolling for arrow keys, PageUp/Down etc.
		return event
	})

	// Global keybindings on mainPages
	mainPages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentPage, _ := mainPages.GetFrontPage()

		// Handle character keys
		switch event.Rune() {
		case '/':
			if currentPage == "pipelines" { // Allow search focus on pipelines page
				app.SetFocus(searchInput)
				return nil
			} else if currentPage == "groups" { // Allow search focus on groups page
				app.SetFocus(groupSearchInput)
				return nil
			}
		}

		// Handle special keys
		switch event.Key() {
		case tcell.KeyCtrlG:
			if currentPage == "pipelines" {
				currentViewMode = "group_list"
				if groupTable.GetRowCount() > 1 {
					groupTable.Select(1, 0) // Select first data row
				}
				mainPages.SwitchToPage("groups")
				app.SetFocus(groupTable)
			} else if currentPage == "groups" {
				currentViewMode = "all_pipelines"
				selectedGroupID = ""
				selectedGroupName = ""
				updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
				mainPages.SwitchToPage("pipelines")
				app.SetFocus(pipelineTable)
			}
			return nil
		}
		return event
	})

	app.SetFocus(pipelineTable)

	// Set global app input capture for 'q' and 'Q' as the outermost layer
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		focused := app.GetFocus()

		switch event.Rune() {
		case 'Q': // Uppercase Q quits
			app.Stop()
			return nil // Consumed
		case 'q': // Lowercase q
			// If searchInput or groupSearchInput is focused, it needs to process 'q' for typing.
			// Their own InputCapture should return event to allow typing.
			if focused == searchInput || focused == groupSearchInputGlobal {
				return event
			}
			// In all other cases where 'q' bubbles up to the app level,
			// we consume it and do nothing. This prevents any default quit.
			// Specific navigation for 'q' should have been handled by component-level
			// input captures (which should return nil).
			return nil // Consumed, do nothing (prevents quit)
		}

		// For other events not handled here (e.g., Ctrl+G, /),
		// they will propagate to other handlers like mainPages.SetInputCapture.
		return event
	})

	return mainPages
}
