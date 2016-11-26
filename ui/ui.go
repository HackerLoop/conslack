package ui

import (
	"github.com/Sirupsen/logrus"
	"github.com/hackerloop/conslack"
	"github.com/jroimartin/gocui"
)

// a room is either a channel, a private channel or a person
type room struct {
	id       string
	title    string
	messages []conslack.Message
}

type state struct {
	currentRoomID string
	rooms         map[string]*room
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

// NewApp returns a new Application connected to a conslack Client
func NewApp(c *conslack.Client) (*App, error) {
	g, err := gocui.NewGui()
	if err != nil {
		return nil, err
	}

	a := App{
		c: c,
		g: g,
		s: &state{},
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

	a.headerWidget = NewHeaderWidget("header", "Conslack: connecting ...", Position{-1, -1, maxX, 1})
	a.statusBarWidget = NewHeaderWidget("status", "StatusBar", Position{-1, maxY - defaultInputWidgetHeight - 3, maxX, maxY - defaultInputWidgetHeight})
	a.inputWidget = NewInputWidget("input", Position{-1, maxY - defaultInputWidgetHeight - 2, maxX, maxY})
	a.messagesWidget = NewMessagesWidget("messages", nil, Position{-1, 0, maxX, maxY - defaultInputWidgetHeight - 1})

	return nil
}

func (a *App) assignGlobalKeyBindings() error {
	if err := a.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	return nil
}

// Loop starts the event loop and will block until
// either an error is returned or that the ui has been
// instructed to exit.
func (a *App) Loop() error {
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
