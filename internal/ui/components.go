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

// SetBookmarkFunctions sets the global bookmark functions
func SetBookmarkFunctions(toggleBookmark func(string) bool, isBookmarked func(string) bool, saveConfig func() error, bookmarks []string) {
	globalToggleBookmark = toggleBookmark
	globalIsBookmarked = isBookmarked
	globalSaveConfig = saveConfig
	globalBookmarks = bookmarks
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

	// Bookmark filtering
	showOnlyBookmarked bool // Toggle between all pipelines and bookmarked only

	// Global configuration for editor and pager
	globalEditorCmd string
	globalPagerCmd  string

	// Global bookmark functions
	globalToggleBookmark func(string) bool
	globalIsBookmarked   func(string) bool
	globalSaveConfig     func() error
	globalBookmarks      []string

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

	// Progressive loading state for pipelines
	isPipelineLoadingInProgress bool
	pipelineLoadingCurrentPage  int
	pipelineLoadingTotalPages   int
	pipelineLoadingComplete     bool
	pipelineLoadingError        error

	// Cache for all pipelines data - only load once per application lifecycle
	allPipelinesCache        []api.Pipeline // Cache for all pipelines (no status filter)
	allPipelinesCacheLoaded  bool           // Whether the cache has been loaded
	allPipelinesCacheLoading bool           // Whether cache loading is in progress

	// Progressive loading state for logs
	isLogLoadingInProgress bool   // Whether log loading is in progress
	logLoadingCurrentJob   int    // Current job being loaded (1-based)
	logLoadingTotalJobs    int    // Total number of jobs to load
	logLoadingComplete     bool   // Whether log loading is complete
	logLoadingError        error  // Error during log loading
	originalRunStatus      string // Original status from run history (to prevent overwriting)
	preserveOriginalStatus bool   // Whether to preserve the original status

	// Vim-style search state for log view
	logSearchActive     bool              // Whether search mode is active
	logSearchQuery      string            // Current search query
	logSearchMatches    []int             // Byte positions of all matches in the log text
	logSearchCurrentIdx int               // Current match index (0-based)
	logOriginalText     string            // Original text without search highlighting
	logSearchInput      *tview.InputField // Search input field for log view
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

	// Restore focus based on current active view
	if isLogViewActive && logViewTextView != nil {
		// If log view is active, restore focus to log view
		appGlobal.SetFocus(logViewTextView)
	} else if isRunHistoryActive && runHistoryTable != nil {
		// If run history is active, restore focus to run history table
		appGlobal.SetFocus(runHistoryTable)
	} else if pipelineTableGlobal != nil && (currentViewMode == "all_pipelines" || currentViewMode == "pipelines_in_group") {
		// Default to pipeline table for pipeline views
		appGlobal.SetFocus(pipelineTableGlobal)
	} else if currentViewMode == "group_list" && groupTableGlobal != nil {
		// Default to group table for group list view
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
		return tcell.ColorWhite
	case "RUNNING":
		return tcell.ColorGreen
	case "FAIL":
		return tcell.ColorRed
	case "FAILED":
		return tcell.ColorRed
	case "CANCELED":
		return tcell.ColorGray
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

	// Build status part
	var statusPart string
	switch strings.ToUpper(currentRunStatus) {
	case "RUNNING":
		statusPart = fmt.Sprintf("Status: [green]%s[-]", currentRunStatus)
	case "SUCCESS":
		statusPart = fmt.Sprintf("Status: [white]%s[-]", currentRunStatus)
	case "FAILED":
		statusPart = fmt.Sprintf("Status: [red]%s[-]", currentRunStatus)
	case "CANCELED":
		statusPart = fmt.Sprintf("Status: [gray]%s[-]", currentRunStatus)
	default:
		statusPart = fmt.Sprintf("Status: [white]%s[-]", currentRunStatus)
	}

	// Build loading progress part (for log loading)
	var loadingPart string
	if isLogLoadingInProgress {
		if logLoadingTotalJobs > 0 {
			loadingPart = fmt.Sprintf(" | Loading logs: %d/%d jobs", logLoadingCurrentJob, logLoadingTotalJobs)
		} else {
			loadingPart = " | Loading logs..."
		}
	}

	// Build auto-refresh part (only for newly created runs or running historical runs)
	var autoRefreshPart string
	if !isLogLoadingInProgress && (isNewlyCreatedRun || strings.ToUpper(currentRunStatus) == "RUNNING") {
		autoRefreshPart = fmt.Sprintf(" | Auto-refresh: %s", getAutoRefreshStatus())
	}

	// Build instructions part
	instructionsPart := " | Press '/' to search, 'f'/'b' page down/up, 'd'/'u' half-page, 'r' refresh, 'X' stop, 'q' return, 'e' edit, 'v' pager"

	// Combine all parts
	statusText = statusPart + loadingPart + autoRefreshPart + instructionsPart
	statusColor = tcell.ColorDefault

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
	pipelineTableGlobal = table // Update global reference

	var title string
	if currentViewMode == "pipelines_in_group" {
		if showOnlyBookmarked {
			title = fmt.Sprintf("Pipelines in '%s' (BOOKMARKED)", selectedGroupName)
		} else if showOnlyRunningWaiting {
			title = fmt.Sprintf("Pipelines in '%s' (RUNNING+WAITING)", selectedGroupName)
		} else {
			title = fmt.Sprintf("Pipelines in '%s'", selectedGroupName)
		}
	} else {
		if showOnlyBookmarked {
			title = "Pipelines (BOOKMARKED)"
		} else if showOnlyRunningWaiting {
			title = "Pipelines (RUNNING+WAITING)"
		} else {
			title = "All Pipelines"
		}
	}

	// Add loading progress to title if loading is in progress
	if isPipelineLoadingInProgress {
		if pipelineLoadingTotalPages > 0 {
			title += fmt.Sprintf(" (Loading... %d/%d pages)", pipelineLoadingCurrentPage, pipelineLoadingTotalPages)
		} else {
			title += " (Loading...)"
		}
	} else if pipelineLoadingComplete {
		title += fmt.Sprintf(" (%d pipelines)", len(allPipelines))
	}

	table.SetTitle(title)

	// Set table headers - with bookmark column
	headers := []string{" ", "Name"}

	// Only clear and set headers if this is a fresh load (not progressive loading)
	if !isPipelineLoadingInProgress || table.GetRowCount() == 0 {
		table.Clear()
		for col, header := range headers {
			cell := tview.NewTableCell(header).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft).
				SetSelectable(false).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(0, col, cell)
		}
		// Clear the pipeline row map when starting fresh
		pipelineRowMap = make(map[int]*api.Pipeline)
	}

	// 1. Get pipelines based on current view mode and status filter
	var tempFilteredByGroup []api.Pipeline
	if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
		// For group pipelines, use the data loaded by the progressive loading system
		// Don't make additional API calls here to avoid duplicate requests
		tempFilteredByGroup = append(tempFilteredByGroup, allPipelines...)
	} else {
		// Use all pipelines for "all_pipelines" view
		tempFilteredByGroup = append(tempFilteredByGroup, allPipelines...)
	}

	// 1.5. Apply client-side status filtering if using cached data
	var tempFilteredByStatus []api.Pipeline
	if showOnlyRunningWaiting && currentViewMode == "all_pipelines" && allPipelinesCacheLoaded {
		// When using cached data for all pipelines view, we need to filter on client side
		for _, p := range tempFilteredByGroup {
			// Check both Status and LastRunStatus for RUNNING/WAITING
			status := strings.ToUpper(p.Status)
			lastRunStatus := strings.ToUpper(p.LastRunStatus)
			if status == "RUNNING" || status == "WAITING" || lastRunStatus == "RUNNING" || lastRunStatus == "WAITING" {
				tempFilteredByStatus = append(tempFilteredByStatus, p)
			}
		}
	} else {
		// No client-side status filtering needed for:
		// 1. No status filter active
		// 2. Group pipelines (server-side filtered)
		// 3. All pipelines loaded from server with status filter
		tempFilteredByStatus = append(tempFilteredByStatus, tempFilteredByGroup...)
	}

	// 2. Filter by search query (fuzzy search)
	tempFilteredBySearch := make([]api.Pipeline, 0)
	if currentSearchQuery != "" {
		for _, p := range tempFilteredByStatus {
			if fuzzyMatch(currentSearchQuery, p.Name) || fuzzyMatch(currentSearchQuery, p.PipelineID) {
				tempFilteredBySearch = append(tempFilteredBySearch, p)
			}
		}
	} else {
		tempFilteredBySearch = append(tempFilteredBySearch, tempFilteredByStatus...)
	}

	// 3. Filter by bookmark status if enabled
	tempFilteredByBookmark := make([]api.Pipeline, 0)
	if showOnlyBookmarked && globalIsBookmarked != nil {
		for _, p := range tempFilteredBySearch {
			if globalIsBookmarked(p.Name) {
				tempFilteredByBookmark = append(tempFilteredByBookmark, p)
			}
		}
	} else {
		tempFilteredByBookmark = append(tempFilteredByBookmark, tempFilteredBySearch...)
	}

	// 4. Sort pipelines: bookmarked first, then others
	finalFilteredPipelines := make([]api.Pipeline, 0)
	bookmarkedPipelines := make([]api.Pipeline, 0)
	nonBookmarkedPipelines := make([]api.Pipeline, 0)

	if globalIsBookmarked != nil && !showOnlyBookmarked {
		// Separate bookmarked and non-bookmarked pipelines
		for _, p := range tempFilteredByBookmark {
			if globalIsBookmarked(p.Name) {
				bookmarkedPipelines = append(bookmarkedPipelines, p)
			} else {
				nonBookmarkedPipelines = append(nonBookmarkedPipelines, p)
			}
		}
		// Combine: bookmarked first, then non-bookmarked
		finalFilteredPipelines = append(finalFilteredPipelines, bookmarkedPipelines...)
		finalFilteredPipelines = append(finalFilteredPipelines, nonBookmarkedPipelines...)
	} else {
		finalFilteredPipelines = tempFilteredByBookmark
	}

	// Populate the table
	if len(finalFilteredPipelines) == 0 && !isPipelineLoadingInProgress {
		// Show "no data" message only if not loading
		cell := tview.NewTableCell("No pipelines match filters.").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter)
		table.SetCell(1, 0, cell)
		for i := 1; i < len(headers); i++ {
			table.SetCell(1, i, tview.NewTableCell(""))
		}
	} else {
		// For progressive loading, we need to determine the starting row
		startRow := 1 // Default for fresh load
		if isPipelineLoadingInProgress && table.GetRowCount() > 1 {
			// For progressive loading, start from the next available row
			startRow = table.GetRowCount()
		}

		for i, p := range finalFilteredPipelines {
			pipelineCopy := p // Important: capture range variable for reference
			row := startRow + i

			// Store the pipeline object in our map
			pipelineRowMap[row] = &pipelineCopy

			// Column 0: Bookmark indicator
			bookmarkText := " "
			if globalIsBookmarked != nil && globalIsBookmarked(pipelineCopy.Name) {
				bookmarkText = "★"
			}
			bookmarkCell := tview.NewTableCell(bookmarkText).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetBackgroundColor(tcell.ColorDefault)
			table.SetCell(row, 0, bookmarkCell)

			// Column 1: Pipeline Name
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
		currentRunStatus = "RUNNING"   // New runs start as RUNNING
		originalRunStatus = "RUNNING"  // Store original status
		preserveOriginalStatus = false // Allow status updates for newly created runs
		isLogViewActive = true
		isNewlyCreatedRun = true // Mark this as a newly created run

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

		// Start automatic log fetching and refreshing every 5 seconds for newly created runs
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
	// Track if current log view is for a newly created run or running historical run
	isNewlyCreatedRun bool // Whether this is a newly created run (should auto-refresh)
)

// startLogAutoRefresh starts automatic log fetching and refreshing every 5 seconds
func startLogAutoRefresh(app *tview.Application, apiClient *api.Client, orgId, pipelineName, branchInfo, repoInfo string) {
	// Stop any existing refresh ticker
	stopLogAutoRefresh()

	// Reset delayed stop state
	finishedRefreshCount = 0
	pipelineFinished = false

	// Only start auto-refresh for newly created runs or running historical runs
	shouldAutoRefresh := isNewlyCreatedRun || strings.ToUpper(currentRunStatus) == "RUNNING"

	// Always fetch logs at least once
	fetchAndDisplayLogs(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)

	// Only start ticker if auto-refresh is needed
	if !shouldAutoRefresh {
		return
	}

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
// This function now uses progressive loading to show logs as they are fetched
func fetchAndDisplayLogs(app *tview.Application, apiClient *api.Client, orgId, pipelineName, branchInfo, repoInfo string) {
	if currentRunID == "" || currentPipelineIDForRun == "" {
		return
	}

	// Check if app is still valid
	if app == nil {
		return
	}

	// Store search state before updating logs
	wasSearchActive := logSearchActive
	searchQuery := logSearchQuery

	// Clear search state temporarily during log update
	if logSearchActive {
		logSearchActive = false
		logOriginalText = ""
	}

	// Start progressive log loading
	startProgressiveLogLoading(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)

	// Note: Search state will be restored in the final update of startProgressiveLogLoading
	// if the search was active before the update
	if wasSearchActive && searchQuery != "" {
		// Schedule search restoration after log loading completes
		go func() {
			// Wait for log loading to complete
			for isLogLoadingInProgress {
				time.Sleep(100 * time.Millisecond)
			}

			// Restore search state
			app.QueueUpdateDraw(func() {
				if isLogViewActive && logViewTextView != nil {
					logOriginalText = logViewTextView.GetText(false)
					performLogSearch(searchQuery, app)
				}
			})
		}()
	}
}

// getVMDeploymentLogs fetches logs for VM deployment jobs
func getVMDeploymentLogs(apiClient *api.Client, orgId, pipelineIdStr, runIdStr string, job api.Job) (string, error) {
	var logs strings.Builder

	// Extract deployOrderId from job actions
	deployOrderId, err := extractDeployOrderIdFromActions(job.Actions)
	if err != nil {
		logs.WriteString(fmt.Sprintf("Error extracting deployOrderId from job actions: %v\n", err))
		// For running deployments, the deployOrderId might not be available yet
		if job.Status == "RUNNING" || job.Status == "QUEUED" {
			logs.WriteString("Deployment is still in progress. Deploy order information will be available once the deployment starts.\n")
		} else if job.Status == "FAILED" {
			logs.WriteString("Deployment job failed. No deploy order information available.\n")
		} else {
			logs.WriteString("Deploy order information is not available for this job.\n")
		}
		return logs.String(), nil
	}

	// Get VM deployment order details
	deployOrder, err := apiClient.GetVMDeployOrder(orgId, pipelineIdStr, deployOrderId)
	if err != nil {
		logs.WriteString(fmt.Sprintf("Error fetching VM deploy order %s: %v\n", deployOrderId, err))
		logs.WriteString("Unable to retrieve deployment details at this time.\n")
		return logs.String(), nil
	}

	logs.WriteString(fmt.Sprintf("[yellow]Deploy Order ID: %d[-]\n", deployOrder.DeployOrderId))
	logs.WriteString(fmt.Sprintf("[yellow]Deploy Status: %s[-]\n", deployOrder.Status))
	logs.WriteString(fmt.Sprintf("[yellow]Current Batch: %d/%d[-]\n", deployOrder.CurrentBatch, deployOrder.TotalBatch))
	logs.WriteString(fmt.Sprintf("[yellow]Host Group ID: %d[-]\n", deployOrder.DeployMachineInfo.HostGroupId))
	logs.WriteString("[yellow]" + strings.Repeat("-", 40) + "[-]\n")

	// Get logs for each machine in the deployment
	if len(deployOrder.DeployMachineInfo.DeployMachines) == 0 {
		logs.WriteString("No machines found in this deployment.\n")
	} else {
		for i, machine := range deployOrder.DeployMachineInfo.DeployMachines {
			logs.WriteString(fmt.Sprintf("[yellow]Machine #%d: %s (SN: %s)[-]\n", i+1, machine.IP, machine.MachineSn))
			logs.WriteString(fmt.Sprintf("[yellow]Machine Status: %s, Client Status: %s[-]\n", machine.Status, machine.ClientStatus))
			logs.WriteString(fmt.Sprintf("[yellow]Batch: %d[-]\n", machine.BatchNum))
			logs.WriteString("[yellow]" + strings.Repeat(".", 30) + "[-]\n")

			// Get machine deployment log
			machineLog, err := apiClient.GetVMDeployMachineLog(orgId, pipelineIdStr, deployOrderId, machine.MachineSn)
			if err != nil {
				logs.WriteString(fmt.Sprintf("Error fetching machine log for %s: %v\n", machine.MachineSn, err))
			} else {
				if machineLog.DeployBeginTime != "" {
					logs.WriteString(fmt.Sprintf("Deploy Begin Time: %s\n", machineLog.DeployBeginTime))
				}
				if machineLog.DeployEndTime != "" {
					logs.WriteString(fmt.Sprintf("Deploy End Time: %s\n", machineLog.DeployEndTime))
				}
				if machineLog.AliyunRegion != "" {
					logs.WriteString(fmt.Sprintf("Region: %s\n", machineLog.AliyunRegion))
				}
				if machineLog.DeployLogPath != "" {
					logs.WriteString(fmt.Sprintf("Log Path: %s\n", machineLog.DeployLogPath))
				}
				logs.WriteString("Deploy Log:\n")
				if machineLog.DeployLog == "" {
					logs.WriteString("No deployment logs available for this machine.\n")
				} else {
					logs.WriteString(machineLog.DeployLog)
					if !strings.HasSuffix(machineLog.DeployLog, "\n") {
						logs.WriteString("\n")
					}
				}
			}
			logs.WriteString("\n")
		}
	}

	return logs.String(), nil
}

// extractDeployOrderIdFromActions extracts deployOrderId from job actions array
// This is a local implementation for UI use
func extractDeployOrderIdFromActions(actions []api.JobAction) (string, error) {
	if len(actions) == 0 {
		return "", fmt.Errorf("no actions found in job")
	}

	// Look for GetVMDeployOrder action
	for _, action := range actions {
		if action.Type == "GetVMDeployOrder" {
			// First try to get deployOrderId from action.params
			if action.Params != nil {
				if deployOrderId, ok := action.Params["deployOrderId"]; ok {
					if id, ok := deployOrderId.(float64); ok {
						return fmt.Sprintf("%.0f", id), nil
					}
					if id, ok := deployOrderId.(string); ok {
						return id, nil
					}
				}
			}

			// Then try to parse from action.data JSON string
			if action.Data != "" {
				var actionData map[string]interface{}
				if err := json.Unmarshal([]byte(action.Data), &actionData); err != nil {
					continue
				}

				// Look for deployOrderId in various possible locations
				if deployOrderId, ok := actionData["deployOrderId"]; ok {
					if id, ok := deployOrderId.(float64); ok {
						return fmt.Sprintf("%.0f", id), nil
					}
					if id, ok := deployOrderId.(string); ok {
						return id, nil
					}
				}

				// Check nested structure
				if data, ok := actionData["data"].(map[string]interface{}); ok {
					if deployOrderIdData, ok := data["deployOrderId"].(map[string]interface{}); ok {
						if id, ok := deployOrderIdData["id"].(float64); ok {
							return fmt.Sprintf("%.0f", id), nil
						}
						if id, ok := deployOrderIdData["id"].(string); ok {
							return id, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("deployOrderId not found in job actions")
}

// startProgressiveLogLoading starts loading logs progressively job by job
func startProgressiveLogLoading(app *tview.Application, apiClient *api.Client, orgId, pipelineName, branchInfo, repoInfo string) {
	// Reset loading state
	isLogLoadingInProgress = true
	logLoadingCurrentJob = 0
	logLoadingTotalJobs = 0
	logLoadingComplete = false
	logLoadingError = nil

	// Initialize log display with header
	app.QueueUpdateDraw(func() {
		if !isLogViewActive || logViewTextView == nil {
			return
		}

		// Build initial header
		var logText strings.Builder
		logText.WriteString(fmt.Sprintf("Pipeline: %s\n", pipelineName))
		logText.WriteString(fmt.Sprintf("Run ID: %s\n", currentRunID))
		logText.WriteString(fmt.Sprintf("Branch: %s\n", branchInfo))
		if repoInfo != "" {
			logText.WriteString(fmt.Sprintf("Repository: %s\n", repoInfo))
		}
		logText.WriteString(fmt.Sprintf("Last Updated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
		logText.WriteString(strings.Repeat("=", 80) + "\n\n")
		logText.WriteString("Loading pipeline run details...\n")

		logViewTextView.SetText(logText.String())
		updateLogStatusBar()
	})

	// Start loading in a goroutine
	go func() {
		// Step 1: Get pipeline run details to obtain job list
		runDetails, err := apiClient.GetPipelineRunDetails(orgId, currentPipelineIDForRun, currentRunID)
		if err != nil {
			logLoadingError = err
			isLogLoadingInProgress = false
			app.QueueUpdateDraw(func() {
				if !isLogViewActive || logViewTextView == nil {
					return
				}

				var logText strings.Builder
				logText.WriteString(fmt.Sprintf("Pipeline: %s\n", pipelineName))
				logText.WriteString(fmt.Sprintf("Run ID: %s\n", currentRunID))
				logText.WriteString(fmt.Sprintf("Branch: %s\n", branchInfo))
				if repoInfo != "" {
					logText.WriteString(fmt.Sprintf("Repository: %s\n", repoInfo))
				}
				logText.WriteString(fmt.Sprintf("Last Updated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
				logText.WriteString(strings.Repeat("=", 80) + "\n\n")
				logText.WriteString(fmt.Sprintf("Error fetching pipeline run details: %v\n\n", err))
				logText.WriteString("Note: Log fetching may require additional parameters or the pipeline may still be initializing.\n")

				logViewTextView.SetText(logText.String())
				updateLogStatusBar()
			})
			return
		}

		// Update status if not preserving original status
		if !preserveOriginalStatus {
			currentRunStatus = runDetails.Status
		}

		// Count total jobs
		totalJobs := 0
		for _, stage := range runDetails.Stages {
			totalJobs += len(stage.Jobs)
		}
		logLoadingTotalJobs = totalJobs

		// Update initial display with run details
		app.QueueUpdateDraw(func() {
			if !isLogViewActive || logViewTextView == nil {
				return
			}

			var logText strings.Builder
			logText.WriteString(fmt.Sprintf("Pipeline: %s\n", pipelineName))
			logText.WriteString(fmt.Sprintf("Run ID: %s\n", currentRunID))
			logText.WriteString(fmt.Sprintf("Branch: %s\n", branchInfo))
			if repoInfo != "" {
				logText.WriteString(fmt.Sprintf("Repository: %s\n", repoInfo))
			}
			logText.WriteString(fmt.Sprintf("Last Updated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
			logText.WriteString("=" + strings.Repeat("=", 80) + "\n\n")
			logText.WriteString(fmt.Sprintf("Pipeline Run Logs - Run ID: %s\n", currentRunID))
			logText.WriteString(fmt.Sprintf("Pipeline ID: %s\n", currentPipelineIDForRun))
			logText.WriteString(fmt.Sprintf("Status: %s\n", runDetails.Status))
			logText.WriteString("=" + strings.Repeat("=", 80) + "\n\n")

			if totalJobs == 0 {
				logText.WriteString("No jobs found in this pipeline run.\n")
				isLogLoadingInProgress = false
				logLoadingComplete = true
			} else {
				logText.WriteString(fmt.Sprintf("Found %d jobs to load. Loading logs progressively...\n\n", totalJobs))
			}

			logViewTextView.SetText(logText.String())
			logViewTextView.ScrollToEnd()
			updateLogStatusBar()
		})

		if totalJobs == 0 {
			return
		}

		// Step 2: Load logs for each job progressively
		currentJobIndex := 0
		for _, stage := range runDetails.Stages {
			if len(stage.Jobs) > 0 {
				// Add stage header
				app.QueueUpdateDraw(func() {
					if !isLogViewActive || logViewTextView == nil {
						return
					}

					currentText := logViewTextView.GetText(false)
					currentText += fmt.Sprintf("[yellow]Stage: %s (%s)[-]\n", stage.Name, stage.Index)
					currentText += "-" + strings.Repeat("-", 60) + "\n\n"

					logViewTextView.SetText(currentText)
					logViewTextView.ScrollToEnd()
				})
			}

			for _, job := range stage.Jobs {
				currentJobIndex++
				logLoadingCurrentJob = currentJobIndex

				// Update progress
				app.QueueUpdateDraw(func() {
					updateLogStatusBar()
				})

				// Add job header
				app.QueueUpdateDraw(func() {
					if !isLogViewActive || logViewTextView == nil {
						return
					}

					currentText := logViewTextView.GetText(false)
					currentText += fmt.Sprintf("[yellow]Job #%d: %s (ID: %d)[-]\n", currentJobIndex, job.Name, job.ID)
					currentText += fmt.Sprintf("[yellow]Job Sign: %s[-]\n", job.JobSign)
					currentText += fmt.Sprintf("[yellow]Status: %s[-]\n", job.Status)
					if !job.StartTime.IsZero() {
						currentText += fmt.Sprintf("[yellow]Start Time: %s[-]\n", job.StartTime.Format("2006-01-02 15:04:05"))
					}
					if !job.EndTime.IsZero() {
						currentText += fmt.Sprintf("[yellow]End Time: %s[-]\n", job.EndTime.Format("2006-01-02 15:04:05"))
					}
					currentText += "[yellow]" + strings.Repeat("=", 50) + "[-]\n"

					logViewTextView.SetText(currentText)
					logViewTextView.ScrollToEnd()
				})

				// Fetch logs for this specific job
				var jobLogs string
				var jobErr error

				// Check if this job has GetVMDeployOrder action
				hasVMDeployAction := false
				for _, action := range job.Actions {
					if action.Type == "GetVMDeployOrder" {
						hasVMDeployAction = true
						break
					}
				}

				if hasVMDeployAction {
					// Handle VM deployment job with full implementation
					jobLogs, jobErr = getVMDeploymentLogs(apiClient, orgId, currentPipelineIDForRun, currentRunID, job)
				} else {
					// Regular job - fetch logs
					jobIdStr := fmt.Sprintf("%d", job.ID)
					jobLogs, jobErr = apiClient.GetPipelineJobRunLog(orgId, currentPipelineIDForRun, currentRunID, jobIdStr)
				}

				// Add job logs to display
				app.QueueUpdateDraw(func() {
					if !isLogViewActive || logViewTextView == nil {
						return
					}

					currentText := logViewTextView.GetText(false)

					if jobErr != nil {
						currentText += fmt.Sprintf("Error fetching logs for job %s: %v\n", fmt.Sprintf("%d", job.ID), jobErr)
					} else if jobLogs == "" {
						currentText += "No logs available for this job.\n"
					} else {
						currentText += jobLogs
						if !strings.HasSuffix(jobLogs, "\n") {
							currentText += "\n"
						}
					}

					currentText += "\n" + strings.Repeat("=", 80) + "\n\n"

					logViewTextView.SetText(currentText)
					logViewTextView.ScrollToEnd()
				})

				// Small delay to make progressive loading visible
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Mark loading as complete
		isLogLoadingInProgress = false
		logLoadingComplete = true

		// Final update
		app.QueueUpdateDraw(func() {
			if !isLogViewActive || logViewTextView == nil {
				return
			}

			currentText := logViewTextView.GetText(false)
			currentText += fmt.Sprintf("Total jobs processed: %d\n", currentJobIndex)

			// Handle delayed auto-refresh stop logic
			if !preserveOriginalStatus {
				finalStatus := strings.ToUpper(currentRunStatus)
				if finalStatus == "SUCCESS" || finalStatus == "FAILED" || finalStatus == "CANCELED" {
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
			}

			logViewTextView.SetText(currentText)
			logViewTextView.ScrollToEnd()
			updateLogStatusBar()
		})
	}()
}

// updateRunHistoryTable updates the run history table for a specific pipeline
func updateRunHistoryTable(table *tview.Table, app *tview.Application, apiClient *api.Client, orgId, pipelineId, pipelineName string) {
	table.Clear()

	// Update title with pagination info
	title := fmt.Sprintf("Run History - %s (Page %d/%d) ] to next page, [ to previous page, 0 to go to first page",
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
	title = fmt.Sprintf("Run History - %s (Page %d/%d) ] to next page, [ to previous page, 0 to go to first page",
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
		runNumCell := tview.NewTableCell(fmt.Sprintf("%d", totalRuns-globalRunIndex)).
			SetTextColor(tcell.ColorLightBlue).
			SetAlign(tview.AlignLeft).
			SetBackgroundColor(tcell.ColorDefault).
			SetExpansion(1) // Minimal width
		table.SetCell(row, 0, runNumCell)

		// Status - make it more compact
		statusCell := tview.NewTableCell(runCopy.Status).
			SetTextColor(getStatusColor(runCopy.Status)).
			SetAlign(tview.AlignLeft).
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
			SetAlign(tview.AlignLeft).
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

// startProgressivePipelineLoading starts loading pipelines progressively page by page
// It uses cached data for all pipelines view to avoid repeated server requests
func startProgressivePipelineLoading(table *tview.Table, app *tview.Application, searchInput *tview.InputField, apiClient *api.Client, orgId string) {
	// For group pipelines, always load from server since they're not cached
	if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
		startProgressivePipelineLoadingFromServer(table, app, searchInput, apiClient, orgId)
		return
	}

	// For all pipelines view with status filter, check if we can use cache
	if !showOnlyRunningWaiting {
		// This is the default "all pipelines" view - use cache if available
		if allPipelinesCacheLoaded {
			// Use cached data immediately
			loadPipelinesFromCache(table, app, searchInput, apiClient, orgId)
			return
		} else if allPipelinesCacheLoading {
			// Cache is being loaded, wait for it
			waitForCacheAndLoad(table, app, searchInput, apiClient, orgId)
			return
		} else {
			// Cache not loaded yet, load it for the first time
			loadAllPipelinesCacheProgressively(table, app, searchInput, apiClient, orgId)
			return
		}
	} else {
		// This is RUNNING+WAITING filter - always load from server
		startProgressivePipelineLoadingFromServer(table, app, searchInput, apiClient, orgId)
		return
	}
}

// loadPipelinesFromCache loads pipelines from the cache immediately
func loadPipelinesFromCache(table *tview.Table, app *tview.Application, searchInput *tview.InputField, apiClient *api.Client, orgId string) {
	// Reset loading state
	isPipelineLoadingInProgress = false
	pipelineLoadingComplete = true
	pipelineLoadingError = nil

	// Use cached data
	allPipelines = make([]api.Pipeline, len(allPipelinesCache))
	copy(allPipelines, allPipelinesCache)

	// Update UI immediately
	updatePipelineTable(table, app, searchInput, apiClient, orgId)

	// Select first row if table has content
	if table.GetRowCount() > 1 {
		table.Select(1, 0)
	}
}

// waitForCacheAndLoad waits for cache loading to complete and then loads data
func waitForCacheAndLoad(table *tview.Table, app *tview.Application, searchInput *tview.InputField, apiClient *api.Client, orgId string) {
	// Show loading state
	isPipelineLoadingInProgress = true
	pipelineLoadingComplete = false
	allPipelines = []api.Pipeline{}
	updatePipelineTable(table, app, searchInput, apiClient, orgId)

	// Wait for cache in a goroutine
	go func() {
		// Poll until cache is loaded
		for allPipelinesCacheLoading && !allPipelinesCacheLoaded {
			time.Sleep(100 * time.Millisecond)
		}

		// Update UI on main thread
		app.QueueUpdateDraw(func() {
			if allPipelinesCacheLoaded {
				loadPipelinesFromCache(table, app, searchInput, apiClient, orgId)
			} else {
				// Cache loading failed, show error
				isPipelineLoadingInProgress = false
				pipelineLoadingComplete = false
				table.Clear()
				headers := []string{" ", "Name"}
				for col, header := range headers {
					cell := tview.NewTableCell(header).
						SetTextColor(tcell.ColorYellow).
						SetAlign(tview.AlignLeft).
						SetSelectable(false).
						SetBackgroundColor(tcell.ColorDefault)
					table.SetCell(0, col, cell)
				}
				cell := tview.NewTableCell("Error loading pipelines from cache").
					SetTextColor(tcell.ColorRed).
					SetAlign(tview.AlignCenter)
				table.SetCell(1, 0, cell)
				table.SetCell(1, 1, tview.NewTableCell(""))
				table.SetTitle("Error Loading Pipelines")
			}
		})
	}()
}

// loadAllPipelinesCacheProgressively loads all pipelines into cache for the first time
func loadAllPipelinesCacheProgressively(table *tview.Table, app *tview.Application, searchInput *tview.InputField, apiClient *api.Client, orgId string) {
	// Mark cache as loading
	allPipelinesCacheLoading = true
	allPipelinesCacheLoaded = false
	allPipelinesCache = []api.Pipeline{}

	// Reset loading state
	isPipelineLoadingInProgress = true
	pipelineLoadingCurrentPage = 0
	pipelineLoadingTotalPages = 0
	pipelineLoadingComplete = false
	pipelineLoadingError = nil
	allPipelines = []api.Pipeline{} // Clear existing pipelines

	// Clear table and show loading state
	updatePipelineTable(table, app, searchInput, apiClient, orgId)

	// Start loading in a goroutine
	go func() {
		// Define callback function for each page
		callback := func(pipelines []api.Pipeline, currentPage, totalPages int, isComplete bool) error {
			// Update loading state
			pipelineLoadingCurrentPage = currentPage
			pipelineLoadingTotalPages = totalPages

			// Append new pipelines to both cache and current list
			allPipelinesCache = append(allPipelinesCache, pipelines...)
			allPipelines = append(allPipelines, pipelines...)

			// Update UI on main thread
			app.QueueUpdateDraw(func() {
				// Update the table with new pipelines
				updatePipelineTable(table, app, searchInput, apiClient, orgId)

				// If this is the first page and table has content, select first row
				if currentPage == 1 && table.GetRowCount() > 1 {
					table.Select(1, 0)
				}
			})

			// Mark as complete if this is the last page
			if isComplete {
				pipelineLoadingComplete = true
				isPipelineLoadingInProgress = false
				allPipelinesCacheLoaded = true
				allPipelinesCacheLoading = false

				// Final UI update
				app.QueueUpdateDraw(func() {
					updatePipelineTable(table, app, searchInput, apiClient, orgId)
				})
			}

			return nil
		}

		// Load all pipelines (no status filter for cache)
		err := apiClient.ListPipelinesWithCallback(orgId, callback)

		// Handle any errors
		if err != nil {
			allPipelinesCacheLoading = false
			allPipelinesCacheLoaded = false
			isPipelineLoadingInProgress = false
			pipelineLoadingComplete = false

			app.QueueUpdateDraw(func() {
				// Show error in table
				table.Clear()
				headers := []string{" ", "Name"}
				for col, header := range headers {
					cell := tview.NewTableCell(header).
						SetTextColor(tcell.ColorYellow).
						SetAlign(tview.AlignLeft).
						SetSelectable(false).
						SetBackgroundColor(tcell.ColorDefault)
					table.SetCell(0, col, cell)
				}

				cell := tview.NewTableCell(fmt.Sprintf("Error loading pipelines: %v", err)).
					SetTextColor(tcell.ColorRed).
					SetAlign(tview.AlignCenter)
				table.SetCell(1, 0, cell)
				table.SetCell(1, 1, tview.NewTableCell(""))

				table.SetTitle("Error Loading Pipelines")
			})
		}
	}()
}

// startProgressivePipelineLoadingFromServer loads pipelines directly from server (for filtered views)
func startProgressivePipelineLoadingFromServer(table *tview.Table, app *tview.Application, searchInput *tview.InputField, apiClient *api.Client, orgId string) {
	// Reset loading state
	isPipelineLoadingInProgress = true
	pipelineLoadingCurrentPage = 0
	pipelineLoadingTotalPages = 0
	pipelineLoadingComplete = false
	pipelineLoadingError = nil
	allPipelines = []api.Pipeline{} // Clear existing pipelines

	// Immediately clear table and show loading state to prevent showing old data
	table.Clear()
	headers := []string{" ", "Name"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetBackgroundColor(tcell.ColorDefault)
		table.SetCell(0, col, cell)
	}

	// Show loading message immediately
	var loadingTitle string
	if currentViewMode == "pipelines_in_group" {
		if showOnlyRunningWaiting {
			loadingTitle = fmt.Sprintf("Loading Pipelines in '%s' (RUNNING+WAITING)...", selectedGroupName)
		} else {
			loadingTitle = fmt.Sprintf("Loading Pipelines in '%s'...", selectedGroupName)
		}
	} else {
		if showOnlyRunningWaiting {
			loadingTitle = "Loading Pipelines (RUNNING+WAITING)..."
		} else {
			loadingTitle = "Loading All Pipelines..."
		}
	}
	table.SetTitle(loadingTitle)

	// Start loading in a goroutine
	go func() {
		if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
			// Handle group pipelines directly
			pipelineLoadingCurrentPage = 1 // Reset for group view
			pipelineLoadingTotalPages = 1  // Assume single load for group

			groupIdInt := 0
			if _, parseErr := fmt.Sscanf(selectedGroupID, "%d", &groupIdInt); parseErr != nil {
				pipelineLoadingError = fmt.Errorf("invalid group ID '%s'", selectedGroupID)
			} else {
				// Prepare options for group pipeline loading
				options := make(map[string]interface{})
				if showOnlyRunningWaiting {
					// Add status filter for group pipelines
					options["statusList"] = "RUNNING,WAITING"
				}

				groupPipelines, groupErr := apiClient.ListPipelineGroupPipelines(orgId, groupIdInt, options)
				if groupErr != nil {
					pipelineLoadingError = groupErr
				} else {
					allPipelines = groupPipelines // Direct assignment
					pipelineLoadingComplete = true
					isPipelineLoadingInProgress = false
					// Update UI on main thread
					app.QueueUpdateDraw(func() {
						updatePipelineTable(table, app, searchInput, apiClient, orgId)
						if table.GetRowCount() > 1 {
							table.Select(1, 0)
						}
					})
				}
			}

			if pipelineLoadingError != nil {
				isPipelineLoadingInProgress = false
				pipelineLoadingComplete = false // Loading did not complete successfully
				app.QueueUpdateDraw(func() {
					table.Clear()
					headers := []string{" ", "Name"}
					for col, header := range headers {
						cell := tview.NewTableCell(header).
							SetTextColor(tcell.ColorYellow).
							SetAlign(tview.AlignLeft).
							SetSelectable(false).
							SetBackgroundColor(tcell.ColorDefault)
						table.SetCell(0, col, cell)
					}
					cell := tview.NewTableCell(fmt.Sprintf("Error loading pipelines for group: %v", pipelineLoadingError)).
						SetTextColor(tcell.ColorRed).
						SetAlign(tview.AlignCenter)
					table.SetCell(1, 0, cell)
					// Clear other cells in the row
					for i := 1; i < len(headers); i++ {
						table.SetCell(1, i, tview.NewTableCell(""))
					}
					table.SetTitle(fmt.Sprintf("Error Loading Pipelines for Group '%s'", selectedGroupName))
				})
			}
		} else {
			// Original logic for "all pipelines" (possibly filtered by status)
			var statusList []string
			if showOnlyRunningWaiting {
				statusList = []string{"RUNNING", "WAITING"}
			}

			// Define callback function for each page
			callback := func(pipelines []api.Pipeline, currentPage, totalPages int, isComplete bool) error {
				// Update loading state
				pipelineLoadingCurrentPage = currentPage
				pipelineLoadingTotalPages = totalPages

				// Append new pipelines to the global list
				allPipelines = append(allPipelines, pipelines...)

				// Update UI on main thread
				app.QueueUpdateDraw(func() {
					// Update the table with new pipelines
					updatePipelineTable(table, app, searchInput, apiClient, orgId)

					// If this is the first page and table has content, select first row
					if currentPage == 1 && table.GetRowCount() > 1 {
						table.Select(1, 0)
					}
				})

				// Mark as complete if this is the last page
				if isComplete {
					pipelineLoadingComplete = true
					isPipelineLoadingInProgress = false

					// Final UI update - only update title, no need to rebuild entire table
					app.QueueUpdateDraw(func() {
						// Just update the title to show completion status
						var title string
						if currentViewMode == "pipelines_in_group" {
							if showOnlyBookmarked {
								title = fmt.Sprintf("Pipelines in '%s' (BOOKMARKED)", selectedGroupName)
							} else if showOnlyRunningWaiting {
								title = fmt.Sprintf("Pipelines in '%s' (RUNNING+WAITING)", selectedGroupName)
							} else {
								title = fmt.Sprintf("Pipelines in '%s'", selectedGroupName)
							}
						} else {
							if showOnlyBookmarked {
								title = "Pipelines (BOOKMARKED)"
							} else if showOnlyRunningWaiting {
								title = "Pipelines (RUNNING+WAITING)"
							} else {
								title = "All Pipelines"
							}
						}
						title += fmt.Sprintf(" (%d pipelines)", len(allPipelines))
						table.SetTitle(title)
					})
				}

				return nil
			}

			// Start the progressive loading
			var err error
			if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
				// For group pipelines, we need to use a different approach
				// since ListPipelineGroupPipelines doesn't support callback yet
				// THIS PATH SHOULD NOT BE HIT ANYMORE DUE TO THE OUTER IF/ELSE
				// BUT KEPT FOR STRUCTURAL SIMILARITY TO ORIGINAL BEFORE REFACTOR
				groupIdInt := 0
				if _, parseErr := fmt.Sscanf(selectedGroupID, "%d", &groupIdInt); parseErr != nil {
					pipelineLoadingError = fmt.Errorf("invalid group ID '%s'", selectedGroupID)
				} else {
					groupPipelines, groupErr := apiClient.ListPipelineGroupPipelines(orgId, groupIdInt, nil)
					if groupErr != nil {
						pipelineLoadingError = groupErr
					} else {
						// Call callback with all group pipelines as a single page
						err = callback(groupPipelines, 1, 1, true)
					}
				}
			} else {
				// Use the new callback-based API for all pipelines
				if len(statusList) > 0 {
					err = apiClient.ListPipelinesWithStatusAndCallback(orgId, statusList, callback)
				} else {
					err = apiClient.ListPipelinesWithCallback(orgId, callback)
				}
			}

			// Handle any errors
			// Note: pipelineLoadingError is for the group-specific loading path in the outer 'if'
			// 'err' here is for the general callback-based loading
			if err != nil || pipelineLoadingError != nil {
				finalErr := err
				if pipelineLoadingError != nil { // Should ideally not happen if err is also present from this path
					finalErr = pipelineLoadingError
				}

				isPipelineLoadingInProgress = false
				pipelineLoadingComplete = false

				app.QueueUpdateDraw(func() {
					// Show error in table
					table.Clear()
					headers := []string{" ", "Name"}
					for col, header := range headers {
						cell := tview.NewTableCell(header).
							SetTextColor(tcell.ColorYellow).
							SetAlign(tview.AlignLeft).
							SetSelectable(false).
							SetBackgroundColor(tcell.ColorDefault)
						table.SetCell(0, col, cell)
					}

					cell := tview.NewTableCell(fmt.Sprintf("Error loading pipelines: %v", finalErr)).
						SetTextColor(tcell.ColorRed).
						SetAlign(tview.AlignCenter)
					table.SetCell(1, 0, cell)
					table.SetCell(1, 1, tview.NewTableCell(""))

					table.SetTitle("Error Loading Pipelines")
				})
			}
		}
	}()
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
	showOnlyBookmarked = false
	isLogViewActive = false
	isRunHistoryActive = false

	// Initialize pipeline loading state
	isPipelineLoadingInProgress = false
	pipelineLoadingComplete = false
	allPipelines = []api.Pipeline{} // Start with empty list

	// Initialize cache state
	if !allPipelinesCacheLoaded && !allPipelinesCacheLoading {
		allPipelinesCache = []api.Pipeline{}
		allPipelinesCacheLoaded = false
		allPipelinesCacheLoading = false
	}

	var fetchErrGroups error
	allPipelineGroups, fetchErrGroups = apiClient.ListPipelineGroups(orgId)

	// UI Elements
	pipelineTable := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	pipelineTable.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
	// Enable table to receive focus and handle input
	pipelineTable.SetSelectable(true, false)
	// Set selected row background color to light gray
	pipelineTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite))
	pipelineTableGlobal = pipelineTable // Set global reference

	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetPlaceholder("Pipeline Name (Press / to focus)...").
		SetFieldWidth(0)
	searchInput.SetFieldBackgroundColor(tcell.ColorDefault) // Background of the text entry area
	searchInput.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorGray))

	// Help info
	helpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=run history, r=run, a=toggle running/all, b=toggle bookmarks, B=bookmark, Ctrl+G=groups, /=search, q=back, Q=quit").
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
	// Set selected row background color to light gray
	groupTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite))
	groupTableGlobal = groupTable

	// Group search input
	groupSearchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetPlaceholder("Group Name (Press / to focus)...").
		SetFieldWidth(0)
	groupSearchInput.SetFieldBackgroundColor(tcell.ColorDefault) // Background of the text entry area
	groupSearchInput.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorGray))
	groupSearchInputGlobal = groupSearchInput

	// Explicitly set the style for the field itself
	// groupSearchInput.SetFieldStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorWhite))

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
	// Set selected row background color to light gray
	runHistoryTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite))

	// Run history help info
	runHistoryHelpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=view logs, r=run pipeline, X=stop run, [/]=prev/next page, 0=first page, q=back to pipelines, Q=quit").
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

	// Start progressive loading of pipelines
	startProgressivePipelineLoading(pipelineTable, app, searchInput, apiClient, orgId)

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
				// For search clear, we can just update the table without reloading data
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
			startProgressivePipelineLoading(pipelineTable, app, searchInput, apiClient, orgId)
			return nil
		case 'b': // Toggle bookmark filter
			showOnlyBookmarked = !showOnlyBookmarked
			// For bookmark filter, we can just update the table without reloading data
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			return nil
		case 'B': // Toggle bookmark for current pipeline
			if rowCount > 1 && currentRow > 0 {
				if selectedPipeline, ok := pipelineRowMap[currentRow]; ok && selectedPipeline != nil {
					if globalToggleBookmark != nil && globalSaveConfig != nil {
						globalToggleBookmark(selectedPipeline.Name)
						// Save configuration
						if err := globalSaveConfig(); err != nil {
							ShowModal("Error", fmt.Sprintf("Failed to save bookmark: %v", err), []string{"OK"}, nil)
						}
						// Refresh table to update bookmark indicators (no API call needed)
						updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
					}
				}
			}
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
		// For search filtering, we can just update the table without reloading data
		updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
	})
	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyDown || key == tcell.KeyUp {
			app.SetFocus(pipelineTable)
		} else if key == tcell.KeyEscape {
			currentSearchQuery = ""
			searchInput.SetText("")
			// For search clear, we can just update the table without reloading data
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
			// Use progressive loading to switch back to all pipelines view
			startProgressivePipelineLoading(pipelineTable, app, searchInput, apiClient, orgId)
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
					startProgressivePipelineLoading(pipelineTable, app, searchInput, apiClient, orgId)
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
			// Use progressive loading to switch back to all pipelines view
			startProgressivePipelineLoading(pipelineTable, app, searchInput, apiClient, orgId)
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
		case 'X':
			// Stop/terminate pipeline run
			if rowCount > 1 && currentRow > 0 {
				if selectedRun, ok := runHistoryRowMap[currentRow]; ok && selectedRun != nil {
					// Show confirmation dialog
					ShowModal("Confirm Stop",
						fmt.Sprintf("Are you sure you want to stop pipeline run #%s?\nStatus: %s", selectedRun.RunID, selectedRun.Status),
						[]string{"Yes", "No"},
						func(buttonIndex int, buttonLabel string) {
							if buttonIndex == 0 { // Yes
								go func() {
									err := apiClient.StopPipelineRun(orgId, currentPipelineIDForRun, selectedRun.RunID)
									app.QueueUpdateDraw(func() {
										if err != nil {
											ShowModal("Error", fmt.Sprintf("Failed to stop pipeline run: %v", err), []string{"OK"}, nil)
										} else {
											ShowModal("Success", "Pipeline run stop request sent successfully.", []string{"OK"}, func(buttonIndex int, buttonLabel string) {
												// Refresh the run history table to show updated status
												updateRunHistoryTable(runHistoryTable, app, apiClient, orgId, currentPipelineIDForRun, currentPipelineName)
											})
										}
									})
								}()
							}
						})
				}
			}
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
					currentRunStatus = selectedRun.Status  // Initialize status from selected run
					originalRunStatus = selectedRun.Status // Store original status
					preserveOriginalStatus = true          // Preserve the original status for historical runs
					isLogViewActive = true
					isNewlyCreatedRun = false // Mark this as a historical run

					// Switch to log view and start progressive log loading
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

						// Start auto-refresh for this historical run (only for running/queued status)
						if selectedRun.Status == "RUNNING" || selectedRun.Status == "QUEUED" {
							// Only auto-refresh for running pipelines
							startLogAutoRefresh(app, apiClient, orgId, pipelineName, branchInfo, repoInfo)
						} else {
							// For completed runs, just fetch once with progressive loading
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
		// Handle vim-style search navigation first
		if logSearchActive {
			switch event.Rune() {
			case 'n':
				// Next search match
				nextLogSearchMatch(app)
				return nil
			case 'N':
				// Previous search match
				prevLogSearchMatch(app)
				return nil
			case '/':
				// Start vim-style search
				startLogSearch(app)
				return nil
			}

			// Handle escape to exit search
			if event.Key() == tcell.KeyEscape {
				exitLogSearch(app)
				return nil
			}
		}

		switch event.Rune() {
		case '/':
			// Start vim-style search
			if !logSearchActive {
				startLogSearch(app)
			}
			return nil
		case 'n':
			// Next search match (only if search is active)
			if logSearchActive {
				nextLogSearchMatch(app)
			}
			return nil
		case 'N':
			// Previous search match (only if search is active)
			if logSearchActive {
				prevLogSearchMatch(app)
			}
			return nil
		case 'f':
			// Page down (same as Ctrl+F)
			logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone), nil)
			return nil
		case 'b':
			// Page up (same as Ctrl+B) - but only if not exiting
			if !logSearchActive {
				logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone), nil)
			}
			return nil
		case 'd':
			// Half page down
			for i := 0; i < 10; i++ {
				logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), nil)
			}
			return nil
		case 'u':
			// Half page up
			for i := 0; i < 10; i++ {
				logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), nil)
			}
			return nil
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
		case 'X':
			// Stop/terminate pipeline run (only for running/init/waiting status)
			if currentRunID != "" && currentPipelineIDForRun != "" {
				// Check if the current run status allows termination
				status := strings.ToUpper(currentRunStatus)
				if status == "RUNNING" || status == "INIT" || status == "WAITING" || status == "QUEUED" {
					// Show confirmation dialog
					ShowModal("Confirm Stop",
						fmt.Sprintf("Are you sure you want to stop the current pipeline run?\nRun ID: %s\nStatus: %s", currentRunID, currentRunStatus),
						[]string{"Yes", "No"},
						func(buttonIndex int, buttonLabel string) {
							if buttonIndex == 0 { // Yes
								go func() {
									err := apiClient.StopPipelineRun(orgId, currentPipelineIDForRun, currentRunID)
									app.QueueUpdateDraw(func() {
										if err != nil {
											ShowModal("Error", fmt.Sprintf("Failed to stop pipeline run: %v", err), []string{"OK"}, nil)
										} else {
											ShowModal("Success", "Pipeline run stop request sent successfully.", []string{"OK"}, func(buttonIndex int, buttonLabel string) {
												// Stop auto-refresh since we terminated the run
												stopLogAutoRefresh()
												// Update status to reflect termination request
												currentRunStatus = "STOPPING"
												updateLogStatusBar()
											})
										}
									})
								}()
							}
						})
				} else {
					// Show message that run cannot be stopped
					ShowModal("Cannot Stop",
						fmt.Sprintf("Pipeline run cannot be stopped.\nCurrent status: %s\n\nOnly runs with status RUNNING, INIT, WAITING, or QUEUED can be stopped.", currentRunStatus),
						[]string{"OK"}, nil)
				}
			} else {
				ShowModal("No Active Run", "No active pipeline run to stop.", []string{"OK"}, nil)
			}
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
		case 'q':
			// Exit search mode if active, otherwise exit log view
			if logSearchActive {
				exitLogSearch(app)
				return nil
			}
			// Exit log view
			isLogViewActive = false
			// Stop auto-refresh when leaving log view
			stopLogAutoRefresh()
			// Exit search mode if active
			if logSearchActive {
				exitLogSearch(app)
			}
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

		// Handle special keys
		switch event.Key() {
		case tcell.KeyCtrlF:
			// Page down
			logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone), nil)
			return nil
		case tcell.KeyCtrlB:
			// Page up
			logViewTextView.InputHandler()(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone), nil)
			return nil
		case tcell.KeyEscape:
			// Exit search mode if active, otherwise exit log view
			if logSearchActive {
				exitLogSearch(app)
				return nil
			}
			// Exit log view
			isLogViewActive = false
			// Stop auto-refresh when leaving log view
			stopLogAutoRefresh()
			// Exit search mode if active
			if logSearchActive {
				exitLogSearch(app)
			}
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
			} else if currentPage == "logs" && !logSearchActive { // Allow search in logs page
				startLogSearch(app)
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
			// If searchInput, groupSearchInput, or logSearchInput is focused, it needs to process 'q' for typing.
			// Their own InputCapture should return event to allow typing.
			if focused == searchInput || focused == groupSearchInputGlobal || focused == logSearchInput {
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

// startLogSearch initiates vim-style search in log view
func startLogSearch(app *tview.Application) {
	if logViewTextView == nil || logPage == nil {
		return
	}

	// Create search input if it doesn't exist
	if logSearchInput == nil {
		logSearchInput = tview.NewInputField().
			SetLabel("Search: ").
			SetPlaceholder("Enter search term...").
			SetFieldWidth(0)
		logSearchInput.SetFieldBackgroundColor(tcell.ColorDefault)
		logSearchInput.SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.ColorGray))
	}

	// Store original text if not already stored
	if logOriginalText == "" {
		logOriginalText = logViewTextView.GetText(false)
	}

	// Clear previous search state
	logSearchQuery = ""
	logSearchMatches = []int{}
	logSearchCurrentIdx = -1
	logSearchActive = true

	// Set up search input handlers
	logSearchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := logSearchInput.GetText()
			if query != "" {
				performLogSearch(query, app)
			}
			// Transfer focus to log text view after pressing Enter
			// so that n/N navigation keys work
			app.SetFocus(logViewTextView)
		} else if key == tcell.KeyEscape {
			exitLogSearch(app)
		}
	})

	logSearchInput.SetChangedFunc(func(text string) {
		if text != "" {
			performLogSearch(text, app)
		} else {
			// Clear search highlighting when text is empty
			clearLogSearchHighlighting()
		}
	})

	// Add search input to log page
	logPage.Clear()
	logPage.AddItem(logSearchInput, 1, 1, true)
	logPage.AddItem(logViewTextView, 0, 1, false)
	logPage.AddItem(logStatusBar, 1, 1, false)

	app.SetFocus(logSearchInput)
}

