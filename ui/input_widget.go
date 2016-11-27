package ui

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/jroimartin/gocui"
)

// InputWidget implements the list of messages that had been posted in a given room.
type InputWidget struct {
	name           string
	discussionName string
	postMessageFn  func(message string) error
	Position
}

const defaultInputWidgetHeight = 2

// NewInputWidget creates a new messages widget for room `r`, positioned at `position`
func NewInputWidget(name string, postMessageFn func(string) error, e executeFn, position Position) *InputWidget {
	return &InputWidget{
		name:          name,
		postMessageFn: postMessageFn,
		Position:      position,
	}
}

// SetPostMessageFn ...
// This is questionnable, something is missing to link the whole thing together
func (w *InputWidget) SetPostMessageFn(fn func(msg string) error) {
	w.postMessageFn = fn
}

// Layout ...
func (w *InputWidget) Layout(g *gocui.Gui) error {
	if v, err := g.SetView(w.name, w.Position.Xa, w.Position.Ya, w.Position.Xb, w.Position.Yb); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Autoscroll = false
		v.Editable = true
		v.Wrap = true
		v.Frame = false
		v.Overwrite = true

		fn := func(g *gocui.Gui, v *gocui.View) error {
			message := v.ViewBuffer()
			message = strings.TrimSpace(message)

			err := w.postMessageFn(message)
			if err != nil {
				logrus.WithError(err).Debugf("failed to post message")
				return err
			}

			v.Clear()
			v.SetCursor(0, 0)

			return nil
		}

		if err := g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, fn); err != nil {
			return err
		}

		_, err := g.SetCurrentView("input")
		if err != nil {
			return err
		}

		_, err = g.SetViewOnTop("input")
		if err != nil {
			return err
		}
	}

	return nil
}

// Height returns vertical size of the input
func (w *InputWidget) Height() int {
	return 2
}
