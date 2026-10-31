package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/jaysonhurd/go-practices/internal/applog"
)

const (
	logSetupFailed = "initialize application log"
	logOpenFailed  = "open fixture"
	logScanFailed  = "scan fixture"
	logSeekFailed  = "rewind fixture"
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
		wg      sync.WaitGroup
		count   int
		err     error
	)

	l, logFile, err := applog.NewLogger("logs/go-practices.log")
	if err != nil {
		slog.Error(logSetupFailed, slog.Any("error", err))
		return err
	}
	defer logFile.Close()

	file, err = os.Open("fixtures/1mb-examplefile-com.txt")
	if err != nil {
		l.Error(logOpenFailed, slog.Any("error", err))
		return err
	}

	defer file.Close()

	scanner = bufio.NewScanner(file)
	count = 0
	for scanner.Scan() {
		wg.Add(1)
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

	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		go func(wg *sync.WaitGroup, text string) {
			printLines(wg, text)
		}(&wg, text)
	}

	wg.Wait()
	fmt.Println("Done")

	return nil
}

// printLines prints one line and marks the task complete.
func printLines(wg *sync.WaitGroup, line string) (err error) {
	defer wg.Done()
	fmt.Println(line)
	return nil
}
