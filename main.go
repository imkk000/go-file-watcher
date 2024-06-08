package main

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
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

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	sig := make(chan os.Signal, 1)
	defer close(sig)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case _, ok := <-timer.C:
			if !ok {
				return
			}
			kill(cmd)
			if err := cmd.Start(); err != nil {
				log.Err(err).Msg("start")
			}
			log.Debug().
				Int("pid", cmd.Process.Pid).
				Msg("start")
		case s, ok := <-sig:
			if !ok {
				return
			}
			log.Debug().
				Int("pid", cmd.Process.Pid).
				Str("signal", s.String()).
				Msg("signal")
			kill(cmd)
			return
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

func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		cmd.Process.Wait()
		log.Debug().
			Int("pid", cmd.Process.Pid).
			Msg("killed")
		cmd.Process = nil
	}
}
