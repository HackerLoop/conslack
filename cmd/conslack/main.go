package main

import (
	"log"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/hackerloop/conslack"
	"github.com/hackerloop/conslack/ui"
	"github.com/jroimartin/gocui"
)

func main() {
	c, err := conslack.New(os.Getenv("SLACK_TOKEN"))
	if err != nil {
		logrus.WithError(err).Fatal("unable to connect to slack")
	}

	app, err := ui.NewApp(c)
	if err != nil {
		log.Panicln(err)
	}
	defer app.Close()

	f, err := os.OpenFile("log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()

	logrus.SetOutput(f)

	if err := app.Loop(); err != nil {
		if err != gocui.ErrQuit {
			log.Panicln(err)
		}
	}
}
