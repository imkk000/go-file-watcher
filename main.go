package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/fsnotify/fsnotify"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly})

	if len(os.Args) <= 1 {
		err := errors.New("no command to run")
		log.Fatal().Err(err).Msg("start watcher")
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("new watcher")
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Fatal().Err(err).Msg("close watcher")
		}
	}()
	if err := watcher.Add("."); err != nil {
		log.Fatal().Err(err).Msg("add watcher")
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	go func() {
		for range timer.C {
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Process = nil
			}
			fmt.Println("start:", cmd)
			if err := cmd.Start(); err != nil {
				fmt.Println("start:", err)
			}
		}
	}()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Debug().
				Str("event", event.Op.String()).
				Str("name", event.Name).
				Bool("reset", timer.Reset(debounce)).
				Msg("has event")
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Err(err).Msg("error")
		}
	}
}

const debounce = 500 * time.Millisecond
