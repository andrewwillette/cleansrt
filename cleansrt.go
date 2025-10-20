package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "cleansrt",
		Usage: "Downloads and cleans YouTube auto-generated subtitles into readable text",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: transcript.txt)",
			},
		},
		ArgsUsage: "<youtube_url>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Please provide a YouTube URL", 1)
			}

			youtubeURL := c.Args().Get(0)
			outputPath := c.String("output")
			if outputPath == "" {
				outputPath = "transcript.txt"
			}

			tmpDir, err := os.MkdirTemp("", "cleansrt_*")
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to create temp dir: %v", err), 1)
			}
			defer os.RemoveAll(tmpDir)

			srtFile := filepath.Join(tmpDir, "transcript.en.srt")
			ytdlpCmd := exec.Command("yt-dlp",
				"--write-auto-sub",
				"--sub-lang", "en",
				"--skip-download",
				"--convert-subs", "srt",
				"-o", filepath.Join(tmpDir, "transcript.%(ext)s"),
				youtubeURL,
			)
			ytdlpCmd.Stdout = os.Stdout
			ytdlpCmd.Stderr = os.Stderr
			if err := ytdlpCmd.Run(); err != nil {
				return cli.Exit(fmt.Sprintf("yt-dlp failed: %v", err), 1)
			}

			f, err := os.Open(srtFile)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to open %s: %v", srtFile, err), 1)
			}
			defer f.Close()

			lines := readLines(f)
			cleaned := cleanSRT(lines)

			err = os.WriteFile(outputPath, []byte(cleaned+"\n"), 0644)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to write output: %v", err), 1)
			}

			fmt.Printf("✅ Clean transcript saved to: %s\n", outputPath)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
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

func cleanSRT(lines []string) string {
	timestampRe := regexp.MustCompile(`^\d{2}:\d{2}:\d{2},\d{3}`)
	var textBuilder strings.Builder
	lastLine := ""

	// Gather all lines into one big text block first
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip timestamps, indices, and empty lines
		if trimmed == "" || timestampRe.MatchString(trimmed) || isNumber(trimmed) {
			continue
		}

		if strings.HasPrefix(trimmed, ">>") {
			trimmed = strings.TrimPrefix(trimmed, ">>")
		}

		trimmed = strings.TrimSpace(trimmed)

		// Avoid repeating the same line twice
		if trimmed != "" && trimmed != lastLine {
			textBuilder.WriteString(trimmed + " ")
			lastLine = trimmed
		}
	}

	// Split sentences on punctuation
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

		// Add punctuation back if present
		if i < len(matches) {
			sentence += text[matches[i][0]:matches[i][1]]
		}

		result = append(result, sentence)
	}

	// Join each sentence with two newlines
	return strings.Join(result, "\n\n")
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
