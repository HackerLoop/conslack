package ui

import (
	"fmt"

	"github.com/jroimartin/gocui"

	"github.com/hackerloop/conslack"
)

// MessagesWidget implements the list of messages that had been posted in a given discussion.
type MessagesWidget struct {
	name string
	e    executeFn
	Position
	lastMessage conslack.Message
}

// NewMessagesWidget creates a new messages widget for discussion `r`, positioned at `position`
func NewMessagesWidget(name string, e executeFn, position Position) *MessagesWidget {
	return &MessagesWidget{
		name:     name,
		Position: position,
		e:        e,
	}
}

// Layout ...
func (w *MessagesWidget) Layout(g *gocui.Gui) error {
	v, err := g.SetView(w.name, w.Position.Xa, w.Position.Ya, w.Position.Xb, w.Position.Yb)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Loading..."
		v.Editable = false
		v.Wrap = true
		v.Frame = false
		v.Autoscroll = true
		v.Overwrite = true
	}

	return nil
}

// AppendMessage ...
func (w *MessagesWidget) AppendMessage(m conslack.Message) {
	w.e(func(g *gocui.Gui) error {
		v, err := g.View(w.name)
		if err != nil {
			return err
		}

		fmt.Fprintf(v, formatMessage(&m))
		return nil
	})
}

// SetMessages ...
func (w *MessagesWidget) SetMessages(ms []conslack.Message) {
	w.e(func(g *gocui.Gui) error {
		v, err := g.View(w.name)
		if err != nil {
			return err
		}

		v.Clear()

		for _, m := range ms {
			fmt.Fprintf(v, formatMessage(&m))
		}
		return nil
	})
}

func formatMessage(m *conslack.Message) string {
	return fmt.Sprintf("%s %s %s\n", m.Date, m.From, m.Text)
}
