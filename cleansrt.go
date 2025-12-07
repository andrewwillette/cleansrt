package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

var debugEnabled bool

func main() {
	app := &cli.App{
		Name:  "cleansrt",
		Usage: "Downloads and cleans YouTube auto-generated subtitles into readable text",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "outputdir",
				Aliases: []string{"od"},
				Usage:   "Output file path (default: current working directory)",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug logging output",
			},
		},
		ArgsUsage: "<youtube_url>",
		Action: func(c *cli.Context) error {
			debugEnabled = c.Bool("debug")
			if c.NArg() < 1 {
				return cli.Exit("Please provide a YouTube URL", 1)
			}

			youtubeURL := c.Args().Get(0)
			outputDir := c.String("outputdir")
			if outputDir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return cli.Exit(fmt.Sprintf("failed to get cwd: %v", err), 1)
				}
				outputDir = wd
			}
			debugf("Using output directory: %s", outputDir)

			titleCmd := exec.Command("yt-dlp", "--get-title", youtubeURL)
			titleBytes, err := titleCmd.Output()
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to get video title with youtubeURL: %v, %v", youtubeURL, err), 1)
			}

			title := strings.TrimSpace(string(titleBytes))
			debugf("Video title: %s", title)

			title = strings.Map(func(r rune) rune {
				switch r {
				case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
					return '-'
				default:
					return r
				}
			}, title)

			outputFile := fmt.Sprintf("%s/%s.txt", outputDir, title)
			debugf("Output file will be: %s", outputFile)

			tmpDir, err := os.MkdirTemp("", "cleansrt_*")
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to create temp dir: %v", err), 1)
			}
			defer os.RemoveAll(tmpDir)
			debugf("Temporary directory: %s", tmpDir)

			srtTmpFile := filepath.Join(tmpDir, "transcript.en.srt")
			debugf("Temporary subtitle file path: %s", srtTmpFile)

			ytdlpCmd := exec.Command("yt-dlp",
				"--write-auto-sub",
				"--sub-lang", "en",
				"--skip-download",
				"--convert-subs", "srt",
				"-o", filepath.Join(tmpDir, "transcript.%(ext)s"),
				youtubeURL,
			)

			if debugEnabled {
				ytdlpCmd.Stdout = os.Stderr // send yt-dlp output to stderr
				ytdlpCmd.Stderr = os.Stderr
			}

			debugf("Running yt-dlp command...")
			if err := ytdlpCmd.Run(); err != nil {
				return cli.Exit(fmt.Sprintf("yt-dlp failed: %v", err), 1)
			}
			debugf("yt-dlp finished successfully")

			f, err := os.Open(srtTmpFile)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to open %s: %v", srtTmpFile, err), 1)
			}
			defer f.Close()

			lines := readLines(f)
			debugf("Read %d lines from subtitle file", len(lines))
			cleaned := formatSRTFileAsHumanReadable(lines)
			debugf("Formatted transcript length: %d bytes", len(cleaned))

			err = os.WriteFile(outputFile, []byte(cleaned+"\n"), 0644)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to write output: %v", err), 1)
			}

			// Output only the file path (for scripts)
			fmt.Println(outputFile)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}

func debugf(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func readLines(r io.Reader) []string {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func formatSRTFileAsHumanReadable(lines []string) string {
	timestampRe := regexp.MustCompile(`^\d{2}:\d{2}:\d{2},\d{3}`)
	var textBuilder strings.Builder
	lastLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || timestampRe.MatchString(trimmed) || isNumber(trimmed) {
			continue
		}

		trimmed = strings.TrimSpace(trimmed)

		if trimmed != "" && trimmed != lastLine {
			textBuilder.WriteString(trimmed + " ")
			lastLine = trimmed
		}
	}

	text := textBuilder.String()
	sentenceRe := regexp.MustCompile(`([.!?])\s+`)
	sentences := sentenceRe.Split(text, -1)
	matches := sentenceRe.FindAllStringSubmatchIndex(text, -1)

	var result []string
	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		if i < len(matches) {
			sentence += text[matches[i][0]:matches[i][1]]
		}

		chunks := splitByLength(sentence, 130)
		result = append(result, chunks...)
	}

	return strings.Join(result, "\n\n")
}

func splitByLength(s string, max int) []string {
	var parts []string
	words := strings.Fields(s)
	var line strings.Builder
	for _, word := range words {
		if line.Len()+len(word)+1 > max {
			parts = append(parts, strings.TrimSpace(line.String()))
			line.Reset()
		}
		line.WriteString(word + " ")
	}
	if line.Len() > 0 {
		parts = append(parts, strings.TrimSpace(line.String()))
	}
	return parts
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
