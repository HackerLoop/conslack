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
	a.g.SelFgColor = gocui.ColorBlack
	a.g.SelBgColor = gocui.ColorWhite
	a.g.Mouse = false
	a.g.InputEsc = true

	a.maxX, a.maxY = a.g.Size()

	a.g.SetManagerFunc(a.layout)

	if err := a.g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, a.sendMessage); err != nil {
		return err
	}

	if err := a.g.SetKeybinding("", gocui.KeyCtrlT, gocui.ModNone, a.jumpToTopic); err != nil {
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

func (a *App) jumpToTopic(g *gocui.Gui, v *gocui.View) error {
	ticker := time.NewTicker(100 * time.Millisecond)

	closeFn := func(g *gocui.Gui, _v *gocui.View) error {
		ticker.Stop()

		if _, err := g.SetViewOnTop("messages"); err != nil {
			return err
		}

		if _, err := g.SetViewOnTop("status"); err != nil {
			return err
		}

		if _, err := g.SetViewOnTop("input"); err != nil {
			return err
		}

		if _, err := g.SetCurrentView("input"); err != nil {
			return err
		}

		return nil
	}

	// TODO https://github.com/nsf/termbox-go/issues/120
	if err := g.SetKeybinding("jump.container", gocui.KeyEsc, gocui.ModNone, closeFn); err != nil {
		logrus.Error("can't assign key")
		return err
	}
	if err := g.SetKeybinding("jump.input", gocui.KeyEsc, gocui.ModNone, closeFn); err != nil {
		logrus.Error("can't assign key")
		return err
	}

	// if err := g.SetKeybinding("jump.input", gocui.KeyEnter, gocui.ModNone, jumpToSelected()); err != nil {
	// 	logrus.Error("can't assign key")
	// 	return err
	// }

	c, err := g.SetViewOnTop("jump.container")
	if err != nil {
		return err
	}
	c.Clear()

	i, err := g.SetViewOnTop("jump.input")
	if err != nil {
		return err
	}
	i.Clear()
	i.SetCursor(0, 0)

	// TODO rename to topics
	discussions, err := a.c.GetDiscussions()
	logrus.Infof("%#v", discussions)
	if err != nil {
		return err
	}

	nextTopicFn := func(g *gocui.Gui, v *gocui.View) error {
		logrus.Info("next")
		g.Execute(func(g *gocui.Gui) error {
			c.MoveCursor(0, 1, false)
			return nil
		})

		return nil
	}
	prevTopicFn := func(g *gocui.Gui, v *gocui.View) error {
		logrus.Info("prev")
		g.Execute(func(g *gocui.Gui) error {
			c.MoveCursor(0, -1, false)
			return nil
		})

		return nil
	}

	enterFn := func(g *gocui.Gui, v *gocui.View) error {
		logrus.Info("prev")

		_, y := c.Cursor()
		topic, err := c.Line(y)
		if err != nil {
			return err
		}

		err = closeFn(g, nil)
		if err != nil {
			return err
		}

		topic = strings.TrimSpace(topic)
		if topic == "" {
			return nil
		}

		go func() {
			if err := a.openTopic(g, topic); err != nil {
				logrus.WithError(err).Error("can't open topic")
				return
			}
		}()

		return nil
	}

	if err := g.SetKeybinding("jump.input", gocui.KeyArrowDown, gocui.ModNone, nextTopicFn); err != nil {
		logrus.Error("can't assign key")
		return err
	}
	if err := g.SetKeybinding("jump.input", gocui.KeyArrowUp, gocui.ModNone, prevTopicFn); err != nil {
		logrus.Error("can't assign key")
		return err
	}
	if err := g.SetKeybinding("jump.input", gocui.KeyEnter, gocui.ModNone, enterFn); err != nil {
		logrus.Error("can't assign key")
		return err
	}

	if err := c.SetCursor(0, 1); err != nil {
		return err
	}

	g.Execute(func(g *gocui.Gui) error {
		fmt.Fprintf(c, "\n\n") // skip input view

		for _, d := range discussions {
			fmt.Fprintf(c, "%s\n", d.Name)
		}

		return nil
	})

	if _, err := g.SetCurrentView("jump.input"); err != nil {
		return err
	}

	go func() {
		oldQuery := ""

		for _ = range ticker.C {
			query := strings.TrimSpace(i.ViewBuffer())
			if query == oldQuery {
				continue
			}

			matches := []conslack.Discussion{}
			for _, d := range discussions {
				if strings.HasPrefix(d.Name, query) || strings.HasPrefix(d.Name, "#"+query) {
					matches = append(matches, d)
				}
			}

			g.Execute(func(g *gocui.Gui) error {
				c.Clear()

				fmt.Fprintf(c, "\n\n") // TODO an abstraction is fucking required for dealing with rendering

				for _, d := range matches {
					fmt.Fprintf(c, "%s\n", d.Name)
				}

				return nil
			})

			oldQuery = query
		}
	}()

	return nil
}

func formatMessage(m *conslack.Message) string {
	return fmt.Sprintf("%s %s %s\n", m.Date, m.From, m.Text)
}

func (a *App) openTopic(g *gocui.Gui, topicName string) error {
	logrus.WithField("topic", topicName).Info("opening topic")
	if a.currentTopic.close != nil {
		logrus.Info("close current topic")
		a.currentTopic.close()
	}

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
		h.Clear()
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
		v.Highlight = true

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
