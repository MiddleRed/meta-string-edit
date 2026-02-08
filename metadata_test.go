package editor

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testDataDir = "./test"
	sample1     = "global-metadata.dat.sample1"
	sample2     = "global-metadata.dat.sample2"
)

// TestLoadMetadata_Sample1 tests loading the first sample file
func TestLoadMetadata_Sample1(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load sample1: %v", err)
	}

	if mf.GetVersion() == 0 {
		t.Error("Version should not be 0")
	}

	if mf.GetStringCount() == 0 {
		t.Error("String count should not be 0")
	}

	t.Logf("Sample1: Version=%d, StringCount=%d, FileSize=%d",
		mf.GetVersion(), mf.GetStringCount(), mf.GetFileSize())
}

// TestLoadMetadata_Sample2 tests loading the second sample file
func TestLoadMetadata_Sample2(t *testing.T) {
	path := filepath.Join(testDataDir, sample2)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load sample2: %v", err)
	}

	if mf.GetVersion() == 0 {
		t.Error("Version should not be 0")
	}

	if mf.GetStringCount() == 0 {
		t.Error("String count should not be 0")
	}

	t.Logf("Sample2: Version=%d, StringCount=%d, FileSize=%d",
		mf.GetVersion(), mf.GetStringCount(), mf.GetFileSize())
}

