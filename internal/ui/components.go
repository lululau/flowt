package ui

import (
	"aliyun-pipelines-tui/internal/api"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	allPipelines      []api.Pipeline
	allPipelineGroups []api.PipelineGroup
	currentViewMode   string // "all_pipelines", "group_list", "pipelines_in_group"
	selectedGroupID   string
	selectedGroupName string

	currentSearchQuery string

	// Maps to store references for table rows
	pipelineRowMap = make(map[int]*api.Pipeline)
	groupRowMap    = make(map[int]*api.PipelineGroup)

	// New state variables for current run and log view
	currentRunID            string
	currentPipelineIDForRun string
	currentPipelineName     string
	isLogViewActive         bool
	isRunHistoryActive      bool
	logViewTextView         *tview.TextView
	logPage                 *tview.Flex        // Flex layout for the log page
	runHistoryTable         *tview.Table       // Table for pipeline run history
	runHistoryPage          *tview.Flex        // Flex layout for the run history page
	pipelineTableGlobal     *tview.Table       // To allow focus from modal
	groupTableGlobal        *tview.Table       // For group list table
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

// updatePipelineTable filters and updates the pipeline table widget.
func updatePipelineTable(table *tview.Table, app *tview.Application, _ *tview.InputField, apiClient *api.Client, orgId string) {
	table.Clear()
	pipelineTableGlobal = table // Update global reference

	var title string
	if currentViewMode == "pipelines_in_group" {
		title = fmt.Sprintf("Pipelines in '%s'", selectedGroupName)
	} else {
		title = "All Pipelines"
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

	// 1. Get pipelines based on current view mode
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
		// Use all pipelines for "all_pipelines" view
		tempFilteredByGroup = append(tempFilteredByGroup, allPipelines...)
	}

	// 2. Filter by search query (case-insensitive)
	tempFilteredBySearch := make([]api.Pipeline, 0)
	if currentSearchQuery != "" {
		sqLower := strings.ToLower(currentSearchQuery)
		for _, p := range tempFilteredByGroup {
			if strings.Contains(strings.ToLower(p.Name), sqLower) || strings.Contains(strings.ToLower(p.PipelineID), sqLower) {
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

	// Populate the table
	if len(allPipelineGroups) == 0 {
		// Show "no data" message
		cell := tview.NewTableCell("No pipeline groups found.").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter)
		table.SetCell(1, 0, cell)
		table.SetCell(1, 1, tview.NewTableCell(""))
	} else {
		for i, g := range allPipelineGroups {
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
	selectedGroupID = ""
	selectedGroupName = ""
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
		SetText("Keys: j/k=move, Enter=run history, r=run, Ctrl+G=groups, /=search, q=back, Q=quit").
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

	// Group help info
	groupHelpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=select group, q=back to all pipelines, Q=quit").
		SetTextAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorGray)
	groupHelpInfo.SetBackgroundColor(tcell.ColorDefault)

	groupListFlexView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(groupTable, 0, 1, true).
		AddItem(groupHelpInfo, 1, 1, false)

	// Log View elements
	logViewTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() }) // Redraw on text change for scrolling
	logViewTextView.SetBorder(true).SetTitle("Logs").SetBackgroundColor(tcell.ColorDefault)

	logPage = tview.NewFlex().AddItem(logViewTextView, 0, 1, true) // TextView takes all space, is focus target

	// Run History View elements
	runHistoryTable = tview.NewTable().SetBorders(false).SetSelectable(true, false)
	runHistoryTable.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
	// Enable table to receive focus and handle input
	runHistoryTable.SetSelectable(true, false)

	// Run history help info
	runHistoryHelpInfo := tview.NewTextView().
		SetText("Keys: j/k=move, Enter=view logs, [/]=prev/next page, 0=first page, q=back to pipelines, Q=quit").
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
					currentPipelineIDForRun = selectedPipeline.PipelineID
					// Show confirmation or directly run
					ShowModal("Run Pipeline?", fmt.Sprintf("Run '%s'?", selectedPipeline.Name), []string{"Run", "Cancel"},
						func(buttonIndex int, buttonLabel string) {
							if buttonLabel == "Run" {
								go func() { // Run in goroutine to avoid blocking UI
									app.QueueUpdateDraw(func() {
										logViewTextView.SetText("Initiating pipeline run...")
										mainPages.SwitchToPage("logs")
										app.SetFocus(logViewTextView)
									})

									runResponse, err := apiClient.RunPipeline(orgId, selectedPipeline.PipelineID, nil)
									if err != nil {
										app.QueueUpdateDraw(func() {
											ShowModal("Error", fmt.Sprintf("Failed to run pipeline: %v", err), []string{"OK"}, nil)
										})
										return
									}
									currentRunID = runResponse.RunID
									isLogViewActive = true

									app.QueueUpdateDraw(func() {
										logViewTextView.SetText(fmt.Sprintf("Pipeline '%s' triggered. Run ID: %s\nFetching run details...\n", selectedPipeline.Name, currentRunID))
										logViewTextView.ScrollToEnd()
									})

									// Fetch initial run details
									runDetails, err := apiClient.GetPipelineRun(orgId, currentPipelineIDForRun, currentRunID)
									app.QueueUpdateDraw(func() {
										if err != nil {
											fmt.Fprintf(logViewTextView, "\nError getting run details: %v\n", err)
										} else {
											fmt.Fprintf(logViewTextView, "\nRun ID: %s\nStatus: %s\nTrigger: %s\nStart: %s\nFinish: %s\n\nFetching logs is not fully implemented yet. Requires JobID.\n",
												runDetails.RunID, runDetails.Status, runDetails.TriggerMode, runDetails.StartTime.String(), runDetails.FinishTime.String())
										}
										logViewTextView.ScrollToEnd()
									})
								}()
							}
						})
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
			updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineTable)
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
		}
		switch event.Key() {
		case tcell.KeyEnter:
			if rowCount > 1 && currentRow > 0 {
				if selectedRun, ok := runHistoryRowMap[currentRow]; ok && selectedRun != nil {
					currentRunID = selectedRun.RunID
					isLogViewActive = true

					// Switch to log view and fetch logs
					go func() {
						app.QueueUpdateDraw(func() {
							logViewTextView.SetText(fmt.Sprintf("Fetching logs for run %s...", currentRunID))
							mainPages.SwitchToPage("logs")
							app.SetFocus(logViewTextView)
						})

						// Fetch logs for this run
						logs, err := apiClient.GetPipelineRunLogs(orgId, currentPipelineIDForRun, currentRunID)
						app.QueueUpdateDraw(func() {
							if err != nil {
								logViewTextView.SetText(fmt.Sprintf("Error fetching logs: %v\n\nNote: Log fetching may require additional JobID parameter which is not yet implemented.", err))
							} else {
								logViewTextView.SetText(logs)
							}
							logViewTextView.ScrollToEnd()
						})
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
		if event.Key() == tcell.KeyEscape || event.Rune() == 'b' || event.Rune() == 'q' {
			isLogViewActive = false
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
			if currentPage == "pipelines" { // Only allow search focus if on pipelines page
				app.SetFocus(searchInput)
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
			// If searchInput is focused, it needs to process 'q' for typing.
			// searchInput's own InputCapture should return event to allow typing.
			if focused == searchInput {
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
