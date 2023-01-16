// Utility to convert files into a Go byte arrays

// Based on http://github.com/cratonica/2goarray by Clint Caywood

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	GENERATED_BY = "// go2array Auto-generated file, do not edit"
)

type options struct {
	variable        string
	variablePrefix  string
	platform        string
	packageName     string
	exportVariables bool
	flatHierarchy   bool
	ignoreFilelist  bool
}

var (
	Options = &options{}

	preloadByteSymbols = make(map[byte]string)
)

func init() {
	for c := byte(0x00); c < byte(0xFF); c++ {
		preloadByteSymbols[c] = fmt.Sprintf("0x%02x, ", c)
	}
	preloadByteSymbols[0xFF] = fmt.Sprintf("0x%02x, ", 0xFF)
}

func main() {
	flag.StringVar(&Options.variable, "var", "", "String for the fixed variable name")
	flag.StringVar(&Options.variablePrefix, "prefix", "", "String to prefix the variable names")
	flag.StringVar(&Options.platform, "platform", "", "Platform file name suffix to append")
	flag.StringVar(&Options.packageName, "package", "binaries", "String to use as package name")
	flag.BoolVar(&Options.exportVariables, "export", false, "Exports also variables")
	flag.BoolVar(&Options.flatHierarchy, "flat", false, "Flatten hierarchy")
	flag.BoolVar(&Options.ignoreFilelist, "nolist", false, "Don't produce list file")

	flag.Parse()
	if !flag.Parsed() {
		log.Fatalf("Could not parse command lines\n")
	}

	if len(Options.variablePrefix) == 0 && len(Options.variable) == 0 {
		log.Fatalf("Variables prefix value must be non-empty\n")
	}
	if len(Options.variablePrefix) > 0 && len(Options.variable) > 0 {
		log.Fatalf("The variables prefix or the fixed name must be non-empty, not both\n")
	}
	if len(Options.packageName) == 0 {
		log.Fatalf("Package name must be non-empty\n")
	}

	listFilename := fmt.Sprintf("%s_%s.go", Options.packageName, getSuffix("filelist", Options.platform))
	listName := ""
	dataFilename := ""
	if len(Options.variable) > 0 {
		listName = strings.Title(fmt.Sprintf("%sList", Options.variable))

		dataFilename = fmt.Sprintf("%s_%s_%s.go", Options.packageName, Options.variable, getSuffix("data", Options.platform))
	} else {
		listName = strings.Title(fmt.Sprintf("%sList", Options.variablePrefix))

		dataFilename = fmt.Sprintf("%s_%s.go", Options.packageName, getSuffix("data", Options.platform))
	}

	// list file create
	var fList *os.File
	var err error
	if !Options.ignoreFilelist {
		fList, err = os.OpenFile(listFilename, os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			log.Fatalf("Could not create file %s", listFilename)
		}
		defer fList.Close()

		fList.WriteString(fmt.Sprintf("package %s\n\n", Options.packageName))
		fList.WriteString(fmt.Sprintf("var %s map[string][]byte \n\n", listName))
		fList.WriteString("func init(){ \n")
		fList.WriteString(fmt.Sprintf("\t%s = make(map[string][]byte)	\n", listName))
	}

	// data file create
	fData, err := os.OpenFile(dataFilename, os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("Could not create file %s", dataFilename)
	}
	defer fData.Close()

	fData.WriteString(fmt.Sprintf("package %s\n\n", Options.packageName))

	// Serial FS scan
	log.Println("list file: ", listFilename, "| ignored: ", Options.ignoreFilelist, "| data file: ", dataFilename)
	harvestPaths := make(map[string][]string)
	totalFiles := 0
	for _, inFile := range flag.CommandLine.Args() {
		f, err := os.Open(inFile)
		if err != nil {
			panic(err)
		}
		stat, err := f.Stat()
		_ = f.Close()
		if err != nil {
			continue
		}

		if !stat.IsDir() {
			harvestPaths[inFile] = []string{} // single file does not have any subresults
			totalFiles++
			continue
		}
		if stat.IsDir() && len(Options.variable) > 0 {
			log.Fatalf("Cannot harvest directory with fixed variable name\n")
			return
		}

		log.Printf("Will harvest the '%s' directory\n", inFile)
		harvestPaths[inFile] = make([]string, 0, 32)
		_ = filepath.WalkDir(inFile, func(path string, dEntry fs.DirEntry, err error) error {
			if err != nil || dEntry.IsDir() {
				return nil
			}
			harvestPaths[inFile] = append(harvestPaths[inFile], path)
			totalFiles++
			return nil
		})
	}

	// Parallel load
	wg := &sync.WaitGroup{}
	wg.Add(totalFiles)

	lock := &sync.Mutex{}

	index := 0
	for root, harvestFiles := range harvestPaths {
		if len(harvestFiles) == 0 {
			go func(_index int, _path, _inFile string) {
				writeFileToPackage(_index, _path, listName, _inFile, fList, fData, lock)
				wg.Done()
			}(index, root, "")
			index++
			continue
		}

		for _, harvestFile := range harvestFiles {
			go func(_index int, _path, _inFile string) {
				writeFileToPackage(_index, _path, listName, _inFile, fList, fData, lock)
				wg.Done()
			}(index, harvestFile, root)
			index++
		}
	}

	wg.Wait()

	// close the list
	if fList != nil {
		fList.WriteString("}\n\n")
	}
}

