// Utility to convert files into a Go byte arrays

// Based on http://github.com/cratonica/2goarray by Clint Caywood

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	GENERATED_BY = "// Auto-generated file, do not edit"
)

type Options struct {
	variablePrefix  string
	packageName     string
	exportVariables bool
}

func main() {
	var options = &Options{}

	flag.StringVar(&options.variablePrefix, "prefix", "binaries", "String to prefix the variable names")
	flag.StringVar(&options.packageName, "package", "binaries", "String to use as package name")
	flag.BoolVar(&options.exportVariables, "export", false, "Exports also variables")

	flag.Parse()
	if !flag.Parsed() {
		log.Fatalf("Could not parse command lines\n")
	}

	if len(options.variablePrefix) == 0 {
		log.Fatalf("Variables's prefix value must be non-empty\n")
	}
	if len(options.packageName) == 0 {
		log.Fatalf("Package name must be non-empty\n")
	}

	listFilename := fmt.Sprintf("%s_filelist.go", options.variablePrefix)
	dataFilename := fmt.Sprintf("%s_data.go", options.variablePrefix)
	listName := strings.Title(fmt.Sprintf("%sList", options.variablePrefix))

	// list file create
	fList, err := os.OpenFile(listFilename, os.O_CREATE, 0777)
	if err != nil {
		log.Fatalf("Could not create file %s", listFilename)
	}
	defer fList.Close()

	fList.WriteString(fmt.Sprintf("package %s\n\n", options.packageName))
	fList.WriteString(fmt.Sprintf("var %s map[string][]byte \n\n", listName))
	fList.WriteString("func init(){ \n")
	fList.WriteString(fmt.Sprintf("\t%s = make(map[string][]byte)	\n", listName))

	// data file create
	fData, err := os.OpenFile(dataFilename, os.O_CREATE, 0777)
	if err != nil {
		log.Fatalf("Could not create file %s", dataFilename)
	}
	defer fData.Close()

	fData.WriteString(fmt.Sprintf("package %s\n\n", options.packageName))

	log.Println("list file: ", listFilename)
	log.Println("data file: ", dataFilename)
	for index, inFile := range flag.CommandLine.Args() {
		func(inFileIndex int, inFileName string) {
			log.Println("file: ", inFileName)

			varname := fmt.Sprintf("%s_%d", options.variablePrefix, inFileIndex)
			if options.exportVariables {
				varname = strings.Title(varname)
			}
			log.Println("varname: ", varname)

			// add the variable to the list
			fList.WriteString(fmt.Sprintf("%s[\"%s\"] = %s \n", listName, inFileName, varname))

			// open the input file
			fInData, err := os.OpenFile(inFileName, os.O_RDONLY, 0777)
			if err != nil {
				log.Fatalf("Could not read file %s", inFileName)
			}
			defer fInData.Close()

			// Read the bytes and convert the file
			br := bufio.NewReader(fInData)

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

			log.Println("End file 1")
		}(index, inFile)

		log.Println("End file 2")
	}

	// close the list
	fList.WriteString("}\n\n")
}
