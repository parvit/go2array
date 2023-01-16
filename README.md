go2array
========
Utility to encode a file (or a series of files) into a Go package.

Having [set up your Go environment](http://golang.org/doc/install), simply run

```
    go get github.com/parvit/go2array
```

You must provide a name for the generated variables and the package name, the input files are specified as 
parameters at the end. For example:

```
    go2array [-prefix <variablePrefix>] [-export] [-flat] [-nolist] [-package <packageName>] [-platform <platformName>] file1 [file2 ...]
```

Default variabilePrefix is the value "binaries"

The output will be two files:

    1. <variablePrefix>_filelist.go : That will contain a map with all the variables assigned to their original name
    2. <variablePrefix>_data.go : Which will contain all the variables data

In <variablePrefix>_data.go you will find:
```
    package <packageName>

    // for each file, a variable is declared incrementing the index
    var <variablePrefix>_<fileIndex> []byte = []byte {
      0x49, 0x20, 0x63, 0x61, 0x6e, 0x27, 0x74, 0x20, 0x62, 0x65, 0x6c, 0x69,
      0x65, 0x76, 0x65, 0x20, 0x79, 0x6f, 0x75, 0x20, 0x61, 0x63, 0x74, 0x75,
      0x61, 0x6c, 0x6c, 0x79, 0x20, 0x64, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x64,
      0x20, 0x74, 0x68, 0x69, 0x73, 0x2e, 0x20, 0x4b, 0x75, 0x64, 0x6f, 0x73,
      0x20, 0x66, 0x6f, 0x72, 0x20, 0x62, 0x65, 0x69, 0x6e, 0x67, 0x20, 0x74,
      0x68, 0x6f, 0x72, 0x6f, 0x75, 0x67, 0x68, 0x2e, 0x0a,
    }
    
```

## Contributors
- [Clint Caywood](https://github.com/cratonica)
- [Paul Vollmer](https://github.com/paulvollmer)
- [Vittorio Parrella](https://github.com/parvit)