// performLogSearch performs the actual search and highlighting
func performLogSearch(query string, app *tview.Application) {
	if logViewTextView == nil || query == "" {
		return
	}

	logSearchQuery = query
	text := logOriginalText
	if text == "" {
		text = logViewTextView.GetText(false)
		logOriginalText = text
	}

	// Find all matches (case-insensitive)
	logSearchMatches = []int{}
	queryLower := strings.ToLower(query)
	textLower := strings.ToLower(text)

	start := 0
	for {
		idx := strings.Index(textLower[start:], queryLower)
		if idx == -1 {
			break
		}
		logSearchMatches = append(logSearchMatches, start+idx)
		start = start + idx + 1
	}

	if len(logSearchMatches) > 0 {
		logSearchCurrentIdx = 0
		highlightLogSearchMatches(text, query, app)
	} else {
		logSearchCurrentIdx = -1
		// Show original text if no matches
		logViewTextView.SetText(text)
	}

	updateLogSearchStatusBar()
}

// highlightLogSearchMatches highlights all search matches in the log text
func highlightLogSearchMatches(text, query string, app *tview.Application) {
	if len(logSearchMatches) == 0 {
		return
	}

	// Create highlighted text
	var result strings.Builder
	lastEnd := 0

	for i, matchPos := range logSearchMatches {
		// Add text before this match
		result.WriteString(text[lastEnd:matchPos])

		// Add highlighted match
		if i == logSearchCurrentIdx {
			// Current match: gray background with gold foreground
			result.WriteString("[gold:gray]")
			result.WriteString(text[matchPos : matchPos+len(query)])
			result.WriteString("[-:-]")
		} else {
			// Other matches: gray background with white foreground
			result.WriteString("[white:gray]")
			result.WriteString(text[matchPos : matchPos+len(query)])
			result.WriteString("[-:-]")
		}

		lastEnd = matchPos + len(query)
	}

	// Add remaining text
	result.WriteString(text[lastEnd:])

	logViewTextView.SetText(result.String())

	// Scroll to current match if there is one
	if logSearchCurrentIdx >= 0 && logSearchCurrentIdx < len(logSearchMatches) {
		scrollToLogSearchMatch(app)
	}
}

