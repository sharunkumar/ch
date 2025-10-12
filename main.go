package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset = "\033[0m"
)

// Preset color palette (bright, readable colors)
var presetColors = []string{
	"\033[38;2;255;105;97m",  // Pastel Red
	"\033[38;2;134;194;29m",  // Pastel Green
	"\033[38;2;240;160;75m",  // Pastel Orange
	"\033[38;2;134;176;189m", // Pastel Blue
	"\033[38;2;255;164;164m", // Pastel Pink
	"\033[38;2;203;166;247m", // Pastel Purple
}

type wordConfig struct {
	original string
	search   string // lowercase version for case-insensitive search
	color    string
}

func hexToANSI(hex string) string {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	if len(hex) != 6 {
		return ""
	}

	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func parseArgs(args []string, caseSensitive bool) []wordConfig {
	var configs []wordConfig
	colorIndex := 0

	for _, arg := range args {
		parts := strings.Split(arg, "::")
		word := parts[0]

		var color string
		if len(parts) == 2 && parts[1] != "" {
			// Custom color specified
			color = hexToANSI(parts[1])
			if color == "" {
				fmt.Fprintf(os.Stderr, "Warning: invalid color '%s' for word '%s', using preset\n", parts[1], word)
				color = presetColors[colorIndex%len(presetColors)]
				colorIndex++
			}
		} else {
			// Use preset color
			color = presetColors[colorIndex%len(presetColors)]
			colorIndex++
		}

		search := word
		if !caseSensitive {
			search = strings.ToLower(word)
		}

		configs = append(configs, wordConfig{
			original: word,
			search:   search,
			color:    color,
		})
	}

	return configs
}

func highlightLine(line string, configs []wordConfig, caseSensitive, wholeWord bool) string {
	if len(configs) == 0 {
		return line
	}

	searchLine := line
	if !caseSensitive {
		searchLine = strings.ToLower(line)
	}

	// Track which positions are already colored (to handle overlapping matches)
	colored := make([]bool, len(line))

	// Store replacements as [start, end, replacement]
	type replacement struct {
		start int
		end   int
		text  string
	}
	var replacements []replacement

	// Find all matches
	for _, cfg := range configs {
		pos := 0
		for {
			idx := strings.Index(searchLine[pos:], cfg.search)
			if idx == -1 {
				break
			}
			idx += pos

			endIdx := idx + len(cfg.search)

			// If wholeWord mode, extend to next space or end of line
			if wholeWord {
				for endIdx < len(line) && line[endIdx] != ' ' && line[endIdx] != '\n' && line[endIdx] != '\t' {
					endIdx++
				}
			}

			// Check if this position is already colored (overlapping match)
			alreadyColored := false
			for i := idx; i < endIdx; i++ {
				if colored[i] {
					alreadyColored = true
					break
				}
			}

			if !alreadyColored {
				// Mark as colored
				for i := idx; i < endIdx; i++ {
					colored[i] = true
				}

				// Store replacement
				matchedText := line[idx:endIdx]
				coloredText := cfg.color + matchedText + Reset
				replacements = append(replacements, replacement{
					start: idx,
					end:   endIdx,
					text:  coloredText,
				})
			}

			pos = idx + 1
		}
	}

	// If no matches, return original line
	if len(replacements) == 0 {
		return line
	}

	// Sort replacements by start position (they should already be mostly sorted)
	// Build result string
	var result strings.Builder
	lastPos := 0

	// Sort replacements by start position
	for i := 0; i < len(replacements); i++ {
		for j := i + 1; j < len(replacements); j++ {
			if replacements[j].start < replacements[i].start {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}

	for _, r := range replacements {
		result.WriteString(line[lastPos:r.start])
		result.WriteString(r.text)
		lastPos = r.end
	}
	result.WriteString(line[lastPos:])

	return result.String()
}

func main() {
	caseSensitive := flag.Bool("s", false, "case-sensitive matching")
	wholeWord := flag.Bool("w", false, "extend match to whole word (until space or EOL)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ch [options] <word1> <word2>::<HEXCOLOR> ...\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -s    case-sensitive matching (default: case-insensitive)\n")
		fmt.Fprintf(os.Stderr, "  -w    extend match to whole word\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  tail -f app.log | ch error warning::FF5500 success::00FF00\n")
		os.Exit(1)
	}

	configs := parseArgs(args, *caseSensitive)

	// Read from stdin line by line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		highlighted := highlightLine(line, configs, *caseSensitive, *wholeWord)
		fmt.Println(highlighted)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
