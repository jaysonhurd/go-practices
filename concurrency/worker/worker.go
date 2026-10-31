package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

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
		ch      chan string
		count   int
		err     error
		wg      *sync.WaitGroup
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
	wg = new(sync.WaitGroup)
	wg.Add(count)

	// Fill the buffered channel
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		go func(ch chan string, text string) {
			addLine(ch, text, wg)
		}(ch, text)
	}
	wg.Wait()
	close(ch)

	time.Sleep(time.Second * 1)
	workerPool := make(chan struct{}, 3)

	for req := range ch {
		workerPool <- struct{}{}
		go func(c chan string) {
			defer func() { <-workerPool }()
			output := fmt.Sprintf("%s  %d", req, len(ch))
			fmt.Println(output)
		}(ch)
	}

	fmt.Println("done")
	return nil
}

// addLine sends one line to the buffered channel.
func addLine(ch chan string, line string, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	ch <- line
	return nil
}
