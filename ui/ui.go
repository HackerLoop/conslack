package ui

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jroimartin/gocui"

	"github.com/hackerloop/conslack"
)

// a discussion is either a channel, a private channel or a person
type discussion struct {
	name     string
	title    string
	messages []conslack.Message
	close    func()
	sync.Mutex
}

type state struct {
	currentDiscussion string
	discussions       map[string]*discussion
	sync.Mutex
}

// App abstracts a ui application.
// It handles updates from a conslack.Client, reflecting them into state
// and finally display them in a gocui.Gui.
type App struct {
	c *conslack.Client
	g *gocui.Gui
	s *state

	// widgets
	headerWidget    *HeaderWidget
	statusBarWidget *HeaderWidget
	messagesWidget  *MessagesWidget
	inputWidget     *InputWidget
}

type executeFn func(f func(*gocui.Gui) error)
type handlerFn func(f func(*gocui.Gui, *gocui.View) error)

// NewApp returns a new Application connected to a conslack Client
func NewApp(c *conslack.Client) (*App, error) {
	g, err := gocui.NewGui()
	if err != nil {
		return nil, err
	}

	a := App{
		c: c,
		g: g,
		s: &state{
			discussions: make(map[string]*discussion),
		},
	}

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorBlack
	g.SelBgColor = gocui.ColorWhite
	g.Mouse = false
	g.InputEsc = true

	if err := a.createWidgets(); err != nil {
		return nil, err
	}

	g.SetManager(a.headerWidget, a.messagesWidget, a.statusBarWidget, a.inputWidget)

	if err := a.assignGlobalKeyBindings(); err != nil {
		return nil, err
	}

	return &a, nil
}

func (a *App) createWidgets() error {
	maxX, maxY := a.g.Size()

	a.headerWidget = NewHeaderWidget(
		"header",
		"Conslack: connecting ...",
		a.g.Execute,
		Position{
			-1,
			-1,
			maxX,
			1,
		})

	a.statusBarWidget = NewHeaderWidget(
		"status",
		"StatusBar",
		a.g.Execute,
		Position{
			-1,
			maxY - defaultInputWidgetHeight - 3,
			maxX,
			maxY - defaultInputWidgetHeight,
		})

	a.inputWidget = NewInputWidget(
		"input",
		nil,
		a.g.Execute,
		Position{
			-1,
			maxY - defaultInputWidgetHeight - 2,
			maxX,
			maxY,
		})

	a.messagesWidget = NewMessagesWidget(
		"messages",
		a.g.Execute,
		Position{
			-1,
			0,
			maxX,
			maxY - defaultInputWidgetHeight - 1,
		})

	return nil
}

func (a *App) assignGlobalKeyBindings() error {
	if err := a.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	return nil
}

func (a *App) jumpToDiscussion(name string) {
	a.s.currentDiscussion = name

	a.inputWidget.SetPostMessageFn(func(message string) error {
		return a.c.PostMessage(name, message)
	})

	if _, ok := a.s.discussions[name]; !ok {
		a.subscribeDiscussion(name)
	}
}

func (a *App) subscribeDiscussion(name string) {
	if _, ok := a.s.discussions[name]; ok {
		return
	}

	// TODO rework how r is created to ensure proper sync
	r := &discussion{
		name:  name,
		title: name,
	}
	a.s.discussions[name] = r
	logrus.WithField("discussion", name).Debugf("subscribing to discussion")

	go func() {
		messages, err := a.c.History(name)
		if err != nil {
			logrus.WithError(err).Error("unable to get history")
			// an error is swallowed here
			return
		}
		logrus.WithField("discussion", name).WithField("count", len(messages)).Debugf("received history")

		r.messages = messages

		ch, close, err := a.c.Messages(name)
		if err != nil {
			logrus.WithError(err).Fatal("unable to get realtime messages")
		}
		r.close = close

		// TODO synchronize this
		if r.name == a.s.currentDiscussion {
			a.messagesWidget.SetMessages(r.messages)
		}

		for m := range ch {
			r.messages = append(r.messages, m)

			// TODO synchronize this
			if r.name == a.s.currentDiscussion {
				a.messagesWidget.AppendMessage(m)
			}
		}
	}()

}

// Loop starts the event loop and will block until
// either an error is returned or that the ui has been
// instructed to exit.
func (a *App) Loop() error {
	// FIXME
	time.Sleep(100 * time.Millisecond)
	a.jumpToDiscussion("#general")
	return a.g.MainLoop()
}

// Close ...
func (a *App) Close() {
	a.g.Close()
}

func quit(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		logrus.WithField("view", v.Name).Debugf("quitting")
	} else {
		logrus.WithField("view", nil).Debugf("quitting")
	}

	return gocui.ErrQuit
}
