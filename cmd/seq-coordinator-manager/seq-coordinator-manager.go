package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/enescakir/emoji"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gdamore/tcell/v2"
	"github.com/offchainlabs/nitro/cmd/seq-coordinator-manager/rediscoordinator"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/rivo/tview"
)

// Tview
var pages = tview.NewPages()
var app = tview.NewApplication()

// Lists
var prioritySeqList = tview.NewList().ShowSecondaryText(false)
var nonPrioritySeqList = tview.NewList().ShowSecondaryText(false)

// Forms
var addSeqForm = tview.NewForm()
var priorityForm = tview.NewForm()
var nonPriorityForm = tview.NewForm()

// Sequencer coordinator management UI data store
type manager struct {
	redisCoordinator *rediscoordinator.RedisCoordinator
	prioritiesSet    map[string]bool
	livelinessSet    map[string]bool
	priorityList     []string
	nonPriorityList  []string
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: seq-coordinator-manager [redis-url]\n")
		os.Exit(1)
	}
	redisURL := args[0]
	redisutilCoordinator, err := redisutil.NewRedisCoordinator(redisURL)
	if err != nil {
		panic(err)
	}

	seqManager := &manager{
		redisCoordinator: &rediscoordinator.RedisCoordinator{
			RedisCoordinator: redisutilCoordinator,
		},
		prioritiesSet: make(map[string]bool),
		livelinessSet: make(map[string]bool),
	}

	seqManager.refreshAllLists(ctx)
	seqManager.populateLists(ctx)

	prioritySeqList.SetSelectedFunc(func(index int, name string, second_name string, shortcut rune) {
		nonPriorityForm.Clear(true)

		n := len(seqManager.priorityList)
		priorities := make([]string, n)
		for i := 0; i < n; i++ {
			priorities[i] = strconv.Itoa(i)
		}

		target := index
		priorityForm.Clear(true)
		priorityForm.AddDropDown("Change priority to ->", priorities, index, func(priority string, selection int) {
			target = selection
		})
		priorityForm.AddButton("Update", func() {
			if target != index {
				seqManager.updatePriorityList(ctx, index, target)
			}
			priorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
			app.SetFocus(prioritySeqList)
		})
		priorityForm.AddButton("Cancel", func() {
			priorityForm.Clear(true)
			pages.SwitchToPage("Menu")
			app.SetFocus(prioritySeqList)
		})
		priorityForm.AddButton("Remove", func() {
			url := seqManager.priorityList[index]
			delete(seqManager.prioritiesSet, url)
			seqManager.updatePriorityList(ctx, index, 0)
			seqManager.priorityList = seqManager.priorityList[1:]

			priorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
			app.SetFocus(prioritySeqList)
		})
		priorityForm.SetFocus(0)
		app.SetFocus(priorityForm)
	})

	nonPrioritySeqList.SetSelectedFunc(func(index int, name string, second_name string, shortcut rune) {
		priorityForm.Clear(true)

		n := len(seqManager.priorityList)
		priorities := make([]string, n+1)
		for i := 0; i < n+1; i++ {
			priorities[i] = strconv.Itoa(i)
		}

		target := index
		nonPriorityForm.Clear(true)
		nonPriorityForm.AddDropDown("Set priority to ->", priorities, index, func(priority string, selection int) {
			target = selection
		})
		nonPriorityForm.AddButton("Update", func() {
			key := seqManager.nonPriorityList[index]
			seqManager.priorityList = append(seqManager.priorityList, key)
			seqManager.prioritiesSet[key] = true

			index = len(seqManager.priorityList) - 1
			seqManager.updatePriorityList(ctx, index, target)

			nonPriorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
			if len(seqManager.nonPriorityList) > 0 {
				app.SetFocus(nonPrioritySeqList)
			} else {
				app.SetFocus(prioritySeqList)
			}
		})
		nonPriorityForm.AddButton("Cancel", func() {
			nonPriorityForm.Clear(true)
			pages.SwitchToPage("Menu")
			app.SetFocus(nonPrioritySeqList)
		})
		nonPriorityForm.SetFocus(0)
		app.SetFocus(nonPriorityForm)
	})

	// UI design
	flex := tview.NewFlex()
	priorityHeading := tview.NewTextView().
		SetTextColor(tcell.ColorYellow).
		SetText("-----Priority List-----")
	nonPriorityHeading := tview.NewTextView().
		SetTextColor(tcell.ColorYellow).
		SetText("-----Not in priority list but online-----")
	instructions := tview.NewTextView().
		SetTextColor(tcell.ColorYellow).
		SetText("(r) to refresh\n(s) to save all changes\n(c) to switch between lists\n(a) to add sequencer\n(q) to quit\n(tab) to navigate")

	flex.SetDirection(tview.FlexRow).
		AddItem(priorityHeading, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(prioritySeqList, 0, 2, true).
			AddItem(priorityForm, 0, 3, true), 0, 12, true).
		AddItem(nonPriorityHeading, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nonPrioritySeqList, 0, 2, true).
			AddItem(nonPriorityForm, 0, 3, true), 0, 12, true).
		AddItem(instructions, 0, 3, false).SetBorder(true)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 114 {
			seqManager.refreshAllLists(ctx)
			priorityForm.Clear(true)
			nonPriorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
			app.SetFocus(prioritySeqList)
		} else if event.Rune() == 115 {
			seqManager.pushUpdates(ctx)
			priorityForm.Clear(true)
			nonPriorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
			app.SetFocus(prioritySeqList)
		} else if event.Rune() == 97 {
			addSeqForm.Clear(true)
			seqManager.addSeqPriorityForm(ctx)
			pages.SwitchToPage("Add Sequencer")
		} else if event.Rune() == 99 {
			if prioritySeqList.HasFocus() || priorityForm.HasFocus() {
				priorityForm.Clear(true)
				app.SetFocus(nonPrioritySeqList)
			} else {
				nonPriorityForm.Clear(true)
				app.SetFocus(prioritySeqList)
			}
		} else if event.Rune() == 113 {
			app.Stop()
		}
		return event
	})

	pages.AddPage("Menu", flex, true, true)
	pages.AddPage("Add Sequencer", addSeqForm, true, false)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// updatePriorityList updates the list by changing the position of seq present at `index` to target
