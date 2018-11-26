package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
	"unicode/utf8"
)

var (
	initial = flag.Duration("base", 1*time.Second, "the base delay per character")
	step    = flag.Duration("step", 100*time.Millisecond, "the amount of proportial delay added per rune")
	bits    = flag.Uint("bits", 3, "the number of bits per rune used to determine an appropriate delay")
	debug   = flag.Bool("debug", false, "print the input character and the calculated delay instead of the output unmodified")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "as slow as possible\n\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	src, dst := io.Reader(os.Stdin), io.Writer(os.Stdout)
	delay := bePatient(*bits, *initial, *step)
	if *debug {
		dst = ioutil.Discard
		delay = printImpatiently(os.Stdout, delay)
	}
	copyRunesWithPatience(dst, src, delay)
}

// a func that determines how long to wait
type patience func(rune) time.Duration

// help debug patience by showing exactly how patient we're being
func printImpatiently(dst io.Writer, f patience) patience {
	return func(b rune) time.Duration {
		delay := f(b)
		fmt.Fprintf(dst, "%q %U %s\n", string(b), b, delay)
		return delay
	}
}

// be patient
func bePatient(bits uint, initial, step time.Duration) patience {
	if bits >= 8 {
		panic("too many bits")
	}
	mask := rune((0x1 << bits) - 1)

	return func(b rune) time.Duration {
		return initial + step*time.Duration(mask&b)
	}
}

// copy runes from src to dst, being patient about writing every byte.
func copyRunesWithPatience(dst io.Writer, src io.Reader, patience patience) error {
	flush := makeFlush(dst)

	scanner := bufio.NewScanner(src)
	scanner.Split(bufio.ScanRunes)

	for scanner.Scan() {
		bs := scanner.Bytes()
		for len(bs) > 0 {
			r, size := utf8.DecodeRune(bs)
			delay := patience(r)

			if _, err := dst.Write(bs[:size]); err != nil {
				return err
			}

			flush()
			time.Sleep(delay)

			bs = bs[size:]
		}
	}
	return scanner.Err()
}

// returns a func that flushes this writer. if the writer is unflushable,
// returns a noop.
//
// this is here just in case you'd like to aslap an io.Writer that isn't
// an os.File
func makeFlush(w io.Writer) func() {
	// http.Flusher
	type flusher interface{ Flush() }
	// bufio.Writer
	type safeFlusher interface{ Flush() error }
	// os.File
	type syncer interface{ Sync() error }

	switch t := w.(type) {
	case flusher:
		return t.Flush
	case safeFlusher:
		return func() { t.Flush() }
	case syncer:
		return func() { t.Sync() }
	default:
		return func() {}
	}
}
