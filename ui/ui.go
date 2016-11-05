package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hackerloop/conslack"
	"github.com/jroimartin/gocui"
)

// App ...
type App struct {
	c            *conslack.Client
	g            *gocui.Gui
	s            *state
	logFile      *os.File
	maxX         int
	maxY         int
	currentTopic struct {
		name  string
		close func()
	}
}

// a topic is either a channel, a private channel or a person
type topic struct {
	messages []conslack.Message
}

type state struct {
	topics map[string]*topic
}

const (
	inputHeight     = 2
	jumpWidth       = 40 // TODO use ratio
	jumpHeightRatio = 0.3
)

// NewApp ...
func NewApp(client *conslack.Client) (*App, error) {
	g, err := gocui.NewGui()
	if err != nil {
		return nil, err
	}

	return &App{
		c: client,
		g: g,
	}, nil
}

// Close ...
func (a *App) Close() {
	a.g.Close()
}

// Loop runs the ui main lopp
func (a *App) Loop() error {
	if err := a.prepare(); err != nil {
		return err
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := a.openTopic(a.g, "#general"); err != nil {
			logrus.WithError(err).Fatal("fu")
		}
	}()

	if err := a.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	return a.g.MainLoop()
}

func (a *App) prepare() error {
	a.g.Highlight = true
	a.g.Cursor = true
	a.g.SelFgColor = gocui.ColorGreen

	a.maxX, a.maxY = a.g.Size()

	a.g.SetManagerFunc(a.layout)

	if err := a.g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, a.sendMessage); err != nil {
		return err
	}

	a.s = &state{
		topics: make(map[string]*topic),
	}

	return nil
}

func (a *App) sendMessage(g *gocui.Gui, v *gocui.View) error {
	message := v.ViewBuffer()
	message = strings.TrimSpace(message)

	err := a.c.PostMessage(a.currentTopic.name, message)
	if err != nil {
		return err
	}

	v.Clear()
	v.SetCursor(0, 0)

	return nil
}

func formatMessage(m *conslack.Message) string {
	return fmt.Sprintf("%s %s %s\n", m.Date, m.From, m.Text)
}

func (a *App) openTopic(g *gocui.Gui, topicName string) error {
	v, err := g.View("messages")
	if err != nil {
		logrus.WithError(err).Error("unable to get messages view")
		return err
	}

	v.Clear()
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)

	h, err := g.View("header")
	if err != nil {
		logrus.WithError(err).Error("unable to get messages status")
		return err
	}

	g.Execute(func(g *gocui.Gui) error {
		fmt.Fprintf(h, topicName)
		return nil
	})

	messages, err := a.c.History(topicName)
	if err != nil {
		logrus.WithError(err).Error("unable to get history")
		return err
	}

	g.Execute(func(g *gocui.Gui) error {
		for _, m := range messages {
			fmt.Fprintf(v, formatMessage(&m))
		}

		return nil
	})

	ch, close, err := a.c.Messages(topicName)
	if err != nil {
		logrus.WithError(err).Fatal("unable to get realtime messages")
	}

	a.currentTopic.name = topicName
	a.currentTopic.close = close

	for m := range ch {
		g.Execute(func(g *gocui.Gui) error {
			fmt.Fprintf(v, formatMessage(&m))
			return nil
		})
	}

	return nil
}

func (a *App) layout(g *gocui.Gui) error {
	jumpX1 := a.maxX/2 - jumpWidth/2
	jumpY1 := int(float32(a.maxY) * jumpHeightRatio)
	jumpX2 := a.maxX/2 + jumpWidth/2
	jumpY2 := int(float32(a.maxY) - float32(a.maxY)*jumpHeightRatio)

	if v, err := g.SetView("jump.container", jumpX1, jumpY1, jumpX2, jumpY2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = false
		v.Wrap = false
		v.Frame = true
		v.Overwrite = true

		g.Execute(func(g *gocui.Gui) error {
			v.FgColor = gocui.ColorWhite
			return nil
		})
	}

	if v, err := g.SetView("jump.input", jumpX1, jumpY1, jumpX1+jumpWidth, jumpY1+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = true
		v.Wrap = false
		v.Frame = false
		v.Overwrite = true

		g.Execute(func(g *gocui.Gui) error {
			v.FgColor = gocui.ColorWhite
			return nil
		})
	}

	if v, err := g.SetView("header", -1, -1, a.maxX, a.maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = false
		v.Wrap = false
		v.Frame = false
		v.Overwrite = true

		g.Execute(func(g *gocui.Gui) error {
			v.BgColor = gocui.ColorBlue
			return nil
		})
	}

	if v, err := g.SetView("messages", -1, 0, a.maxX, a.maxY-inputHeight-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Loading..."
		v.Editable = false
		v.Wrap = true
		v.Frame = false
		v.Autoscroll = true
		v.Overwrite = true

		return nil
	}

	if v, err := g.SetView("status", -1, a.maxY-inputHeight-3, a.maxX, a.maxY-inputHeight); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = false
		v.Wrap = false
		v.Frame = false

		g.Execute(func(g *gocui.Gui) error {
			v.BgColor = gocui.ColorBlue
			return nil
		})
	}

	if v, err := g.SetView("input", -1, a.maxY-inputHeight-2, a.maxX, a.maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = true
		v.Wrap = true
		v.Frame = false
		v.Overwrite = true

		_, err := g.SetCurrentView("input")
		if err != nil {
			return err
		}

		_, err = g.SetViewOnTop("input")
		if err != nil {
			return err
		}

		g.Execute(func(g *gocui.Gui) error {
			v.FgColor = gocui.ColorWhite
			return nil
		})
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
