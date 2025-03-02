# AudioCondenser

This is a simple cli util to produce audio files from audio/video with pauses in dialog removed using information from corresponding subtitle files.

## Usage

Provide input file (audio or video), subtitle (.srt or .ass) and maximum pause length. As the output program will produce condensed mp3 file.

Instead of individual files, a directory for both audio/video and subtitle paths could be provided as parameters. This will result in processing of all available file pairs. Audio/video and subtitle files must have corresponding order in their respective directories.

For working with audio/video this program uses ffmpeg which must be available in $PATH (e.g. installed with package manager).

By default this program does not process files and instead only prints expected result (i.e. which audio/video and subtitle pairs will be processed and how much they will be reduced in duration). To actually run condensing through ffmpeg supply additional --run flag.

Use can use --help to get list of available flags and their usage.