// scrollToLogSearchMatch scrolls the log view to show the current search match
func scrollToLogSearchMatch(app *tview.Application) {
	if logViewTextView == nil || logSearchCurrentIdx < 0 || logSearchCurrentIdx >= len(logSearchMatches) {
		return
	}

	// Calculate line number of current match
	matchPos := logSearchMatches[logSearchCurrentIdx]
	text := logOriginalText
	if text == "" {
		return
	}

	// Count newlines before the match position
	lineNum := strings.Count(text[:matchPos], "\n")

	// Scroll to the line (tview uses 0-based line numbers)
	logViewTextView.ScrollTo(lineNum, 0)
}

// nextLogSearchMatch moves to the next search match
func nextLogSearchMatch(app *tview.Application) {
	if len(logSearchMatches) == 0 {
		return
	}

	logSearchCurrentIdx = (logSearchCurrentIdx + 1) % len(logSearchMatches)
	highlightLogSearchMatches(logOriginalText, logSearchQuery, app)
	updateLogSearchStatusBar()
}

// prevLogSearchMatch moves to the previous search match
func prevLogSearchMatch(app *tview.Application) {
	if len(logSearchMatches) == 0 {
		return
	}

	logSearchCurrentIdx--
	if logSearchCurrentIdx < 0 {
		logSearchCurrentIdx = len(logSearchMatches) - 1
	}
	highlightLogSearchMatches(logOriginalText, logSearchQuery, app)
	updateLogSearchStatusBar()
}

