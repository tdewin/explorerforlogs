package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)


func dump(basepath string,settingsptr *Settings) {
		
		collection := buildLogIndex(basepath)
		testIndexerLogCollection(collection)
		
		detectUTC(collection)
		
		//filter := []string{"job","task"}
		//detectConcurrent(collection,false,&filter, detectFirstTime, settingsptr)
		
		//filter = []string{"job"}
		//detectConcurrent(collection,false,&filter, detectType, settingsptr)
		
		//testDetectionLogCollection(collection)
}
func main() () {
	//mostly regex which should be concurrency safe, set in explorer_util
	globals()
	
	//input parsing
	dirname := flag.String("dir",".","Provide log dir")
	veeamdir := flag.Bool("veeamdir",false,"Use -veeamdir to use the default log dir %programdata%\\Veeam\\Backup , -dir will be overwritten")
	action := flag.String("action","dump","'dump'")
	flag.Parse()
	
	if(*veeamdir) {
		veeamdirpath := os.ExpandEnv("${programdata}\\Veeam\\Backup")
		dirname = &veeamdirpath
	}
	
	dirpath, error := filepath.Abs(*dirname)
	errorPanic("Path is not correct",error)
	
	if stat, err := os.Stat(dirpath); err == nil {
		if stat.IsDir() {
			found,basepath := findBase(dirpath)
			if(found) {
				fmt.Printf("Basepath is %s\n",basepath)
				settings := Settings{utchours:2,utcminutes:0}
				
				switch *action {
					case "dump": {
						dump(basepath,&settings)
					}
				}
			} else {
				fmt.Printf("Could not find the base path, check if Svc.VeeamBackup.log is somewhere available in the path %s\n",dirpath)
			}
			
		} else {
			errorPanic("Path does not exist",errors.New(fmt.Sprintf("Path does not exists %s",dirpath)))
		}
	} else {
		errorPanic("Path does not exist",error)
	}
	
}
