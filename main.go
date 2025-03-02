package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/shirerpeton/audioCondenser/internal/common"
	"github.com/shirerpeton/audioCondenser/internal/condenser"
	"github.com/shirerpeton/audioCondenser/internal/parser"
)

func printStats(files []common.CondenseFile) error {
	for _, file := range files {
		percent := (float64(file.CondensedDuration) / float64(file.OriginalDuration)) * 100
		fmt.Printf(
			"input: %s | sub: %s | duration %v -> output: %s | condensed duration: %v (%.1f%%)\n",
			file.Input,
			file.Sub,
			file.OriginalDuration,
			file.Output,
			file.CondensedDuration,
			percent)
	}
	return nil
}

func getOutputPath(input string) string {
	var result string
	extensionIdx := strings.LastIndex(input, ".")
	if extensionIdx != -1 {
		result = input[:extensionIdx]
	}
	result += "_condensed.mp3"
	return result
}

func main() {
	input := flag.String("input", "", "Path to input audio or video file")
	sub := flag.String("sub", "", "Path to input .ass sub file")
	maxGap := flag.Float64("gap", 1.0, "Maximum allowed gap in dialog (in seconds)")
	output := flag.String("out", "", "Path to output mp3 file, defaults to input filename with _condensed suffix and mp3 extension")
	check := flag.Bool("check", false, "Calculate and print how condensed audio will be with parameters provided, but do not process")
	flag.Parse()

	if *input == "" || *sub == "" {
		fmt.Println("Provide input audio or video and subtitle file paths")
		os.Exit(1)
	}

	if *maxGap < 0 {
		fmt.Println("Max gap must be > 0")
		os.Exit(1)
	}
	file := &common.CondenseFile{
		Input: *input,
		Sub: *sub,
	}
	if *output != "" {
		file.Output = *output
	} else {
		file.Output = getOutputPath(file.Input)
	}
	file, err := parser.Parse(file, *maxGap);
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	files := make([]common.CondenseFile, 1)
	files[0] = *file

	printStats(files)

	if *check {
		return
	}

	err = condenser.ProcessFile(*file);
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
