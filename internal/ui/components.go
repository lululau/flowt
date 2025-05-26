package ui

import (
	"aliyun-pipelines-tui/internal/api"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	allPipelines        []api.Pipeline
	allPipelineGroups   []api.PipelineGroup
	currentViewMode     string // "all_pipelines", "group_list", "pipelines_in_group"
	selectedGroupID     string
	selectedGroupName   string

	currentSearchQuery  string
	currentStatusFilter string
	statusesToCycle     = []string{"ALL", "SUCCESS", "RUNNING", "FAILED", "CANCELED"}
	currentStatusIndex  = 0

	// New state variables for current run and log view
	currentRunID              string
	currentPipelineIDForRun   string
	isLogViewActive           bool
	logViewTextView           *tview.TextView
	logPage                   *tview.Flex // Flex layout for the log page
	pipelineListGlobal        *tview.List // To allow focus from modal
	mainPagesGlobal           *tview.Pages // To allow modal to be added/removed
	appGlobal                 *tview.Application // For setting focus from modal
)

// ShowModal displays a modal dialog.
func ShowModal(title, text string, buttons []string, doneFunc func(buttonIndex int, buttonLabel string)) {
	if mainPagesGlobal == nil || appGlobal == nil {
		// Should not happen if app is initialized properly
		return
	}
	modal := tview.NewModal().
		SetText(text).
		SetTitle(title).
		AddButtons(buttons).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
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
	// Try to restore focus to a sensible default, like the pipeline list
	if pipelineListGlobal != nil && (currentViewMode == "all_pipelines" || currentViewMode == "pipelines_in_group") {
		appGlobal.SetFocus(pipelineListGlobal)
	} else if currentViewMode == "group_list" { // Assuming groupListGlobal exists and is similar
		// appGlobal.SetFocus(groupListGlobal) // Needs groupListGlobal
	}
}


// updatePipelineList filters and updates the pipeline list widget.
// SIMULATION NOTE: Filtering by selectedGroupID currently simulates by checking if pipeline.Name contains selectedGroupName,
// as api.Pipeline does not have GroupID populated by ListPipelines.
func updatePipelineList(list *tview.List, app *tview.Application, _ *tview.InputField) {
	list.Clear() // Clear current items
	pipelineListGlobal = list // Update global reference

	var title string
	if currentViewMode == "pipelines_in_group" {
		title = fmt.Sprintf("Pipelines in '%s' (Filter: %s)", selectedGroupName, currentStatusFilter)
	} else {
		title = fmt.Sprintf("All Pipelines (Filter: %s)", currentStatusFilter)
	}
	list.SetTitle(title)

	// 1. Filter by selected group (simulation)
	tempFilteredByGroup := make([]api.Pipeline, 0)
	if currentViewMode == "pipelines_in_group" && selectedGroupID != "" {
		for _, p := range allPipelines {
			// SIMULATION: Using selectedGroupName against pipeline.Name
			// A real implementation would check p.GroupID == selectedGroupID
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(selectedGroupName)) {
				tempFilteredByGroup = append(tempFilteredByGroup, p)
			}
		}
	} else {
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

	// 3. Filter by status
	finalFilteredPipelines := make([]api.Pipeline, 0)
	if currentStatusFilter != "ALL" && currentStatusFilter != "" {
		for _, p := range tempFilteredBySearch {
			if strings.EqualFold(p.Status, currentStatusFilter) {
				finalFilteredPipelines = append(finalFilteredPipelines, p)
			}
		}
	} else {
		finalFilteredPipelines = append(finalFilteredPipelines, tempFilteredBySearch...)
	}

	// Populate the list
	if len(finalFilteredPipelines) == 0 {
		list.AddItem("No pipelines match filters.", "", 0, nil)
	} else {
		for i, p := range finalFilteredPipelines {
			pipelineCopy := p // Important: capture range variable for reference
			mainText := pipelineCopy.Name
			if mainText == "" {
				mainText = pipelineCopy.PipelineID
			}
			var shortcut rune
			if i < 9 {
				shortcut = rune(fmt.Sprintf("%d", i+1)[0])
			}
			secondaryText := fmt.Sprintf("ID: %s, Status: %s", pipelineCopy.PipelineID, pipelineCopy.Status)
			if mainText == pipelineCopy.PipelineID {
				secondaryText = fmt.Sprintf("Status: %s", pipelineCopy.Status)
			}
			// Store the pipeline object itself as reference
			list.AddItem(mainText, secondaryText, shortcut, nil).SetItemReference(list.GetItemCount()-1, pipelineCopy)
		}
	}
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}
}

