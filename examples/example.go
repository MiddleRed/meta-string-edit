// try this script by running `go run ./examples` from the project root

package main

import (
	"fmt"
	"log"

	metadata "github.com/MiddleRed/meta-string-edit"
)

const metadataFilePath = "./test/global-metadata.dat.sample1"

func main() {
	fmt.Println("=== Example 1: Load and Inspect ===")
	loadAndInspect()

	fmt.Println("\n=== Example 2: Search Strings ===")
	searchStrings()

	fmt.Println("\n=== Example 3: Modify Strings ===")
	modifyStrings()

	fmt.Println("\n=== Example 4: Export to JSON ===")
	exportToJSON()
}

// Example 1: Load and inspect a metadata file
func loadAndInspect() {
	// Load the metadata file
	mf, err := metadata.LoadMetadata(metadataFilePath)
	if err != nil {
		log.Fatalf("Failed to load metadata: %v", err)
	}

	// Display basic information
	fmt.Printf("Metadata Version: %d\n", mf.GetVersion())
	fmt.Printf("File Size: %d bytes (%.2f MB)\n", mf.GetFileSize(), float64(mf.GetFileSize())/1024/1024)
	fmt.Printf("String Count: %d\n", mf.GetStringCount())

	// Get and display first few strings
	fmt.Println("\nFirst 5 strings:")
	strings := mf.ListStrings()
	for i := 0; i < 5 && i < len(strings); i++ {
		str := strings[i]
		value := str.Value
		if len(value) > 60 {
			value = value[:57] + "..."
		}
		fmt.Printf("  [%d] %s\n", str.Nth, value)
	}
}

// Example 2: Search for strings
func searchStrings() {
	mf, err := metadata.LoadMetadata(metadataFilePath)
	if err != nil {
		log.Fatalf("Failed to load metadata: %v", err)
	}

	// Normal search (case-insensitive substring)
	fmt.Println("Searching for 'element'...")
	results, err := mf.SearchStrings("element", false)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("Found %d matches (showing first 5):\n", len(results))
	for i := 0; i < 5 && i < len(results); i++ {
		str := results[i]
		value := str.Value
		if len(value) > 60 {
			value = value[:57] + "..."
		}
		fmt.Printf("  [%d] %s\n", str.Nth, value)
	}

	// Regex search
	fmt.Println("\nSearching with regex '^[A-Z][a-z]+Type$'...")
	results, err = mf.SearchStrings("^[A-Z][a-z]+Type$", true)
	if err != nil {
		log.Fatalf("Regex search failed: %v", err)
	}
	fmt.Printf("Found %d matches:\n", len(results))
	for _, str := range results {
		fmt.Printf("  [%d] %s\n", str.Nth, str.Value)
	}
}

// Example 3: Modify strings and save
func modifyStrings() {
	// Load the metadata file
	mf, err := metadata.LoadMetadata(metadataFilePath)
	if err != nil {
		log.Fatalf("Failed to load metadata: %v", err)
	}

	// Get original value
	original, _ := mf.GetString(0)
	fmt.Printf("Original string [0]: %q\n", original.Value)

	// Modify the string
	newValue := "MODIFIED_BY_GO_EXAMPLE"
	if err := mf.ModifyString(0, newValue); err != nil {
		log.Fatalf("Failed to modify string: %v", err)
	}
	fmt.Printf("Modified string [0]: %q\n", newValue)

	// Modify multiple strings
	modifications := map[int]string{
		1: "Second_Modified",
		2: "Third_Modified",
	}
	for idx, val := range modifications {
		if err := mf.ModifyString(idx, val); err != nil {
			log.Fatalf("Failed to modify string %d: %v", idx, err)
		}
		fmt.Printf("Modified string [%d]: %q\n", idx, val)
	}

	// Save to a new file
	outputPath := "./modified_example.dat"
	if err := mf.Save(outputPath); err != nil {
		log.Fatalf("Failed to save: %v", err)
	}
	fmt.Printf("\nSaved modified metadata to: %s\n", outputPath)

	// Verify the changes
	mf2, err := metadata.LoadMetadata(outputPath)
	if err != nil {
		log.Fatalf("Failed to reload: %v", err)
	}
	modified, _ := mf2.GetString(0)
	fmt.Printf("Verified string [0] in saved file: %q\n", modified.Value)
}

// Example 4: Export to JSON
func exportToJSON() {
	mf, err := metadata.LoadMetadata(metadataFilePath)
	if err != nil {
		log.Fatalf("Failed to load metadata: %v", err)
	}

	// Export all strings to JSON
	outputPath := "./strings.json"
	if err := mf.ExportJSON(outputPath); err != nil {
		log.Fatalf("Failed to export JSON: %v", err)
	}

	fmt.Printf("Exported %d strings to: %s\n", mf.GetStringCount(), outputPath)
	fmt.Println("You can now view or process the JSON file with other tools.")
}
