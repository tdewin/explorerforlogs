package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"runtime"
)


func timestamp(basepath string,settingsptr *Settings, outfile *string) {
		
		collection := buildLogIndex(basepath)
		//testIndexerLogCollection(collection)
		
		if(settingsptr.utchours == 24) {
			detectUTC(collection,settingsptr)
			fmt.Printf("Using UTC %d:%d\n",settingsptr.utchours,settingsptr.utcminutes)
		}
		
		filter := []string{"job","task"}
		detectConcurrent(collection,false,&filter, detectFirstTime, settingsptr)
		
		filter = []string{"job"}
		detectConcurrent(collection,false,&filter, detectType, settingsptr)
		
		//testDetectionLogCollection(collection)
		
		filter = []string{"vddkbackup","vddkreplica","backupbackupsync"}
		logsfordump := parseLogTree(collection,false,&filter, timeStampLogs, settingsptr)
		if (*outfile) == "" {
			testStampingLogCollection(logsfordump)
		} else {
			writeTimeStampHtml(logsfordump,outfile)
		}
		
		
}
func main() () {
	runtime.GOMAXPROCS(4)
	t0 := time.Now()
	
	//mostly regex which should be concurrency safe, set in explorer_util
	globals()
	
	//input parsing
	dirname := flag.String("dir",".","Provide log dir")
	veeamdir := flag.Bool("veeamdir",false,"Use -veeamdir to use the default log dir %programdata%\\Veeam\\Backup , -dir will be overwritten")
	action := flag.String("action","timestamp","'timestamp'")
	utc := flag.Int("utc",24,"UTC adjustment, 24 will try to autodetect")
	utcmin := flag.Int("utcmin",0,"For some countries")
	fastskipping := flag.Int("fastskipping",25,"Fast Skipping (time stamp parse every x lines)")
	outfile := flag.String("out","","'Where to output'")
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
				settings := Settings{utchours:*utc,utcminutes:*utcmin,skew:3600,fastSkipping:*fastskipping}
				
				switch *action {
					case "timestamp": {
						timestamp(basepath,&settings,outfile)
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
	t1 := time.Now()
	fmt.Printf("Parsed in %v\n", t1.Sub(t0))
}
