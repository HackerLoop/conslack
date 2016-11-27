package ui

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/jroimartin/gocui"
)

// HeaderWidget implements the top bar just above a message view.
// It's used to display informations related to what that message view displays,
// such as topic if it's a channel that is displayed, nickname if it's a PM, etc ...
type HeaderWidget struct {
	name  string
	value string
	Position
}

// NewHeaderWidget creates a new header named `name` positioned at `position`
func NewHeaderWidget(name string, value string, e executeFn, position Position) *HeaderWidget {
	return &HeaderWidget{
		name:     name,
		value:    value,
		Position: position,
	}
}

// Layout defines how that header is being drawn
func (h *HeaderWidget) Layout(g *gocui.Gui) error {
	if v, err := g.SetView(h.name, h.Position.Xa, h.Position.Ya, h.Position.Xb, h.Position.Yb); err != nil {
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
			fmt.Fprintf(v, h.value)
			return nil
		})
	}

	logrus.WithField("name", h.name).Debugf("Layout")

	return nil
}

// Value returns the string being displayed in that header
func (h *HeaderWidget) Value() string {
	return h.value
}

// SetValue sets the string being displayed in that header
func (h *HeaderWidget) SetValue(value string) {
	h.value = value
}
