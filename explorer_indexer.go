package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"
	"strconv"
)

type LogCollection struct {
	basepath string
	all []*LogTree
}
type LogTree struct {
	name string
	base bool
	logs []*Log
}
type Log struct {
	name string
	prefix string
	path string
	logtype string
	parts []*PartialLog
	genericname string
}
type PartialLog struct {
  seq int
  prefix string
  name string
  filename string
  path string
  directory string
  firsttime *time.Time
  firstimeUnix int64
}


func findBase(dirpath string) (bool,string) {
	matchsvcbackup := regexp.MustCompile("(?i)^svc[.]VeeamBackup([.]([0-9]+))?[.]log$")
	basepath := dirpath
	found := false
	
	errval := filepath.Walk(dirpath, func(path string, fileinfo os.FileInfo, localerror error) error {
		if(!fileinfo.IsDir()) {
			if (matchsvcbackup.MatchString(fileinfo.Name())) {
				found = true
				basepath = filepath.Dir(path)
			}
		}
		return localerror
	})
	
	if(errval != nil && !found) {
		fmt.Println("Some error occurred during Svc.VeeamBackup.log lookup. Because the log could not be find, the program will stop")
		fmt.Println(errval)
	}
	
	return found,basepath
}
func testDumpLogCollection(collection *LogCollection) {
	fmt.Printf("+%s\n",collection.basepath)
	for _,tree := range collection.all {
		fmt.Printf("++%s\n",tree.name)
		for _,log := range tree.logs {
			fmt.Printf("+++%s %s\n",log.prefix,log.name)
			for _,plog := range log.parts {
				fmt.Printf("++++%s %s %d\n",plog.prefix,plog.name,plog.seq)
			}
		}
	}
}

func partialLog(dir string, file string, strarr *[]string) (*PartialLog) {
	filepathstr := path.Clean(filepath.Join(dir,file))
	prefix := (*strarr)[1]
	logname := (*strarr)[2]
	
	seqn := 0
	if((*strarr)[3] != "") {
		seqnt,err := strconv.Atoi((*strarr)[4])
		if (err == nil) { seqn = seqnt }
	}
	
	plog := PartialLog{seq:seqn,prefix:prefix,name:logname,filename:file,path:filepathstr,directory:dir}
	//fmt.Printf("%s\n",plog.path)
	return &plog
}

func buildLogIndex(dirpath string) (*LogCollection) {
	matchfiles := regexp.MustCompile("(?i)^(job|task|svc|veeamagent|veeam|util|wmi|rts)[.](.*?)([.]([0-9]+))?[.]log$")

	
	
	files,err := ioutil.ReadDir(dirpath)
	errorPanic(fmt.Sprintf("Could not list files in %s",dirpath),err)
	
	collection := LogCollection{basepath:dirpath}
	maintree := LogTree{name:"base",base:true}
	
	collection.all = append(collection.all,&maintree)
	
	
	for _,fileinfo := range files {
		if(fileinfo.IsDir()) {
			jobpath := path.Clean(filepath.Join(dirpath,fileinfo.Name()))
			jobfiles,err := ioutil.ReadDir(jobpath)
			errorPanic(fmt.Sprintf("Could not list files in %s",jobpath),err)
			
			jobtree := LogTree{name:fileinfo.Name(),base:false}
			collection.all = append(collection.all,&jobtree)
			
			for _,jobfileinfo := range jobfiles {
				if strarrs := matchfiles.FindStringSubmatch(jobfileinfo.Name());len(strarrs) > 1 {
					plog := partialLog(jobpath,jobfileinfo.Name(),&strarrs)
					
					var log *Log
					for _,matchinglog := range jobtree.logs {
						if(matchinglog.prefix == plog.prefix && matchinglog.name == plog.name && matchinglog.path == plog.directory) {
							log = matchinglog
						}
					}
					if(log == nil) {
						newlog := Log{prefix:plog.prefix,name:plog.name,path:plog.directory,logtype:strarrs[1],genericname:(fmt.Sprintf("%s.%s.<x>.log",plog.prefix,plog.name))}
						log = &newlog
						jobtree.logs = append(jobtree.logs,log)
					}
					log.parts = append(log.parts,plog)
				}
			}
			
		} else {
			if strarrs := matchfiles.FindStringSubmatch(fileinfo.Name());len(strarrs) > 1 {
				//fmt.Printf("found file %s type %s :%s\n",strarrs[2],strarrs[1],fileinfo.Name())
				plog := partialLog(dirpath,fileinfo.Name(),&strarrs)
				
				var log *Log
				
				for _,matchinglog := range maintree.logs {
					if(matchinglog.prefix == plog.prefix && matchinglog.name == plog.name && matchinglog.path == plog.directory) {
						log = matchinglog
					}
				}
				if(log == nil) {
					newlog := Log{prefix:plog.prefix,name:plog.name,path:plog.directory,logtype:strarrs[1],genericname:(fmt.Sprintf("%s.%s.<x>.log",plog.prefix,plog.name))}
					log = &newlog
					maintree.logs = append(maintree.logs,log)
				}
				log.parts = append(log.parts,plog)
			} else {
				//fmt.Printf("Don't know %s\n",fileinfo.Name())
			}
		}
	}
	
	return &collection
	
}