package main

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
	"sync"
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
		wg      *sync.WaitGroup
		//ctx     context.Context
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
	//ctx = context.Background()
	//ctx, cancelFunc := context.WithTimeout(ctx, time.Second*10)
	//defer cancelFunc()
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
}

func addLine(ch chan string, line string, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	ch <- line
	return nil
}
