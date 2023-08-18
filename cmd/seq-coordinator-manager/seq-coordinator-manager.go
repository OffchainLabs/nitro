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

// Sequencer coordinator managment UI data store
type manager struct {
	redisCoordinator *rediscoordinator.RedisCoordinator
	prioritiesMap    map[string]int
	livelinessMap    map[string]int
	priorityList     []string
	nonPriorityList  []string
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: redis-seq-manager [redis-url]\n")
		os.Exit(1)
	}
	redisURL := args[0]
	redisCoordinator, err := rediscoordinator.NewRedisCoordinator(redisURL)
	if err != nil {
		panic(err)
	}

	seqManager := &manager{
		redisCoordinator: redisCoordinator,
		prioritiesMap:    make(map[string]int),
		livelinessMap:    make(map[string]int),
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
		priorityForm.AddTextView("Additional details:", "Status:\nBlockNumber:", 0, 2, false, true)
		priorityForm.AddDropDown("Change priority to ->", priorities, index, func(priority string, selection int) {
			target = selection
		})
		priorityForm.AddButton("Save", func() {
			if target != index {
				seqManager.updatePriorityList(ctx, index, target)
			}
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
		})
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
		nonPriorityForm.AddTextView("Additional details:", "Status:\nBlockNumber:", 0, 2, false, true)
		nonPriorityForm.AddDropDown("Set priority to ->", priorities, index, func(priority string, selection int) {
			target = selection
		})
		nonPriorityForm.AddButton("Save", func() {
			seqManager.priorityList = append(seqManager.priorityList, seqManager.nonPriorityList[index])
			index = len(seqManager.priorityList) - 1
			seqManager.updatePriorityList(ctx, index, target)
			nonPriorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
		})
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
		SetText("(r) to refresh \n(a) to add sequencer\n(q) to quit")

	flex.SetDirection(tview.FlexRow).
		AddItem(priorityHeading, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(prioritySeqList, 0, 2, true).
			AddItem(priorityForm, 0, 3, false), 0, 12, false).
		AddItem(nonPriorityHeading, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nonPrioritySeqList, 0, 2, true).
			AddItem(nonPriorityForm, 0, 3, false), 0, 12, false).
		AddItem(instructions, 0, 2, false).SetBorder(true)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 114 {
			seqManager.refreshAllLists(ctx)
			priorityForm.Clear(true)
			nonPriorityForm.Clear(true)
			seqManager.populateLists(ctx)
			pages.SwitchToPage("Menu")
		} else if event.Rune() == 97 {
			addSeqForm.Clear(true)
			seqManager.addSeqPriorityForm(ctx)
			pages.SwitchToPage("Add Sequencer")
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
	err := sm.redisCoordinator.UpdatePriorities(ctx, sm.priorityList)
	if err != nil {
		log.Warn("Failed to update priority, reverting change", "sequencer", sm.priorityList[target], "err", err)
	}
	sm.refreshAllLists(ctx)
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
		status := fmt.Sprintf("%v ", emoji.RedCircle)
		if _, ok := sm.livelinessMap[seqURL]; ok {
			status = fmt.Sprintf("%v ", emoji.GreenCircle)
		}
		prioritySeqList.AddItem(status+seqURL+sec, "", rune(48+index), nil).SetSecondaryTextColor(tcell.ColorPurple)
	}

	nonPrioritySeqList.Clear()
	status := fmt.Sprintf("%v ", emoji.GreenCircle)
	for _, seqURL := range sm.nonPriorityList {
		nonPrioritySeqList.AddItem(status+seqURL, "", rune(45), nil)
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
		if _, ok := sm.prioritiesMap[URL]; !ok && URL != "" {
			sm.priorityList = append(sm.priorityList, URL)
			err := sm.redisCoordinator.UpdatePriorities(ctx, sm.priorityList)
			if err != nil {
				log.Warn("Failed to add sequencer to the priority list", URL)
			}
			sm.refreshAllLists(ctx)
		}
		sm.populateLists(ctx)
		pages.SwitchToPage("Menu")
	})
	return addSeqForm
}

// refreshAllLists gets the current status of all the lists displayed in the UI
func (sm *manager) refreshAllLists(ctx context.Context) {
	sequencerURLList, mapping, err := sm.redisCoordinator.GetPriorities(ctx)
	if err != nil {
		panic(err)
	}
	sm.priorityList = sequencerURLList
	sm.prioritiesMap = mapping

	mapping, err = sm.redisCoordinator.GetLivelinessMap(ctx)
	if err != nil {
		panic(err)
	}
	sm.livelinessMap = mapping

	urlList := []string{}
	for url := range sm.livelinessMap {
		if _, ok := sm.prioritiesMap[url]; !ok {
			urlList = append(urlList, url)
		}
	}
	sm.nonPriorityList = urlList
}
