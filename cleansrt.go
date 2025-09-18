package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "clean_srt",
		Usage: "Converts .srt subtitles into human-readable text",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
		},
		ArgsUsage: "<input.srt>",
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Please provide an input .srt file", 1)
			}

			inputPath := c.Args().Get(0)
			outputPath := c.String("output")

			f, err := os.Open(inputPath)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to open file: %v", err), 1)
			}
			defer f.Close()

			lines := readLines(f)
			cleaned := cleanSRT(lines)

			if outputPath != "" {
				err := os.WriteFile(outputPath, []byte(cleaned), 0644)
				if err != nil {
					return cli.Exit(fmt.Sprintf("Failed to write file: %v", err), 1)
				}
				fmt.Printf("Transcript written to: %s\n", outputPath)
			} else {
				fmt.Println(cleaned)
			}
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
	var paragraphs []string
	var currentParagraph []string
	lastLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if len(currentParagraph) > 0 {
				paragraphs = append(paragraphs, strings.Join(currentParagraph, " "))
				currentParagraph = nil
			}
			continue
		}

		if timestampRe.MatchString(trimmed) || isNumber(trimmed) {
			continue
		}

		if strings.HasPrefix(trimmed, ">>") {
			trimmed = strings.TrimPrefix(trimmed, ">>")
		}

		trimmed = strings.TrimSpace(trimmed)

		if trimmed != "" && trimmed != lastLine {
			currentParagraph = append(currentParagraph, trimmed)
			lastLine = trimmed
		}
	}

	if len(currentParagraph) > 0 {
		paragraphs = append(paragraphs, strings.Join(currentParagraph, " "))
	}

	return strings.Join(paragraphs, "\n\n")
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
