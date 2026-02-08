package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	metadata "github.com/MiddleRed/meta-string-edit"
	"github.com/spf13/cobra"
)

func safePrintable(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		default:
			if unicode.IsPrint(r) {
				b.WriteRune(r)
			} else {
				if r <= 0xFFFF {
					b.WriteString(fmt.Sprintf("\\u%04X", r))
				} else {
					b.WriteString(fmt.Sprintf("\\U%08X", r))
				}
			}
		}
	}
	return b.String()
}

var (
	// Flags
	infoFlag   bool
	listFlag   string
	searchFlag string
	regexFlag  string
	dumpFlag   bool
	editFlags  []string
	outputFlag string
)

func CLIMain() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "metastringedit <file_path>",
	Short: "Unity IL2CPP global-metadata.dat string editor",
	Long: `A tool for viewing and editing string literals in Unity IL2CPP global-metadata.dat files.

Examples:
  # Show file info
  metastringedit global-metadata.dat
  metastringedit global-metadata.dat -i

  # List strings (page 0, 20 items)
  metastringedit global-metadata.dat -l 0:20

  # List page 2 (items 40-60)
  metastringedit global-metadata.dat -l 2:20

  # List single string at index 12
  metastringedit global-metadata.dat -l 12

  # List strings from index 1 to 20
  metastringedit global-metadata.dat -l 1-20

  # Search for strings
  metastringedit global-metadata.dat -s "Player"

  # Regex search
  metastringedit global-metadata.dat -r "^Player[0-9]+"

  # Dump to JSON
  metastringedit global-metadata.dat -d
  metastringedit global-metadata.dat -d -o output.json

  # Edit strings
  metastringedit global-metadata.dat -e "0=NewValue"
  metastringedit global-metadata.dat -e "0=Value1" -e "1=Value2" -o modified.dat
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCommand,
}

func init() {
	rootCmd.Flags().BoolVarP(&infoFlag, "info", "i", false, "Show metadata file information (default)")
	rootCmd.Flags().StringVarP(&listFlag, "list", "l", "", "List strings:\n- [page:num] for pagination, e.g. 0:20\n- [nth] for single string, e.g. 3\n- [start-end] for range, e.g. 1-20")
	rootCmd.Flags().StringVarP(&searchFlag, "search", "s", "", "Search for strings (case-insensitive substring)")
	rootCmd.Flags().StringVarP(&regexFlag, "regex", "r", "", "Search for strings using regex")
	rootCmd.Flags().BoolVarP(&dumpFlag, "dump", "d", false, "Dump all strings to JSON")
	rootCmd.Flags().StringArrayVarP(&editFlags, "edit", "e", []string{}, "Edit strings (format: nth=value, can input multiple pairs)")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path")
}

func runCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cmd.Usage()
		return nil
	}

	// load file
	filePath := args[0]

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	mf, err := metadata.LoadMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %v", err)
	}

	// action dispatch
	switch {
	case listFlag != "":
		return listStrings(mf, listFlag)
	case searchFlag != "":
		return searchStrings(mf, searchFlag, false)
	case regexFlag != "":
		return searchStrings(mf, regexFlag, true)
	case dumpFlag:
		return dumpJSON(mf)
	case len(editFlags) > 0:
		return editStrings(mf, filePath, editFlags)
	default:
		return showInfo(mf, filePath)
	}
}

func showInfo(mf *metadata.MetadataFile, filePath string) error {
	fmt.Println("=== Metadata File Information ===")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("Metadata Version: %d\n", mf.GetVersion())
	fmt.Printf("File Size: %d bytes (%.2f MB)\n", mf.GetFileSize(), float64(mf.GetFileSize())/1024/1024)
	fmt.Printf("String Count: %d\n", mf.GetStringCount())
	return nil
}

func listStrings(mf *metadata.MetadataFile, spec string) error {
	allStrings := mf.ListStrings()
	totalCount := len(allStrings)

	if strings.Contains(spec, ":") {
		// page:num mode
		parts := strings.Split(spec, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid page format, expected `page:num` (e.g., `0:20`)")
		}

		page, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid page number: %v", err)
		}

		pageSize, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid page size: %v", err)
		}

		if page < 0 || pageSize <= 0 {
			return fmt.Errorf("page must be >= 0 and page size must be > 0")
		}

		start := page * pageSize
		end := start + pageSize

		if start >= totalCount {
			return fmt.Errorf("page %d is out of range (total pages: %d)", page, (totalCount+pageSize-1)/pageSize)
		}

		if end > totalCount {
			end = totalCount
		}

		fmt.Printf("=== Page %d (showing %d-%d of %d) ===\n", page, start, end-1, totalCount)
		for i := start; i < end; i++ {
			str := allStrings[i]
			value := str.Value
			value = safePrintable(value)
			fmt.Printf("[%d] %s\n", str.Nth, value)
		}

		totalPages := (totalCount + pageSize - 1) / pageSize
		fmt.Printf("\nTotal Pages: %d | Current Page: %d | Page Size: %d\n", totalPages, page, pageSize)

	} else if strings.Contains(spec, "-") {
		// range mode
		parts := strings.Split(spec, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid range format, expected `start-end` (e.g., `1-20`)")
		}

		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid start index: %v", err)
		}

		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid end index: %v", err)
		}

		if start < 0 || end < start {
			return fmt.Errorf("invalid range: start must be >= 0 and end >= start")
		}

		if start >= totalCount {
			return fmt.Errorf("start index %d is out of range [0, %d)", start, totalCount)
		}

		if end >= totalCount {
			end = totalCount - 1
		}

		fmt.Printf("=== Range %d-%d (showing %d-%d of %d) ===\n", start, end, start, end, totalCount)
		for i := start; i <= end; i++ {
			str := allStrings[i]
			value := str.Value
			value = safePrintable(value)
			fmt.Printf("[%d] %s\n", str.Nth, value)
		}

	} else {
		// nth mode
		nth, err := strconv.Atoi(spec)
		if err != nil {
			return fmt.Errorf("invalid index format: expected number, `page:num`, or `start-end`")
		}

		if nth < 0 || nth >= totalCount {
			return fmt.Errorf("index %d is out of range [0, %d)", nth, totalCount)
		}

		str := allStrings[nth]
		value := str.Value
		value = safePrintable(value)
		fmt.Printf("[%d] %s\n", str.Nth, value)
	}

	return nil
}

func searchStrings(mf *metadata.MetadataFile, pattern string, useRegex bool) error {
	results, err := mf.SearchStrings(pattern, useRegex)
	if err != nil {
		return fmt.Errorf("search failed: %v", err)
	}

	searchType := "substring"
	if useRegex {
		searchType = "regex"
	}

	fmt.Printf("=== Search Results (%s: %q) ===\n", searchType, pattern)
	fmt.Printf("Found %d match(es):\n\n", len(results))

	for _, str := range results {
		value := str.Value
		if len(value) > 100 {
			value = value[:97] + "..."
		}
		value = safePrintable(value)
		fmt.Printf("[%d] %s\n", str.Nth, value)
	}

	return nil
}

func dumpJSON(mf *metadata.MetadataFile) error {
	// Determine output path
	outputPath := outputFlag
	if outputPath == "" {
		outputPath = "stringliteral.json"
	}

	// If output is a directory, use default filename
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		outputPath = filepath.Join(outputPath, "stringliteral.json")
	}

	if err := mf.ExportJSON(outputPath); err != nil {
		return fmt.Errorf("failed to export JSON: %v", err)
	}

	fmt.Printf("Successfully exported %d strings to: %s\n", mf.GetStringCount(), outputPath)
	return nil
}

func editStrings(mf *metadata.MetadataFile, inputPath string, edits []string) error {
	// Parse and apply all edits
	modifications := make(map[int]string)

	for _, edit := range edits {
		parts := strings.SplitN(edit, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid edit format: %q (expected 'nth=value')", edit)
		}

		nth, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid index in edit %q: %v", edit, err)
		}

		modifications[nth] = parts[1]
	}

	// Validate all indices first
	for nth := range modifications {
		if nth < 0 || nth >= mf.GetStringCount() {
			return fmt.Errorf("index %d is out of range [0, %d)", nth, mf.GetStringCount())
		}
	}

	// Apply modifications
	fmt.Printf("Applying %d modification(s)...\n", len(modifications))
	for nth, value := range modifications {
		if err := mf.ModifyString(nth, value); err != nil {
			return fmt.Errorf("failed to modify string %d: %v", nth, err)
		}
		fmt.Printf("  [%d] -> %q\n", nth, value)
	}

	// Determine output path
	outputPath := outputFlag
	if outputPath == "" {
		dir := filepath.Dir(inputPath)
		base := filepath.Base(inputPath)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		outputPath = filepath.Join(dir, name+".edited"+ext)
	}

	// If output is a directory, generate filename
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		base := filepath.Base(inputPath)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		outputPath = filepath.Join(outputPath, name+".edited"+ext)
	}

	// Save
	if err := mf.Save(outputPath); err != nil {
		return fmt.Errorf("failed to save: %v", err)
	}

	fmt.Printf("\nSuccessfully saved modified metadata to: %s\n", outputPath)
	return nil
}