// NewMainView creates the main layout for the application.
func NewMainView(app *tview.Application, apiClient *api.Client, orgId string) tview.Primitive {
	// Initialize global references for modal helpers
	appGlobal = app
	
	currentViewMode = "all_pipelines"
	currentSearchQuery = ""
	currentStatusFilter = "ALL"
	currentStatusIndex = 0
	selectedGroupID = ""
	selectedGroupName = ""
	isLogViewActive = false


	var fetchErrPipelines error
	allPipelines, fetchErrPipelines = apiClient.ListPipelines(orgId)

	var fetchErrGroups error
	allPipelineGroups, fetchErrGroups = apiClient.ListPipelineGroups(orgId)

	// UI Elements
	pipelineList := tview.NewList().SetSelectedFocusOnly(true)
	pipelineListGlobal = pipelineList // Set global reference
	
	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetPlaceholder("Pipeline name/ID (Ctrl+F to focus)...").
		SetFieldWidth(0)
	
	pipelineListFlexView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 1, 1, false).
		AddItem(pipelineList, 0, 1, false)

	groupList := tview.NewList().SetSelectedFocusOnly(true)
	groupList.SetBorder(true).SetTitle("Pipeline Groups")
	// groupListGlobal = groupList // For HideModal focus restoration

	if fetchErrGroups != nil {
		groupList.AddItem(fmt.Sprintf("Error fetching groups: %v", fetchErrGroups), "", 0, nil)
	} else if len(allPipelineGroups) == 0 {
		groupList.AddItem("No pipeline groups found.", "", 0, nil)
	} else {
		for i, g := range allPipelineGroups {
			groupCopy := g
			var shortcut rune = 0
			if i < 9 { shortcut = rune(fmt.Sprintf("%d", i+1)[0]) }
			groupList.AddItem(groupCopy.Name, fmt.Sprintf("ID: %s", groupCopy.GroupID), shortcut, nil).SetItemReference(groupList.GetItemCount()-1, groupCopy)
		}
	}
	
	// Log View elements
	logViewTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() }) // Redraw on text change for scrolling
	logViewTextView.SetBorder(true).SetTitle("Logs")

	logPage = tview.NewFlex().AddItem(logViewTextView, 0, 1, true) // TextView takes all space, is focus target

	// Main pages
	mainPages := tview.NewPages().
		AddPage("pipelines", pipelineListFlexView, true, true).
		AddPage("groups", groupList, true, false).
		AddPage("logs", logPage, true, false) // Log page, initially not visible
	mainPagesGlobal = mainPages // Set global reference for modals

	// Initial population of the pipeline list
	if fetchErrPipelines != nil {
		pipelineList.Clear()
		pipelineList.AddItem(fmt.Sprintf("Error fetching pipelines: %v", fetchErrPipelines), "", 0, nil)
	} else {
		updatePipelineList(pipelineList, app, searchInput)
	}
	
	// --- Event Handlers for pipelineList ---
	pipelineList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentIndex := pipelineList.GetCurrentItem()
		itemCount := pipelineList.GetItemCount()

		switch event.Rune() {
		case 'j':
			if itemCount > 0 { pipelineList.SetCurrentItem((currentIndex + 1) % itemCount) }
			return nil
		case 'k':
			if itemCount > 0 { pipelineList.SetCurrentItem((currentIndex - 1 + itemCount) % itemCount) }
			return nil
		case 'r': // Run pipeline
			if itemCount > 0 {
				ref := pipelineList.GetItemReference(currentIndex)
				if selectedPipeline, ok := ref.(api.Pipeline); ok {
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
			// TODO: Show pipeline details/runs view (future subtask)
			return nil
		case tcell.KeyEscape:
			if currentViewMode == "pipelines_in_group" {
				currentViewMode = "group_list"
				mainPages.SwitchToPage("groups")
				app.SetFocus(groupList)
				return nil
			}
		}
		return event
	})

	// --- Event Handlers for searchInput ---
	searchInput.SetChangedFunc(func(text string) {
		currentSearchQuery = text
		updatePipelineList(pipelineList, app, searchInput)
	})
	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyDown || key == tcell.KeyUp {
			app.SetFocus(pipelineList)
		} else if key == tcell.KeyEscape {
			currentSearchQuery = ""
			searchInput.SetText("")
			updatePipelineList(pipelineList, app, searchInput)
			app.SetFocus(pipelineList)
		}
	})

	// --- Event Handlers for groupList ---
	groupList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentIndex := groupList.GetCurrentItem()
		itemCount := groupList.GetItemCount()
		switch event.Rune() {
		case 'j':
			if itemCount > 0 { groupList.SetCurrentItem((currentIndex + 1) % itemCount) }
			return nil
		case 'k':
			if itemCount > 0 { groupList.SetCurrentItem((currentIndex - 1 + itemCount) % itemCount) }
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			if itemCount > 0 {
				ref := groupList.GetItemReference(currentIndex)
				if selectedGroup, ok := ref.(api.PipelineGroup); ok {
					selectedGroupID = selectedGroup.GroupID
					selectedGroupName = selectedGroup.Name
					currentViewMode = "pipelines_in_group"
					currentSearchQuery = "" 
					searchInput.SetText("")
					updatePipelineList(pipelineList, app, searchInput)
					mainPages.SwitchToPage("pipelines")
					app.SetFocus(pipelineList)
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyEscape { 
			currentViewMode = "all_pipelines"
			selectedGroupID = ""
			selectedGroupName = ""
			updatePipelineList(pipelineList, app, searchInput)
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineList)
			return nil
		}
		return event
	})
	
	// --- Event Handlers for logViewTextView ---
	logViewTextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'b' {
			isLogViewActive = false
			mainPages.SwitchToPage("pipelines")
			app.SetFocus(pipelineList) // Or pipelineListFlexView
			return nil
		}
		// Allow default scrolling for arrow keys, PageUp/Down etc.
		return event
	})


	// Global keybindings (Ctrl+F, Ctrl+S, Ctrl+G) on mainPages
	mainPages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentPage, _ := mainPages.GetFrontPage()
		switch event.Key() {
		case tcell.KeyCtrlF:
			if currentPage == "pipelines" { // Only allow search focus if on pipelines page
				app.SetFocus(searchInput)
				return nil
			}
		case tcell.KeyCtrlS: 
			if currentPage == "pipelines" {
				currentStatusIndex = (currentStatusIndex + 1) % len(statusesToCycle)
				currentStatusFilter = statusesToCycle[currentStatusIndex]
				updatePipelineList(pipelineList, app, searchInput)
				return nil
			}
		case tcell.KeyCtrlG: 
			if currentPage == "pipelines" {
				currentViewMode = "group_list"
				if groupList.GetItemCount() > 0 { groupList.SetCurrentItem(0) }
				mainPages.SwitchToPage("groups")
				app.SetFocus(groupList)
			} else if currentPage == "groups" {
				currentViewMode = "all_pipelines" 
				selectedGroupID = ""
				selectedGroupName = ""
				updatePipelineList(pipelineList, app, searchInput)
				mainPages.SwitchToPage("pipelines")
				app.SetFocus(pipelineList) 
			}
			return nil
		}
		return event
	})
	
	app.SetFocus(pipelineListFlexView)

	return mainPages
}