// exitLogSearch exits search mode and restores normal log view
func exitLogSearch(app *tview.Application) {
	logSearchActive = false
	logSearchQuery = ""
	logSearchMatches = []int{}
	logSearchCurrentIdx = -1

	// Clear the search input field when exiting search mode
	if logSearchInput != nil {
		logSearchInput.SetText("")
	}

	// Restore original text
	if logOriginalText != "" {
		logViewTextView.SetText(logOriginalText)
		logOriginalText = ""
	}

	// Restore original log page layout
	logPage.Clear()
	logPage.AddItem(logViewTextView, 0, 1, true)
	logPage.AddItem(logStatusBar, 1, 1, false)

	app.SetFocus(logViewTextView)
	updateLogStatusBar() // Restore normal status bar
}

// clearLogSearchHighlighting clears search highlighting but keeps search mode active
func clearLogSearchHighlighting() {
	if logViewTextView != nil && logOriginalText != "" {
		logViewTextView.SetText(logOriginalText)
	}
	logSearchMatches = []int{}
	logSearchCurrentIdx = -1
	updateLogSearchStatusBar()
}

// updateLogSearchStatusBar updates the status bar to show search information
func updateLogSearchStatusBar() {
	if logStatusBar == nil {
		return
	}

	if logSearchActive {
		var searchInfo string
		if len(logSearchMatches) > 0 {
			searchInfo = fmt.Sprintf("Search: '%s' (%d/%d matches) | 'n' next, 'N' prev, '/' search, Esc/q to exit",
				logSearchQuery, logSearchCurrentIdx+1, len(logSearchMatches))
		} else if logSearchQuery != "" {
			searchInfo = fmt.Sprintf("Search: '%s' (no matches) | Esc to exit", logSearchQuery)
		} else {
			searchInfo = "Search mode | Enter search term, Esc to exit"
		}
		logStatusBar.SetText(searchInfo)
	} else {
		// Restore normal status bar
		updateLogStatusBar()
	}
}
