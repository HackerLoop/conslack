package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jroimartin/gocui"

	"github.com/HackerLoop/conslack"
)

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}

	return g.SetViewOnTop(name)
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// inputWindowHeigh := int(maxY / 6)
	inputWindowHeigh := 6

	if v, err := g.SetView("discussions", 0, 0, maxX/2-1, maxY-inputWindowHeigh-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Discussions"
		v.Editable = false
		v.Wrap = true
		v.Autoscroll = true

		return nil
	}

	if v, err := g.SetView("messages", 0, 0, maxX/2-1, maxY-inputWindowHeigh-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Loading..."
		v.Editable = false
		v.Wrap = true
		v.Autoscroll = true

		// if _, err = setCurrentViewOnTop(g, "messages"); err != nil {
		//  return err
		// }

		return nil
	}

	if v, err := g.SetView("logs", maxX/2, 0, maxX-1, maxY-inputWindowHeigh-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = fmt.Sprintf("Logs")
		v.Editable = false
		v.Wrap = true
		v.Autoscroll = true

		// if _, err = setCurrentViewOnTop(g, "messages"); err != nil {
		//  return err
		// }

	}
	if v, err := g.SetView("input", 0, maxY-inputWindowHeigh, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Message"
		v.Wrap = false
		v.Autoscroll = false
		v.Editable = true

		if _, err = setCurrentViewOnTop(g, "input"); err != nil {
			return err
		}
	}

	return nil
}

var idx = 0
var channelLists = []string{"#dev-backend", "#dev-general", "#random", "#general", "@jhchabran", "#dev-glue"}

func showChannelList(g *gocui.Gui, v *gocui.View) error {
	currentClose()

	channelName = channelLists[idx]
	idx++
	if idx >= len(channelLists) {
		idx = 0
	}

	go openDiscussion(g, channelName)

	return nil
}

func historyBack(g *gocui.Gui, v *gocui.View) error {
	messagesView, err := g.View("messages")
	if err != nil {
		return err
	}

	messagesView.Autoscroll = false

	x, y := messagesView.Origin()
	if y <= 1 {
		return nil
	}
	if err := messagesView.SetOrigin(x, y-1); err != nil {
		return err
	}

	addLog(fmt.Sprintf("historyBack (x:%v, y:%v)", x, y))

	return nil
}

func historyNext(g *gocui.Gui, v *gocui.View) error {
	messagesView, err := g.View("messages")
	if err != nil {
		return err
	}

	messagesView.Autoscroll = false

	x, y := messagesView.Origin()
	newY := y + 1
	if err := messagesView.SetOrigin(x, newY); err != nil {
		return err
	}
	_, yy := messagesView.Size()
	if newY >= yy {
		messagesView.Autoscroll = true
	}
	addLog(fmt.Sprintf("historyNext (x:%v, y:%v) (as: %t)", x, y, messagesView.Autoscroll))

	return nil
}

func openDiscussionsList(g *gocui.Gui, v *gocui.View) error {
	addLog("open")
	if _, err := setCurrentViewOnTop(g, "discussions"); err != nil {
		return err
	}

	ds, err := client.GetDiscussions()
	if err != nil {
		return err
	}

	discussionsView, err := g.View("discussions")
	if err != nil {
		return err
	}

	g.Execute(func(g *gocui.Gui) error {
		for _, d := range ds {
			fmt.Fprintf(discussionsView, "%s\n", d.Name)
		}

		return nil
	})

	return nil
}

func closeDiscussionsList(g *gocui.Gui, v *gocui.View) error {
	addLog("close")
	if _, err := setCurrentViewOnTop(g, "messages"); err != nil {
		return err
	}

	if _, err := setCurrentViewOnTop(g, "input"); err != nil {
		return err
	}

	return nil
}
func sentMessage(g *gocui.Gui, v *gocui.View) error {
	text := v.ViewBuffer()
	if text == "" {
		return nil
	}

	// to remove the latest \n
	text = text[0 : len(text)-2]

	v.Clear()
	v.SetCursor(0, 0)

	client.PostMessage(channelName, text)

	return nil
}

var (
	channelName  = "#general"
	client       *conslack.Client
	currentClose = func() {}
	gui          *gocui.Gui
)

func main() {
	c, err := conslack.New(os.Getenv("SLACK_TOKEN"))
	if err != nil {
		logrus.WithError(err).Fatal("unable to connect to slack")
	}

	client = c
	if len(os.Args) > 1 {
		channelName = os.Args[1]
	}

	g, err := gocui.NewGui()
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	gui = g

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen
	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, showChannelList); err != nil {
		log.Panicln(err)
	}

	// if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, showPreviousMessages); err != nil {
	//  log.Panicln(err)
	// }

	// if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, showNextMessages); err != nil {
	//  log.Panicln(err)
	// }

	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, historyBack); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, historyNext); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyF1, gocui.ModNone, openDiscussionsList); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("discussions", gocui.KeyEnter, gocui.ModNone, closeDiscussionsList); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, sentMessage); err != nil {
		log.Panicln(err)
	}

	go func() {
		// FIXME
		time.Sleep(100 * time.Millisecond)
		openDiscussion(g, channelName)
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func openDiscussion(g *gocui.Gui, channelName string) {
	addLog("Opening discussion: " + channelName)

	messagesView, err := g.View("messages")
	if err != nil {
		logrus.WithError(err).Fatal("unable to get messages view")
	}

	messagesView.Clear()
	messagesView.SetCursor(0, 0)
	messagesView.SetOrigin(0, 0)
	messagesView.Title = "Discussion: " + channelName

	messages, err := client.History(channelName)
	if err != nil {
		logrus.WithError(err).Fatal("unable to get history")
	}

	g.Execute(func(g *gocui.Gui) error {
		for _, m := range messages {
			addMessage(messagesView, m)
		}

		return nil
	})
	ch, close, err := client.Messages(channelName)
	if err != nil {
		logrus.WithError(err).Fatal("unable to get realtime messages")
	}
	currentClose = close

	for m := range ch {
		g.Execute(func(g *gocui.Gui) error {
			fmt.Fprintf(messagesView, "%s: %s - %s\n", m.Date, m.From, m.Text)
			return nil
		})
	}
	addLog("Ending discussion: " + channelName)
}

func addMessage(v *gocui.View, m conslack.Message) {
	fmt.Fprintf(v, "%s: %s - %s\n", m.Date, m.From, m.Text)
}

func addLog(msg string) {
	v, err := gui.View("logs")
	if err != nil {
		return
	}

	gui.Execute(func(g *gocui.Gui) error {
		fmt.Fprintf(v, "%s\n", msg)
		return nil
	})
}
