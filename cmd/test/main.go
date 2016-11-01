package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"

	"github.com/HackerLoop/conslack"
)

func main() {
	c, err := conslack.New(os.Getenv("SLACK_TOKEN"))
	if err != nil {
		logrus.WithError(err).Fatal("unable to connect to slack")
	}

	ds, err := c.GetDiscussions()
	if err != nil {
		logrus.WithError(err).Fatal("unable to get discussions")
	}

	for _, d := range ds {
		fmt.Printf("%#v\n", d)
	}
}
