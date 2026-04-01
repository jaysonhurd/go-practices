package main

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
	"sync"
)

func main() {
	var (
		l       zerolog.Logger
		file    *os.File
		scanner *bufio.Scanner
		wg      sync.WaitGroup
		count   int
		err     error
	)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	l = zerolog.New(os.Stderr).With().Timestamp().Logger()

	file, err = os.Open("fixtures/1mb-examplefile-com.txt")
	if err != nil {
		l.Fatal().Err(err).Msg("failed to open file")
	}

	scanner = bufio.NewScanner(file)
	count = 0
	for scanner.Scan() {
		wg.Add(1)
		count++
	}
	fmt.Printf("File has %d lines\n", count)

	file.Seek(0, io.SeekStart)

	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		go func(wg *sync.WaitGroup, text string) {
			printLines(wg, text)
		}(&wg, text)
	}

	wg.Wait()
	fmt.Println("Done")

}

func printLines(wg *sync.WaitGroup, line string) (err error) {
	defer wg.Done()
	fmt.Println(line)
	return nil
}