func writeFileToPackage(inFileIndex int, inFileName, listName, baseFilename string, fList, fData *os.File, lock *sync.Mutex) {
	origFileName := inFileName
	inFileName = strings.ReplaceAll(inFileName, baseFilename, "")
	inFileName = strings.ReplaceAll(inFileName, `\`, "/")
	inFileName = filepath.ToSlash(inFileName)
	if inFileName[0] == '/' {
		inFileName = inFileName[1:]
	}
	if Options.flatHierarchy {
		inFileName = filepath.Base(inFileName)
	}

	varname := ""
	if len(Options.variable) > 0 {
		varname = Options.variable
	} else {
		varname = fmt.Sprintf("%s_%03d", Options.variablePrefix, inFileIndex)
	}
	if Options.exportVariables {
		varname = strings.Title(varname)
	}
	log.Println("file: ", inFileName, "| base: ", baseFilename, "| varname: ", varname)

	// add the variable to the list if not ignored
	if fList != nil {
		lock.Lock()
		_, _ = fList.WriteString(fmt.Sprintf("\t%s[\"%s\"] = %s \n", listName, inFileName, varname))
		lock.Unlock()
	}

	// open the input file
	fInData, err := os.OpenFile(origFileName, os.O_RDONLY, 0777)
	if err != nil {
		log.Fatalf("Could not read file %s", inFileName)
	}
	defer fInData.Close()

	//_, _ = fInData.Seek(0, syscall.FILE_BEGIN)
	// fInData.Read()

	// Read the bytes and convert the file
	reader := bufio.NewReader(fInData)

	buffer := bytes.NewBufferString("")

	buffer.WriteString(fmt.Sprintf("// original file: %s\n", origFileName))
	buffer.WriteString(fmt.Sprintf("var %s []byte = []byte{", varname))

	count := 0
	for char, err := reader.ReadByte(); err == nil; char, err = reader.ReadByte() {
		if count%16 == 0 {
			buffer.WriteString("\n\t")
		}
		buffer.WriteString(preloadByteSymbols[char])
		count++
	}

	// close the variable
	buffer.WriteString("\n}\n\n")

	// flush to the output file
	lock.Lock()
	defer lock.Unlock()
	_, _ = fData.WriteString(buffer.String())
	_ = fData.Sync()
}

func getSuffix(base, suffix string) string {
	if len(suffix) > 0 {
		return base + "_" + suffix
	}
	return base
}
