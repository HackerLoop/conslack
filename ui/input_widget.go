package ui

import "github.com/jroimartin/gocui"

// InputWidget implements the list of messages that had been posted in a given room.
type InputWidget struct {
	name string
	Position
}

const defaultInputWidgetHeight = 2

// NewInputWidget creates a new messages widget for room `r`, positioned at `position`
func NewInputWidget(name string, position Position) *InputWidget {
	return &InputWidget{
		name:     name,
		Position: position,
	}
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
