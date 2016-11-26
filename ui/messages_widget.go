package ui

import "github.com/jroimartin/gocui"

// MessagesWidget implements the list of messages that had been posted in a given room.
type MessagesWidget struct {
	name string
	r    *room
	Position
}

// NewMessagesWidget creates a new messages widget for room `r`, positioned at `position`
func NewMessagesWidget(name string, r *room, position Position) *MessagesWidget {
	return &MessagesWidget{
		name:     name,
		r:        r,
		Position: position,
	}
}

// Layout ...
func (w *MessagesWidget) Layout(g *gocui.Gui) error {
	if v, err := g.SetView(w.name, w.Position.Xa, w.Position.Ya, w.Position.Xb, w.Position.Yb); err != nil {
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
