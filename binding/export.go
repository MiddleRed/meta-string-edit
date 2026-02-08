package main

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"unsafe"

	cli "metastringedit/cli"

	editor "github.com/MiddleRed/meta-string-edit"
)

var (
	mu        sync.Mutex
	handles         = make(map[int64]*editor.MetadataFile)
	nextID    int64 = 1
	lastError string
)

func setErr(err error) {
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		lastError = err.Error()
	} else {
		lastError = ""
	}
}

func getHandle(h C.longlong) *editor.MetadataFile {
	mu.Lock()
	defer mu.Unlock()
	return handles[int64(h)]
}

func tryGetHandle(h C.longlong) (*editor.MetadataFile, error) {
	mf := getHandle(h)

	if mf == nil {
		return nil, fmt.Errorf("cannot get handle")
	}
	return mf, nil
}

// ---------- CLI ----------

//export RunAsCLI
func RunAsCLI(argc C.int, argv **C.char) {
	length := int(argc)
	tmpslice := unsafe.Slice(argv, length)

	goArgs := make([]string, length)
	for i := 0; i < length; i++ {
		goArgs[i] = C.GoString(tmpslice[i])
	}

	os.Args = goArgs
	cli.CLIMain()
}

// ---------- Lifecycle ----------

//export MetaLoadMetadata
func MetaLoadMetadata(filePath *C.char) C.longlong {
	path := C.GoString(filePath)
	mf, err := editor.LoadMetadata(path)
	if err != nil {
		setErr(err)
		return -1
	}

	mu.Lock()
	id := nextID
	nextID++
	handles[id] = mf
	mu.Unlock()

	setErr(nil)
	return C.longlong(id)
}

//export MetaFreeMetadata
func MetaFreeMetadata(handle C.longlong) {
	mu.Lock()
	delete(handles, int64(handle))
	mu.Unlock()
}

// ---------- Error ----------

//export MetaGetLastError
func MetaGetLastError() *C.char {
	mu.Lock()
	e := lastError
	mu.Unlock()
	return C.CString(e)
}

//export MetaFreeString
func MetaFreeString(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}

// ---------- Properties ----------

//export MetaGetVersion
func MetaGetVersion(handle C.longlong) C.int {
	mf, err := tryGetHandle(handle)
	if err != nil {
		setErr(err)
		return 0
	}

	return C.int(mf.GetVersion())
}

//export MetaGetStringCount
func MetaGetStringCount(handle C.longlong) C.longlong {
	mf, err := tryGetHandle(handle)
	if err != nil {
		setErr(err)
		return 0
	}
	return C.longlong(mf.GetStringCount())
}

//export MetaGetFileSize
func MetaGetFileSize(handle C.longlong) C.longlong {
	mf, err := tryGetHandle(handle)
	if err != nil {
		setErr(err)
		return 0
	}
	return C.longlong(mf.GetFileSize())
}

// ---------- String access ----------

//export MetaListStrings
func MetaListStrings(handle C.longlong) *C.char {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return nil
	}

	strs := mf.ListStrings()
	data, err := json.Marshal(strs)
	if err != nil {
		setErr(err)
		return nil
	}

	setErr(nil)
	return C.CString(string(data))
}

//export MetaGetString
func MetaGetString(handle C.longlong, nth C.longlong) *C.char {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return nil
	}

	str, err := mf.GetString(int(nth))
	if err != nil {
		setErr(err)
		return nil
	}

	data, err := json.Marshal(str)
	if err != nil {
		setErr(err)
		return nil
	}

	setErr(nil)
	return C.CString(string(data))
}

//export MetaSearchStrings
func MetaSearchStrings(handle C.longlong, pattern *C.char, useRegex C.int) *C.char {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return nil
	}

	results, err := mf.SearchStrings(C.GoString(pattern), useRegex != 0)
	if err != nil {
		setErr(err)
		return nil
	}

	if results == nil {
		results = []editor.StringLiteral{}
	}

	data, err := json.Marshal(results)
	if err != nil {
		setErr(err)
		return nil
	}

	setErr(nil)
	return C.CString(string(data))
}

// ---------- Mutations ----------

//export MetaModifyString
func MetaModifyString(handle C.longlong, nth C.longlong, newValue *C.char) C.int {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return -1
	}

	err := mf.ModifyString(int(nth), C.GoString(newValue))
	if err != nil {
		setErr(err)
		return -1
	}

	setErr(nil)
	return 0
}

//export MetaSave
func MetaSave(handle C.longlong, outputPath *C.char) C.int {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return -1
	}

	err := mf.Save(C.GoString(outputPath))
	if err != nil {
		setErr(err)
		return -1
	}

	setErr(nil)
	return 0
}

//export MetaExportJSON
func MetaExportJSON(handle C.longlong, outputPath *C.char) C.int {
	mf := getHandle(handle)
	if mf == nil {
		setErr(fmt.Errorf("invalid handle"))
		return -1
	}

	err := mf.ExportJSON(C.GoString(outputPath))
	if err != nil {
		setErr(err)
		return -1
	}

	setErr(nil)
	return 0
}

func main() {}
