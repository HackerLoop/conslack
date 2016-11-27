package main

import (
	"log"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/jroimartin/gocui"

	"github.com/hackerloop/conslack"
	"github.com/hackerloop/conslack/ui"
)

func main() {
	f, err := os.OpenFile("conslack.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(f)

	c, err := conslack.New(os.Getenv("SLACK_TOKEN"))
	if err != nil {
		logrus.WithError(err).Fatal("unable to connect to slack")
	}

	app, err := ui.NewApp(c)
	if err != nil {
		log.Panicln(err)
	}
	defer app.Close()

	if err := app.Loop(); err != nil {
		if err != gocui.ErrQuit {
			log.Panicln(err)
		}
	}

}