func (sm *manager) updatePriorityList(ctx context.Context, index int, target int) {
	for i := index - 1; i >= target; i-- {
		sm.priorityList[i], sm.priorityList[i+1] = sm.priorityList[i+1], sm.priorityList[i]
	}
	for i := index + 1; i <= target; i++ {
		sm.priorityList[i], sm.priorityList[i-1] = sm.priorityList[i-1], sm.priorityList[i]
	}

	urlList := []string{}
	for url := range sm.livelinessSet {
		if _, ok := sm.prioritiesSet[url]; !ok {
			urlList = append(urlList, url)
		}
	}
	sm.nonPriorityList = urlList
}

// populateLists populates seq's in priority list and seq's that are online but not in priority
func (sm *manager) populateLists(ctx context.Context) {
	prioritySeqList.Clear()
	chosen, err := sm.redisCoordinator.CurrentChosenSequencer(ctx)
	if err != nil {
		panic(err)
	}
	for index, seqURL := range sm.priorityList {
		sec := ""
		if seqURL == chosen {
			sec = fmt.Sprintf(" %vchosen", emoji.LeftArrow)
		}
		status := fmt.Sprintf("(%d) %v ", index, emoji.RedCircle)
		if _, ok := sm.livelinessSet[seqURL]; ok {
			status = fmt.Sprintf("(%d) %v ", index, emoji.GreenCircle)
		}
		prioritySeqList.AddItem(status+seqURL+sec, "", rune(0), nil).SetSecondaryTextColor(tcell.ColorPurple)
	}

	nonPrioritySeqList.Clear()
	status := fmt.Sprintf("(-) %v ", emoji.GreenCircle)
	for _, seqURL := range sm.nonPriorityList {
		nonPrioritySeqList.AddItem(status+seqURL, "", rune(0), nil)
	}
}

// addSeqPriorityForm returns a form with fields to add a new sequencer to priority list
func (sm *manager) addSeqPriorityForm(ctx context.Context) *tview.Form {
	URL := ""
	addSeqForm.AddInputField("Sequencer URL", "", 0, nil, func(url string) {
		URL = url
	})
	addSeqForm.AddButton("Cancel", func() {
		priorityForm.Clear(true)
		sm.populateLists(ctx)
		pages.SwitchToPage("Menu")
	})
	addSeqForm.AddButton("Add", func() {
		// check if url is valid, i.e it doesnt already exist in the priority list
		if _, ok := sm.prioritiesSet[URL]; !ok && URL != "" {
			sm.prioritiesSet[URL] = true
			sm.priorityList = append(sm.priorityList, URL)
		}
		sm.populateLists(ctx)
		pages.SwitchToPage("Menu")
	})
	return addSeqForm
}

// pushUpdates pushes the local changes to the redis server
func (sm *manager) pushUpdates(ctx context.Context) {
	err := sm.redisCoordinator.UpdatePriorities(ctx, sm.priorityList)
	if err != nil {
		log.Warn("Failed to push local changes to the priority list")
	}
	sm.refreshAllLists(ctx)
}

// refreshAllLists gets the current status of all the lists displayed in the UI
func (sm *manager) refreshAllLists(ctx context.Context) {
	priorityList, err := sm.redisCoordinator.GetPriorities(ctx)
	if err != nil {
		panic(err)
	}
	sm.priorityList = priorityList
	sm.prioritiesSet = getMapfromlist(priorityList)

	livelinessList, err := sm.redisCoordinator.GetLiveliness(ctx)
	if err != nil {
		panic(err)
	}
	sm.livelinessSet = getMapfromlist(livelinessList)

	urlList := []string{}
	for url := range sm.livelinessSet {
		if _, ok := sm.prioritiesSet[url]; !ok {
			urlList = append(urlList, url)
		}
	}
	sm.nonPriorityList = urlList
}

func getMapfromlist(list []string) map[string]bool {
	mapping := make(map[string]bool)
	for _, url := range list {
		mapping[url] = true
	}
	return mapping
}
