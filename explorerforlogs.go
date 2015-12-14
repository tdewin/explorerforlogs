package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)


func dump(dirpath string) {
	found,basepath := findBase(dirpath)
	
	if(found) {
		fmt.Printf("Basepath is %s\n",basepath)
		fileindex := buildLogIndex(basepath)
		testDumpLogCollection(fileindex)
	} else {
		fmt.Printf("Could not find the base path, check if Svc.VeeamBackup.log is somewhere available in the path\n")
	}
	
}
func main() () {
	//input parsing
	dirname := flag.String("dir",os.ExpandEnv("${programdata}\\Veeam\\Backup"),"Provide log dir")
	action := flag.String("action","dump","'dump'")
	flag.Parse()
	
	dirpath, error := filepath.Abs(*dirname)
	errorPanic("Path is not correct",error)
	
	if stat, err := os.Stat(dirpath); err == nil {
		if stat.IsDir() {
			switch *action {
				case "dump": {
					dump(dirpath)
				}
			}
			
		} else {
			errorPanic("Path does not exist",errors.New(fmt.Sprintf("Path does not exists %s",dirpath)))
		}
	} else {
		errorPanic("Path does not exist",error)
	}
	
}