// TestLoadMetadata_InvalidFile tests loading an invalid file
func TestLoadMetadata_InvalidFile(t *testing.T) {
	_, err := LoadMetadata("nonexistent.dat")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

// TestGetVersion tests version retrieval
func TestGetVersion(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	version := mf.GetVersion()
	if version <= 0 {
		t.Errorf("Invalid version: %d", version)
	}
}

// TestGetStringCount tests string count retrieval
func TestGetStringCount(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	count := mf.GetStringCount()
	if count <= 0 {
		t.Errorf("Invalid string count: %d", count)
	}
}

// TestListStrings tests listing all strings
func TestListStrings(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	strings := mf.ListStrings()
	if len(strings) != mf.GetStringCount() {
		t.Errorf("ListStrings returned %d strings, expected %d", len(strings), mf.GetStringCount())
	}

	// Verify indices are correct
	for i, str := range strings {
		if str.Nth != i {
			t.Errorf("String at position %d has incorrect Nth value: %d", i, str.Nth)
		}
	}

	// Log first few strings for debugging
	t.Logf("First 5 strings:")
	for i := 0; i < 5 && i < len(strings); i++ {
		t.Logf("  [%d] %q", strings[i].Nth, strings[i].Value)
	}
}

// TestGetString tests getting a single string
func TestGetString(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Test valid index
	str, err := mf.GetString(0)
	if err != nil {
		t.Errorf("Failed to get string at index 0: %v", err)
	}
	if str.Nth != 0 {
		t.Errorf("Expected Nth=0, got %d", str.Nth)
	}

	// Test invalid index
	_, err = mf.GetString(-1)
	if err == nil {
		t.Error("Expected error for negative index")
	}

	_, err = mf.GetString(mf.GetStringCount())
	if err == nil {
		t.Error("Expected error for out-of-bounds index")
	}
}

// TestSearchStrings_Normal tests normal substring search
func TestSearchStrings_Normal(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Search for empty string should return all
	results, err := mf.SearchStrings("", false)
	if err != nil {
		t.Errorf("Failed to search: %v", err)
	}
	if len(results) != mf.GetStringCount() {
		t.Errorf("Empty search should return all strings, got %d, expected %d",
			len(results), mf.GetStringCount())
	}
}

// TestSearchStrings_Regex tests regex search
func TestSearchStrings_Regex(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Test valid regex
	results, err := mf.SearchStrings(".*", true)
	if err != nil {
		t.Errorf("Failed to search with regex: %v", err)
	}
	if len(results) != mf.GetStringCount() {
		t.Errorf("Regex '.*' should match all strings")
	}

	// Test invalid regex
	_, err = mf.SearchStrings("[invalid", true)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

// TestSearchStrings_NoMatch tests search with no matches
func TestSearchStrings_NoMatch(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Search for something unlikely to exist
	results, err := mf.SearchStrings("ThisShouldDefinitelyNotExistInMetadata12345", false)
	if err != nil {
		t.Errorf("Failed to search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected no matches, got %d", len(results))
	}
}

// TestModifyString_SameLength tests modifying a string to the same length
func TestModifyString_SameLength(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	original, _ := mf.GetString(0)
	newValue := "test"
	if len(original.Value) < 4 {
		// Find a string that's at least 4 chars
		for i := 0; i < mf.GetStringCount(); i++ {
			str, _ := mf.GetString(i)
			if len(str.Value) >= 4 {
				original = str
				break
			}
		}
	}

	// Modify to same length
	newValue = ""
	for i := 0; i < len(original.Value); i++ {
		newValue += "x"
	}

	err = mf.ModifyString(original.Nth, newValue)
	if err != nil {
		t.Errorf("Failed to modify string: %v", err)
	}

	modified, _ := mf.GetString(original.Nth)
	if modified.Value != newValue {
		t.Errorf("String not modified correctly: got %q, expected %q", modified.Value, newValue)
	}
}

// TestModifyString_Shorter tests modifying a string to be shorter
func TestModifyString_Shorter(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Find a string longer than 5 chars
	var targetIdx int
	for i := 0; i < mf.GetStringCount(); i++ {
		str, _ := mf.GetString(i)
		if len(str.Value) > 5 {
			targetIdx = i
			break
		}
	}

	newValue := "short"
	err = mf.ModifyString(targetIdx, newValue)
	if err != nil {
		t.Errorf("Failed to modify string: %v", err)
	}

	modified, _ := mf.GetString(targetIdx)
	if modified.Value != newValue {
		t.Errorf("String not modified correctly: got %q, expected %q", modified.Value, newValue)
	}
}

// TestModifyString_Longer tests modifying a string to be longer
func TestModifyString_Longer(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	newValue := "This is a much longer string than before"
	err = mf.ModifyString(0, newValue)
	if err != nil {
		t.Errorf("Failed to modify string: %v", err)
	}

	modified, _ := mf.GetString(0)
	if modified.Value != newValue {
		t.Errorf("String not modified correctly: got %q, expected %q", modified.Value, newValue)
	}
}

// TestModifyString_InvalidIndex tests modifying with invalid index
func TestModifyString_InvalidIndex(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	err = mf.ModifyString(-1, "test")
	if err == nil {
		t.Error("Expected error for negative index")
	}

	err = mf.ModifyString(mf.GetStringCount(), "test")
	if err == nil {
		t.Error("Expected error for out-of-bounds index")
	}
}

// TestModifyString_Unicode tests modifying with Unicode characters
func TestModifyString_Unicode(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	newValue := "Hello 世界 🌍 Привет"
	err = mf.ModifyString(0, newValue)
	if err != nil {
		t.Errorf("Failed to modify string with Unicode: %v", err)
	}

	modified, _ := mf.GetString(0)
	if modified.Value != newValue {
		t.Errorf("Unicode string not preserved: got %q, expected %q", modified.Value, newValue)
	}
}

// TestSave_NoModifications tests saving without modifications
func TestSave_NoModifications(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test_save.dat")
	err = mf.Save(tmpFile)
	if err != nil {
		t.Errorf("Failed to save: %v", err)
	}

	// Reload and verify
	mf2, err := LoadMetadata(tmpFile)
	if err != nil {
		t.Errorf("Failed to reload saved file: %v", err)
	}

	if mf2.GetStringCount() != mf.GetStringCount() {
		t.Errorf("String count mismatch after save")
	}
}

// TestSave_WithModifications tests saving with modifications
func TestSave_WithModifications(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Modify a string
	newValue := "Modified String Value"
	err = mf.ModifyString(0, newValue)
	if err != nil {
		t.Fatalf("Failed to modify string: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test_modified.dat")
	err = mf.Save(tmpFile)
	if err != nil {
		t.Errorf("Failed to save: %v", err)
	}

	// Reload and verify
	mf2, err := LoadMetadata(tmpFile)
	if err != nil {
		t.Errorf("Failed to reload saved file: %v", err)
	}

	modified, _ := mf2.GetString(0)
	if modified.Value != newValue {
		t.Errorf("Modification not persisted: got %q, expected %q", modified.Value, newValue)
	}
}

// TestSave_MultipleModifications tests saving with multiple modifications
func TestSave_MultipleModifications(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Modify multiple strings
	modifications := map[int]string{
		0: "First modification",
		1: "Second modification",
		2: "Third modification",
	}

	for idx, val := range modifications {
		if idx < mf.GetStringCount() {
			err = mf.ModifyString(idx, val)
			if err != nil {
				t.Fatalf("Failed to modify string %d: %v", idx, err)
			}
		}
	}

	tmpFile := filepath.Join(t.TempDir(), "test_multi.dat")
	err = mf.Save(tmpFile)
	if err != nil {
		t.Errorf("Failed to save: %v", err)
	}

	// Reload and verify
	mf2, err := LoadMetadata(tmpFile)
	if err != nil {
		t.Errorf("Failed to reload saved file: %v", err)
	}

	for idx, expectedVal := range modifications {
		if idx < mf2.GetStringCount() {
			str, _ := mf2.GetString(idx)
			if str.Value != expectedVal {
				t.Errorf("String %d not modified correctly: got %q, expected %q",
					idx, str.Value, expectedVal)
			}
		}
	}
}

// TestExportJSON tests JSON export
func TestExportJSON(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "export.json")
	err = mf.ExportJSON(tmpFile)
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	// Verify file exists and has content
	stat, err := os.Stat(tmpFile)
	if err != nil {
		t.Errorf("JSON file not created: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("JSON file is empty")
	}
}

// TestRoundTrip tests load -> modify -> save -> reload cycle
func TestRoundTrip(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	originalCount := mf.GetStringCount()

	// Make various modifications
	testData := []struct {
		idx   int
		value string
	}{
		{0, "Short"},
		{1, "A much longer string than the original one was"},
		{2, "Unicode: 你好世界 🎉"},
	}

	for _, td := range testData {
		if td.idx < originalCount {
			if err := mf.ModifyString(td.idx, td.value); err != nil {
				t.Fatalf("Failed to modify string %d: %v", td.idx, err)
			}
		}
	}

	// Save
	tmpFile := filepath.Join(t.TempDir(), "roundtrip.dat")
	if err := mf.Save(tmpFile); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Reload
	mf2, err := LoadMetadata(tmpFile)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Verify count unchanged
	if mf2.GetStringCount() != originalCount {
		t.Errorf("String count changed: got %d, expected %d", mf2.GetStringCount(), originalCount)
	}

	// Verify modifications
	for _, td := range testData {
		if td.idx < originalCount {
			str, err := mf2.GetString(td.idx)
			if err != nil {
				t.Errorf("Failed to get string %d: %v", td.idx, err)
			}
			if str.Value != td.value {
				t.Errorf("String %d mismatch: got %q, expected %q", td.idx, str.Value, td.value)
			}
		}
	}
}

// TestAlignment tests that saved files maintain 4-byte alignment
func TestAlignment(t *testing.T) {
	path := filepath.Join(testDataDir, sample1)
	mf, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Modify to create unaligned data (3 bytes)
	if mf.GetStringCount() > 0 {
		err = mf.ModifyString(0, "ABC") // 3 bytes, should trigger padding
		if err != nil {
			t.Fatalf("Failed to modify: %v", err)
		}
	}

	tmpFile := filepath.Join(t.TempDir(), "alignment.dat")
	err = mf.Save(tmpFile)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Reload and check that it succeeds (verifies alignment is correct)
	mf2, err := LoadMetadata(tmpFile)
	if err != nil {
		t.Errorf("Failed to reload aligned file: %v", err)
	}

	// Verify the data section size is aligned
	if mf2.stringLiteralDataCount%AlignmentBoundary != 0 {
		t.Errorf("Data section not aligned: size=%d, should be multiple of %d",
			mf2.stringLiteralDataCount, AlignmentBoundary)
	}
}
