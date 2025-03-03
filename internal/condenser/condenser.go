package condenser

import (
	"strings"
	"fmt"
	"path/filepath"
	"os"
	"os/exec"

	"github.com/shirerpeton/audioCondenser/internal/common"
)

func ProcessFile(file common.CondenseFile, track int) error {
	var filterParts []string
	inputStream := fmt.Sprintf("[0:a:%d]", track)
	if len(file.CondenseIntervals) == 1 {
		filterParts = append(
			filterParts,
			fmt.Sprintf(
				"%satrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[out]",
				inputStream,
				file.CondenseIntervals[0].Start.Seconds(),
				file.CondenseIntervals[0].End.Seconds()))
	} else {
		for idx, trim := range file.CondenseIntervals {
			part := fmt.Sprintf(
				"%satrim=start=%.3f:end=%.3f,asetpts=PTS-STARTPTS[s%d]",
				inputStream,
				trim.Start.Seconds(),
				trim.End.Seconds(),
				idx)
			filterParts = append(filterParts, part)
		}
		var inputs string
		for i := range len(file.CondenseIntervals) {
			inputs += fmt.Sprintf("[s%d]", i)
		}
		concat := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[out]", inputs, len(file.CondenseIntervals))
		filterParts = append(filterParts, concat)
	}
	filterComplex := strings.Join(filterParts, ";")

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", file.Input,
		"-filter_complex", filterComplex,
		"-map", "[out]",
		file.Output)
	outputDir := filepath.Dir(file.Output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
