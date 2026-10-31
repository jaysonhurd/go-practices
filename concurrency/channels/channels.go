package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jaysonhurd/go-practices/internal/applog"
)

const (
	logSetupFailed     = "initialize application log"
	logOpenFailed      = "open fixture"
	logScanFailed      = "scan fixture"
	logSeekFailed      = "rewind fixture"
	logContextCanceled = "context canceled"
)

// main runs the concurrency example from the repository root.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run owns the log and fixture resources for the concurrency demonstration.
func run() error {
	var (
		l       *slog.Logger
		file    *os.File
		scanner *bufio.Scanner
		ch      chan string
		count   int
		err     error
		ctx     context.Context
	)
	l, logFile, err := applog.NewLogger("logs/go-practices.log")
	if err != nil {
		slog.Error(logSetupFailed, slog.Any("error", err))
		return err
	}
	defer logFile.Close()

	file, err = os.Open("fixtures/small-examplefile-com.txt")
	if err != nil {
		l.Error(logOpenFailed, slog.Any("error", err))
		return err
	}

	defer file.Close()

	scanner = bufio.NewScanner(file)
	count = 0
	for scanner.Scan() {
		count++
	}
	fmt.Printf("File has %d lines\n", count)
	if err := scanner.Err(); err != nil {
		l.Error(logScanFailed, slog.Any("error", err))
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		l.Error(logSeekFailed, slog.Any("error", err))
		return err
	}

	ch = make(chan string, count)

	ctx = context.Background()
	ctx, cancelFunc := context.WithTimeout(ctx, time.Second*10)
	defer cancelFunc()

	// Fill the buffered channel
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		go func(ch chan string, text string) {
			addLine(ch, text)
		}(ch, text)
	}

	// empty the buffered channel
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				fmt.Println("Channel closed")
				return nil
			}
			fmt.Printf("%d line: %s\n", count, line)
			time.Sleep(time.Second * 2)
			count--
			if count == 0 {
				close(ch)
			}
		case <-ctx.Done():
			l.Error(logContextCanceled, slog.Any("error", ctx.Err()))
			return ctx.Err()
		}

	}

}

// addLine sends one line to the buffered channel.
func addLine(ch chan string, line string) (err error) {
	ch <- line
	return nil
}
