// Utility to convert files into a Go byte arrays

// Based on http://github.com/cratonica/2goarray by Clint Caywood

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	GENERATED_BY = "// Auto-generated file, do not edit"
)

type options struct {
	variablePrefix  string
	packageName     string
	exportVariables bool
	flatHierarchy   bool
}

var Options = &options{}

func main() {
	flag.StringVar(&Options.variablePrefix, "prefix", "binaries", "String to prefix the variable names")
	flag.StringVar(&Options.packageName, "package", "binaries", "String to use as package name")
	flag.BoolVar(&Options.exportVariables, "export", false, "Exports also variables")
	flag.BoolVar(&Options.flatHierarchy, "flat", false, "Flatten hierarchy")

	flag.Parse()
	if !flag.Parsed() {
		log.Fatalf("Could not parse command lines\n")
	}

	if len(Options.variablePrefix) == 0 {
		log.Fatalf("Variables's prefix value must be non-empty\n")
	}
	if len(Options.packageName) == 0 {
		log.Fatalf("Package name must be non-empty\n")
	}

	listFilename := fmt.Sprintf("%s_filelist.go", Options.variablePrefix)
	dataFilename := fmt.Sprintf("%s_data.go", Options.variablePrefix)
	listName := strings.Title(fmt.Sprintf("%sList", Options.variablePrefix))

	// list file create
	fList, err := os.OpenFile(listFilename, os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("Could not create file %s", listFilename)
	}
	defer fList.Close()

	fList.WriteString(fmt.Sprintf("package %s\n\n", Options.packageName))
	fList.WriteString(fmt.Sprintf("var %s map[string][]byte \n\n", listName))
	fList.WriteString("func init(){ \n")
	fList.WriteString(fmt.Sprintf("\t%s = make(map[string][]byte)	\n", listName))

	// data file create
	fData, err := os.OpenFile(dataFilename, os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Fatalf("Could not create file %s", dataFilename)
	}
	defer fData.Close()

	fData.WriteString(fmt.Sprintf("package %s\n\n", Options.packageName))

	log.Println("list file: ", listFilename)
	log.Println("data file: ", dataFilename)
	index := 0
	for _, inFile := range flag.CommandLine.Args() {
		f, err := os.Open(inFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if stat, err := f.Stat(); err == nil {
			if !stat.IsDir() {
				writeFileToPackage(index, inFile, listName, listFilename, fList, fData)
				continue
			}

			log.Printf("Will harvest the '%s' directory\n", inFile)
			filepath.WalkDir(inFile, func(path string, dEntry fs.DirEntry, err error) error {
				if err != nil || dEntry.IsDir() {
					return nil
				}
				index++

				writeFileToPackage(index, path, listName, listFilename, fList, fData)
				return nil
			})
		}
		index++
	}

	// close the list
	fList.WriteString("}\n\n")
}

func writeFileToPackage(inFileIndex int, inFileName, listName, listFilename string, fList, fData *os.File) {
	origFileName := inFileName
	if Options.flatHierarchy {
		inFileName = filepath.Base(inFileName)
	}
	log.Println("file: ", inFileName)

	varname := fmt.Sprintf("%s_%03d", Options.variablePrefix, inFileIndex)
	if Options.exportVariables {
		varname = strings.Title(varname)
	}
	log.Println("varname: ", varname)

	// add the variable to the list
	fList.WriteString(fmt.Sprintf("%s[\"%s\"] = %s \n", listName, inFileName, varname))

	// open the input file
	fInData, err := os.OpenFile(origFileName, os.O_RDONLY, 0777)
	if err != nil {
		log.Fatalf("Could not read file %s", inFileName)
	}
	defer fInData.Close()

	// Read the bytes and convert the file
	br := bufio.NewReader(fInData)

	fData.WriteString(fmt.Sprintf("// original file: %s\n", origFileName))
	fData.WriteString(fmt.Sprintf("var %s []byte = []byte{", varname))

	count := 0
	for buf, err := br.ReadByte(); err == nil; buf, err = br.ReadByte() {
		if count%12 == 0 {
			fData.WriteString("\n\t")
		}
		fData.WriteString(fmt.Sprintf("0x%02x, ", buf))
		count++
	}

	// close the variable
	fData.WriteString("\n}\n\n")
}
