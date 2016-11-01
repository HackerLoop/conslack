package conslack

import (
	"fmt"

	"github.com/nlopes/slack"
)

// Client represents the client interacting with Slack.
type Client struct {
	api                  *slack.Client
	cache                map[string]string
	currentUserID        string
	channelSubscriptions map[string]chan Message
}

// Message represents a message coming from the Slack API.
type Message struct {
	Channel string
	From    string
	Date    string // TODO: This should probably be a time.Time.
	Text    string
}

// Discussion represents a channel or a user
type Discussion struct {
	ID   string
	Name string
}

// New returns a new Client configured with the proper token.
// This function tries to authenticate on Slack. If an error occurs, the error
// is returned to the user.
// A websocket connection is also opened(RTM) to handle updates from Slack.
// There is currently no way to close it.
func New(token string) (*Client, error) {
	c := &Client{
		api:                  slack.New(token),
		cache:                make(map[string]string),
		channelSubscriptions: make(map[string]chan Message),
	}

	// Trying to authenticate on Slack
	auth, err := c.api.AuthTest()
	if err != nil {
		return nil, err
	}

	// storing the user id on the client. Maybe useful at some point?
	c.currentUserID = auth.UserID

	// We are getting all the users to store them on an in memory cache.
	users, err := c.api.GetUsers()
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		c.cache[u.ID] = fmt.Sprintf("@%v", u.Name)
	}

	// We are getting all the channels to store them on an in memory cache.
	channels, err := c.api.GetChannels(true)
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		c.cache[ch.ID] = fmt.Sprintf("#%v", ch.Name)
	}

	// Starting the websocket handling process in a Goroutine.
	go c.processEvents()

	return c, nil
}

// History returns the discussion history(100 latest messages) for the given channelName.
// channelName can be a #channel or a @user.
func (c *Client) History(channelName string) ([]Message, error) {
	// We must convert the channel name to the channel ID.
	id, err := c.cacheGetUserIDFromName(channelName)
	if err != nil {
		return nil, err
	}

	params := slack.HistoryParameters{
		Count:     100,
		Inclusive: false,
		Unreads:   false,
	}

	var h *slack.History

	switch id[0] {
	case 'C':
		// If the ID belongs to a channel
		h, err = c.api.GetChannelHistory(id, params)
		if err != nil {
			return nil, err
		}
	case 'U':
		// if the ID belongs to a user, we are getting the IM channel(this could also be cached at some point)
		_, _, channelID, err := c.api.OpenIMChannel(id)
		if err != nil {
			return nil, err
		}
		h, err = c.api.GetIMHistory(channelID, params)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("channel type not handled")
	}

	var messages []Message
	for _, m := range h.Messages {
		messages = append([]Message{Message{
			From: c.cache[m.User],
			Date: formatTimestamp(m.Timestamp),
			Text: m.Text, //TODO: should parse text and replace <@U0549KTN8|arkan> with @arkan for example
		}}, messages...)
	}

	return messages, nil
}

// Messages is used to retrieve in realtime messages from a given channelName.
// This method returns 3 parameters:
// - A read-only chan of Message that can be read to get any new message in realtime.
// - A close function that must be closed once you no longer to to get updates. It will close the channel and cleanup event routing locally.
// - And a potential error
func (c *Client) Messages(channelName string) (<-chan Message, func(), error) {
	var channelID string
	if channelName[0] == '#' {
		var err error

		channelID, err = c.cacheGetUserIDFromName(channelName)
		if err != nil {
			return nil, nil, err
		}
	} else if channelName[0] == '@' {
		id, err := c.cacheGetUserIDFromName(channelName)
		if err != nil {
			return nil, nil, err
		}

		_, _, channelID, err = c.api.OpenIMChannel(id)
		if err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, fmt.Errorf("unknown id")
	}

	// TODO: this not not safe - this should be protected by a mutex
	ch, ok := c.channelSubscriptions[channelID]
	if !ok {
		ch = make(chan Message)
		c.channelSubscriptions[channelID] = ch
	}

	close := func() {
		ch, ok := c.channelSubscriptions[channelID]
		if ok {
			close(ch)
			delete(c.channelSubscriptions, channelID)
		}
	}

	return ch, close, nil
}

// PostMessage posts a message to channelName.
// channelName can be a #channel or a @user.
func (c *Client) PostMessage(channelName string, message string) error {
	// https://godoc.org/github.com/nlopes/slack#PostMessageParameters
	postParams := slack.PostMessageParameters{
		AsUser: true,
	}

	id, err := c.cacheGetUserIDFromName(channelName)
	if err != nil {
		return err
	}

	// https://godoc.org/github.com/nlopes/slack#Client.PostMessage
	_, _, err = c.api.PostMessage(id, message, postParams)
	return err
}

// GetDiscussions returns all the discussions the user can talk to.
func (c *Client) GetDiscussions() ([]Discussion, error) {
	var discussions []Discussion

	channels, err := c.api.GetChannels(false)
	if err != nil {
		return nil, err
	}

	for _, c := range channels {
		discussions = append(discussions, Discussion{
			ID:   c.ID,
			Name: fmt.Sprintf("#%v", c.Name),
		})
	}
	// TODO: add users too

	return discussions, nil
}

// processEvents processes events from Slack in realtime with the websocket(RTM)
func (c *Client) processEvents() {
	rtm := c.api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
			case *slack.MessageEvent:
				//TODO: handle update/delete/etc... here
				//TODO: add an ID to the message as well - could be a slack.NewRefMsg
				c.dispatchMessage(Message{
					Channel: ev.Channel,
					From:    c.cache[ev.User],
					Text:    ev.Text,
					Date:    formatTimestamp(ev.Timestamp),
				})
			case *slack.PresenceChangeEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				return
			default:
			}
		}
	}
}

// dispatchMessage dispatches the message to the subscriber.
func (c *Client) dispatchMessage(m Message) {
	ch, ok := c.channelSubscriptions[m.Channel]
	if !ok {
		return
	}

	ch <- m
}

// TODO: this must be rewritten
func (c *Client) cacheGetUserIDFromName(n string) (string, error) {
	for k, v := range c.cache {
		if v == n {
			return k, nil
		}
	}

	return "", fmt.Errorf("channel not found in cache")
}
