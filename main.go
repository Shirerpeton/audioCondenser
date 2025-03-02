package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/fatih/color"

	"github.com/shirerpeton/audioCondenser/internal/common"
	"github.com/shirerpeton/audioCondenser/internal/condenser"
	"github.com/shirerpeton/audioCondenser/internal/parser"
)

func printStats(files []*common.CondenseFile) error {
	for _, file := range files {
		percent := (float64(file.CondensedDuration) / float64(file.OriginalDuration)) * 100
		color.Set(color.FgYellow)
		fmt.Print("input: ")
		color.Set(color.FgGreen)
		fmt.Printf("%s\n", file.Input)
		color.Set(color.FgYellow)
		fmt.Print("sub: ")
		color.Set(color.FgGreen)
		fmt.Printf("%s\n", file.Sub)
		color.Set(color.FgYellow)
		fmt.Print("duration: ")
		color.Set(color.FgGreen)
		fmt.Printf("%v\n", file.OriginalDuration)
		color.Set(color.FgYellow)
		fmt.Print("output: ")
		color.Set(color.FgMagenta)
		fmt.Printf("%s\n", file.Output)
		color.Set(color.FgYellow)
		fmt.Print("condensed duration: ")
		color.Set(color.FgMagenta)
		fmt.Printf("%v (%.1f%%)\n", file.CondensedDuration, percent)
		color.Unset()
		fmt.Println()
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

func getFiles(input string, sub string, output string, isDir bool) ([]*common.CondenseFile, error) {
	files := make([]*common.CondenseFile, 0)

	if !isDir {
		file := &common.CondenseFile{
			Input: input,
			Sub: sub,
		}
		if output != "" {
			file.Output = output
		} else {
			file.Output = getOutputPath(file.Input)
		}
		files = append(files, file)
	} else {
		outputFolder := output
		if outputFolder == "" {
			outputFolder = "./output/"
		}
		err := os.Mkdir(outputFolder, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if !strings.HasSuffix(outputFolder, "/") {
			outputFolder += "/"
		}
		var inputEntries []string
		var subsEntries []string
		var outputEntries []string
		entries, err := os.ReadDir(input)
		if err != nil {
			return nil, err
		}
		for _, entr := range entries {
			if entr.IsDir() {
				continue
			}
			path := input + entr.Name()
			inputEntries = append(inputEntries, path)
			outputEntries = append(outputEntries, outputFolder + getOutputPath(entr.Name()))
		}
		entries, err = os.ReadDir(sub)
		if err != nil {
			return nil, err
		}
		for _, entr := range entries {
			if entr.IsDir() {
				continue
			}
			path := sub + entr.Name()
			subsEntries = append(subsEntries, path)
		}
		if len(inputEntries) == 0 {
			return nil, errors.New("no input files")
		}
		if len(subsEntries) == 0 {
			return nil, errors.New("no input subtitles")
		}
		for i := 0; i < len(inputEntries) && i < len(subsEntries); i++ {
			file := &common.CondenseFile{
				Input: inputEntries[i],
				Sub: subsEntries[i],
				Output: outputEntries[i],
			}
			files = append(files, file)
		}
	}
	return files, nil
}

func main() {
	input := flag.String("input", "", "Path to input audio/video file or directory containing them")
	sub := flag.String("sub", "", "Path to input subtitle file or directory containing them")
	maxGap := flag.Float64("gap", 1.0, "Maximum allowed gap in dialog (in seconds, decimal)")
	output := flag.String("out", "", "Path to output mp3 file, defaults to input filename with _condensed suffix and mp3 extension, for diretory processing must be a directory name as well")
	run := flag.Bool("run", false, "Supply to run ffmpeg commands to condense files, in absense of this flag, command will just calculate and print stats")
	flag.Parse()

	if *input == "" || *sub == "" {
		fmt.Println("Provide input audio or video and subtitle file paths")
		os.Exit(1)
	}

	if *maxGap < 0 {
		fmt.Println("Max gap must be > 0")
		os.Exit(1)
	}

	inputStat, err := os.Stat(*input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	subStat, err := os.Stat(*sub)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if (inputStat.IsDir() && !subStat.IsDir()) || (!inputStat.IsDir() && subStat.IsDir()) {
		fmt.Println("Either both input and sub parameters should be files or directories")
		os.Exit(1)
	}

	files, err := getFiles(*input, *sub, *output, inputStat.IsDir())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var g errgroup.Group

	for _, file := range files {
		g.Go(func() error {
			err = parser.Parse(file, *maxGap)
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("Error parsing files", err)
		os.Exit(1)
	}

	printStats(files)

	if !*run {
		return
	}


	for _, file := range files {
		g.Go(func() error {
			err = condenser.ProcessFile(*file)
			if err != nil {
				return err
			}
			fmt.Print("File ");
			color.Set(color.FgMagenta)
			fmt.Printf("%s", file.Output);
			color.Unset()
			fmt.Println(" - done");
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("Error processing files", err)
		os.Exit(1)
	}
}
