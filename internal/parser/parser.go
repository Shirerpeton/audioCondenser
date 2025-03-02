package parser

import (
	"time"
	"strings"
	"fmt"
	"strconv"
	"os"
	"os/exec"
	"math"
	"errors"

	"github.com/shirerpeton/audioCondenser/internal/common"
)

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
	duration := hours*int(time.Hour) + minutes*int(time.Minute) + seconds*int(time.Second) + miliseconds*int(time.Millisecond);
	return time.Duration(duration), nil
}

func parseSub(path string) ([]common.Interval, error) {
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

func parseSrtSub(content string) ([]common.Interval, error) {
	intervals := make([]common.Interval, 0)
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
		intervals = append(intervals, common.Interval{ Start: start, End: end })
	}
	return intervals, nil
}

func parseAssSub(content string) ([]common.Interval, error) {
	intervals := make([]common.Interval, 0)
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
		intervals = append(intervals, common.Interval{ Start: start, End: end })
	}
	return intervals, nil
}

func getCondenseIntervals(dialogs []common.Interval, maxGap time.Duration, totalDuration time.Duration) ([]*common.Interval, error) {
	intervals := make([]*common.Interval, 1)
	intervals[0] = &common.Interval{
		Start: dialogs[0].Start,
		End: dialogs[0].End,
	}
	in := intervals[0]
	if in.Start > 0 {
		in.Start = time.Duration(math.Max(0, float64(in.Start - maxGap)))
	}
	for i := 1; i < len(dialogs); i++ {
		currDialog := dialogs[i]
		if currDialog.Start - in.End <= maxGap {
			in.End = currDialog.End
		} else {
			in.End += maxGap / 2
			in = &common.Interval{
				Start: currDialog.Start - (maxGap / 2),
				End: currDialog.End,
			}
			intervals = append(intervals, in)
		}
	}
	if lastIn := intervals[len(intervals) - 1]; lastIn.End < totalDuration {
		lastIn.End = time.Duration(math.Min(float64(totalDuration), float64(lastIn.End + maxGap)))
	}
	return intervals, nil
}

func getOriginalDuration(path string) (time.Duration, error) {
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

func getCondensedDuration(condensedIntervals []*common.Interval) (time.Duration, error) {
	var condensedDuration time.Duration
	for _, intrv := range condensedIntervals {
		if intrv.Start >= intrv.End {
			return 0, errors.New("Malformed dialog timings (end > start)")
		}
		condensedDuration += intrv.End - intrv.Start
	}
	return condensedDuration, nil
}

func Parse(file *common.CondenseFile, maxGap float64) error {
	dialogs, err := parseSub(file.Sub)
	if err != nil {
		return err
	}
	if len(dialogs) == 0 {
		return errors.New("No dialogs found in sub file");
	}

	originalDuration, err := getOriginalDuration(file.Input)
	if err != nil {
		return err
	}
	file.OriginalDuration = originalDuration

	condenseIntervals, err := getCondenseIntervals(
		dialogs,
		time.Duration(maxGap*float64(time.Second)),
		originalDuration)
	if err != nil {
		return err
	}
	file.CondenseIntervals = condenseIntervals

	condensedDuration, err := getCondensedDuration(file.CondenseIntervals)
	if err != nil {
		return err
	}
	file.CondensedDuration = condensedDuration

	return nil
}
