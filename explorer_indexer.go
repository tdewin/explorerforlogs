package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"archive/zip"
)




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
func testIndexerLogCollection(collection *LogCollection) {
	fmt.Printf("Start Path %s\n####################\n",collection.basepath)
	for _,tree := range collection.all {

		fmt.Printf("\tTree %s\n",tree.name)
		for _,log := range tree.logs {
			
			fmt.Printf("\t\tLog %s %s type : %s\n",log.prefix,log.name,log.logtype)
			for _,plog := range log.parts {
				fmt.Printf("\t\t Part %d %s %s  \n",plog.seq,plog.prefix,plog.name)
				if(plog.compressed) {
						fmt.Printf("\t\t   Compressed %s file %s in  %s \n",plog.compressedType,plog.path,plog.compressedParentPath)
				} else {
						fmt.Printf("\t\t   Plain %s \n",plog.path)
				}
			}
			fmt.Println("")
		}
		fmt.Println("")
		fmt.Println("")
	}
}

func addToMatchingTreeLog(plog *PartialLog,logtree *LogTree,base bool) {
	var log *Log
				
	for _,matchinglog := range logtree.logs {
		if(matchinglog.prefix == plog.prefix && matchinglog.name == plog.name && matchinglog.path == plog.directory) {
			log = matchinglog
		} 
	}
	if(log == nil) {
		if(base) {
			newlog := Log{prefix:plog.prefix,name:plog.name,path:plog.directory,logtype:(fmt.Sprintf("%s.%s",plog.prefix,plog.name)),genericname:(fmt.Sprintf("%s.%s.<x>.log",plog.prefix,plog.name))}
			log = &newlog
		} else {
			newlog := Log{prefix:plog.prefix,name:plog.name,path:plog.directory,logtype:plog.prefix,genericname:(fmt.Sprintf("%s.%s.<x>.log",plog.prefix,plog.name))}
			log = &newlog
		}
		
		logtree.logs = append(logtree.logs,log)
	}
	log.parts = append(log.parts,plog)
}

func partialLog(dir string, file string, strarr *[]string,compressed bool,compressedType string,parentfile string) (*PartialLog) {
	var filepathstr string
	parentpathstr := ""
	
	if(compressed) {
		filepathstr = file
		parentpathstr = path.Clean(filepath.Join(dir,parentfile))
	} else {
		filepathstr = path.Clean(filepath.Join(dir,file))
	}
	prefix := (*strarr)[3]
	logname := (*strarr)[4]
	
	seqn := 0
	if((*strarr)[5] != "") {
		seqnt,err := strconv.Atoi((*strarr)[6])
		if (err == nil) { seqn = seqnt }
	}
	
	plog := PartialLog{seq:seqn,prefix:prefix,name:logname,filename:file,path:filepathstr,directory:dir,compressed:compressed,compressedType:compressedType,compressedParentPath:parentpathstr}
	//fmt.Printf("%s\n",plog.path)
	return &plog
}

func buildLogIndex(dirpath string) (*LogCollection) {
	matchfiles := regexp.MustCompile("(?i)^([0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{6})?_)?(job|task|svc|veeamagent|veeam|util|wmiserver|rts)[.](.*?)([.]([0-9]+))?[.]log$")
	matchcompressedfiles := regexp.MustCompile("(?i)^([0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{6})?_)?(job|task|svc|veeamagent|veeam|util|wmi|rts)[.](.*?)([.]([0-9]+))?[.](log[.]gz|zip)$")
	matchcompressedjobfiles := regexp.MustCompile("(?i)^([0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{6})?_)?(.*?)[.](zip)$")
	
	subgzremover := regexp.MustCompile("(?i)[.]gz$")
	
	
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
					plog := partialLog(jobpath,jobfileinfo.Name(),&strarrs,false,"","")
					addToMatchingTreeLog(plog,&jobtree,false)
				} else if strarrc := matchcompressedjobfiles.FindStringSubmatch(jobfileinfo.Name());len(strarrc) > 1 {
					if(strarrc[4] == "zip") {
						zippath := filepath.Join(jobpath,jobfileinfo.Name())
						zipr, err := zip.OpenReader(zippath)
						errorSoft(fmt.Sprintf("Could not read zip %s",zippath),err)
						defer zipr.Close()
						for _, f := range zipr.File {
							if strarrz := matchfiles.FindStringSubmatch(f.Name);len(strarrz) > 1 { 
								plog := partialLog(jobpath,f.Name,&strarrz,true,"zip",jobfileinfo.Name())
								addToMatchingTreeLog(plog,&jobtree,false)
								//fmt.Printf("\n :)))) %s %s \n",fileinfo.Name(),f.Name)
							} else {
								//fmt.Printf("\n :( %s %s :( \n",fileinfo.Name(),f.Name)
							}
						}
					}
				} else {
					
				} 
			}
			
		} else {
			if strarrs := matchfiles.FindStringSubmatch(fileinfo.Name());len(strarrs) > 1 {
				plog := partialLog(dirpath,fileinfo.Name(),&strarrs,false,"","")
				addToMatchingTreeLog(plog,&maintree,true)
			} else if strarrc := matchcompressedfiles.FindStringSubmatch(fileinfo.Name());len(strarrc) > 1 {
				if(strarrc[7] == "log.gz") { 
					plog := partialLog(dirpath,subgzremover.ReplaceAllLiteralString(fileinfo.Name(),""),&strarrc,true,"gz",fileinfo.Name())
					addToMatchingTreeLog(plog,&maintree,true)
				} else  if(strarrc[7] == "zip") {
					zippath := filepath.Join(dirpath,fileinfo.Name())
					zipr, err := zip.OpenReader(zippath)
					errorSoft(fmt.Sprintf("Could not read zip %s",),err)
					defer zipr.Close()
					for _, f := range zipr.File {
						if strarrz := matchfiles.FindStringSubmatch(f.Name);len(strarrz) > 1 { 
							plog := partialLog(dirpath,f.Name,&strarrz,true,"zip",fileinfo.Name())
							addToMatchingTreeLog(plog,&maintree,true)
						} else {
							fmt.Printf("\n :( %s %s :( \n",fileinfo.Name(),f.Name)
						}
					}
				}
			} else {
				//fmt.Printf("Don't know %s\n",fileinfo.Name())
			} 
		}
	}
	
	return &collection
	
}