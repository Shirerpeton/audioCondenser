package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Interval struct {
	start time.Duration
	end time.Duration
}

func getDurationFromTimestamp(timestamp string) (time.Duration, error) {
	parts := strings.Split(timestamp, ":")
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		err = fmt.Errorf("Can't convert timestamp to duration, error in hours %v:", err)
		return 0, err
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		err = fmt.Errorf("Can't convert timestamp to duration, error in minutes %v:", err)
		return 0, err
	}
	var sep string
	if strings.Contains(parts[2], ".") {
		sep = "."
	} else if strings.Contains(parts[2], ",") {
		sep = ","
	}
	parts = strings.Split(parts[2], sep)
	seconds, err := strconv.Atoi(parts[0])
	if err != nil {
		err = fmt.Errorf("Can't convert timestamp to duration, error in seconds %v:", err)
		return 0, err
	}
	if len(parts[1]) == 2 {
		parts[1] += "0"
	}
	miliseconds, err := strconv.Atoi(parts[1])
	if err != nil {
		err = fmt.Errorf("Can't convert timestamp to duration, error in miliseconds%v:", err)
		return 0, err
	}
	totalMiliseconds := (hours*3600 + minutes*60 + seconds) * 1000 + miliseconds
	return time.Duration(float64(totalMiliseconds) * float64(time.Millisecond)), nil
}

func parseSub(path string) ([]Interval, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	contentStr := string(content)
	if strings.HasSuffix(path, ".ass") {
		return parseAssSub(contentStr)
	} else {
		return parseSrtSub(contentStr)
	}
}

func parseSrtSub(content string) ([]Interval, error) {
	intervals := make([]Interval, 0)
	for line := range strings.SplitSeq(content, "\n") {
		if !strings.Contains(line, "-->") {
			continue;
		}
		parts := strings.Split(line, " ")
		start, err := getDurationFromTimestamp(parts[0])
		if err != nil {
			return nil, err
		}
		end, err := getDurationFromTimestamp(parts[2])
		if err != nil {
			return nil, err
		}
		intervals = append(intervals, Interval{ start, end })
	}
	return intervals, nil
}

func parseAssSub(content string) ([]Interval, error) {
	intervals := make([]Interval, 0)
	for line := range strings.SplitSeq(content, "\n") {
		if !strings.HasPrefix(line, "Dialogue: ") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 10 {
			return nil, errors.New("Malformed subtitle file")
		}
		start, err := getDurationFromTimestamp(parts[1])
		if err != nil {
			return nil, err
		}
		end, err := getDurationFromTimestamp(parts[2])
		if err != nil {
			return nil, err
		}
		intervals = append(intervals, Interval{ start, end })
	}
	return intervals, nil
}

func getResultIntervals(dialogs []Interval, maxGap time.Duration, totalDuration time.Duration) ([]*Interval, error) {
	intervals := make([]*Interval, 1)
	intervals[0] = &Interval{
		start: dialogs[0].start,
		end: dialogs[0].end,
	}
	in := intervals[0]
	if in.start > 0 {
		in.start = time.Duration(math.Max(0, float64(in.start - maxGap)))
	}
	for i := 1; i < len(dialogs); i++ {
		currDialog := dialogs[i]
		if currDialog.start - in.end <= maxGap {
			in.end = currDialog.end
		} else {
			in.end += maxGap / 2
			in = &Interval{
				start: currDialog.start - (maxGap / 2),
				end: currDialog.end,
			}
			intervals = append(intervals, in)
		}
	}
	if lastIn := intervals[len(intervals) - 1]; lastIn.end < totalDuration {
		lastIn.end = time.Duration(math.Min(float64(totalDuration), float64(lastIn.end + maxGap)))
	}
	return intervals, nil
}

func getOriginalLength(path string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}
	outputStr := strings.TrimSpace(string(output))
	seconds, err := strconv.ParseFloat(outputStr, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func printStats(path string, intervals []*Interval, originalDuration time.Duration) error {
	var newDuratinon time.Duration
	for _, interval := range intervals {
		newDuratinon += interval.end - interval.start
	}

	percent := (float64(newDuratinon) / float64(originalDuration)) * 100

	fmt.Printf("file: %s\n", path)
	fmt.Printf("original duration: %v\n", originalDuration)
	fmt.Printf("condensed duration: %v, (%.1f%%)\n", newDuratinon, percent)

	return nil
}

func main() {
	audioPath := flag.String("audio", "", "Path to input mp3 file")
	subPath := flag.String("sub", "", "Path to input .ass sub file")
	maxGap := flag.Float64("gap", 1.0, "Maximum allowed gap in dialog (in seconds)")
	output := flag.String("out", "output.mp3", "Path to output mp3 file")
	check := flag.Bool("check", false, "Calculate and print how condensed audio will be with parameters provided, but do not process")
	flag.Parse()

	if *audioPath == "" || *subPath == "" {
		fmt.Println("Provide audio and subtitle file paths")
		os.Exit(1)
	}

	if *maxGap < 0 {
		fmt.Println("Max gap must be > 0")
		os.Exit(1)
	}

	dialogs, err := parseSub(*subPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(dialogs) == 0 {
		fmt.Println("No dialogs found in sub file")
		os.Exit(1)
	}

	originalDuration, err := getOriginalLength(*audioPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	resultIntervals, err := getResultIntervals(
		dialogs,
		time.Duration(*maxGap*float64(time.Second)),
		originalDuration)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	printStats(*audioPath, resultIntervals, originalDuration)

	if *check {
		return
	}

	var filterParts []string
	if len(resultIntervals) == 1 {
		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"[0:a]atrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[out]",
				resultIntervals[0].start.Seconds(),
				resultIntervals[0].end.Seconds()))
	} else {
		for idx, trim := range resultIntervals {
			part := fmt.Sprintf(
				"[0:a]atrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[s%d]",
				trim.start.Seconds(),
				trim.end.Seconds(),
				idx)
			filterParts = append(filterParts, part)
		}
		var inputs string
		for i := range len(resultIntervals) {
			inputs += fmt.Sprintf("[s%d]", i)
		}
		concat := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[out]", inputs, len(resultIntervals))
		filterParts = append(filterParts, concat)
	}
	filterComplex := strings.Join(filterParts, ";")

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", *audioPath,
		"-filter_complex", filterComplex,
		"-map", "[out]",
		*output)
	outputDir := filepath.Dir(*output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running ffmpeg: %v\n", err)
		os.Exit(1)
	}
}
