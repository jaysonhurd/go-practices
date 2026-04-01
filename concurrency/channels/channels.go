package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
	"time"
)

func main() {
	var (
		l       zerolog.Logger
		file    *os.File
		scanner *bufio.Scanner
		ch      chan string
		count   int
		err     error
		ctx     context.Context
	)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	l = zerolog.New(os.Stderr).With().Timestamp().Logger()

	file, err = os.Open("fixtures/small-examplefile-com.txt")
	if err != nil {
		l.Fatal().Err(err).Msg("failed to open file")
	}

	scanner = bufio.NewScanner(file)
	count = 0
	for scanner.Scan() {
		count++
	}
	fmt.Printf("File has %d lines\n", count)
	file.Seek(0, io.SeekStart)

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
				return
			}
			fmt.Println(fmt.Sprintf("%d line: %s", count, line))
			time.Sleep(time.Second * 2)
			count--
			if count == 0 {
				close(ch)
			}
		case <-ctx.Done():
			l.Fatal().Err(ctx.Err()).Msg("context canceled")
		}

	}

}

func addLine(ch chan string, line string) (err error) {
	ch <- line
	return nil
}
