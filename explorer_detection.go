package main

import (
 "sync"
 "strings"
 "os"
 "bufio"
 "time"
 "io"
 "fmt"
 "sort"
 "regexp"
 "strconv"
 "compress/gzip"
 "archive/zip"
)

func testDetectionLogCollection(collection *LogCollection) {
	fmt.Printf("Start Path %s\n####################\n",collection.basepath)
	for _,tree := range collection.all {

		fmt.Printf("\tTree %s\n",tree.name)
		for _,log := range tree.logs {
			
			fmt.Printf("\t\tLog %s %s type : %s\n",log.prefix,log.name,log.logtype)
			for _,plog := range log.parts {
				fmt.Printf("\t\t Part %d %s %s \n",plog.seq,plog.prefix,plog.name)
				if(plog.firstimeUnix != 0) {
					fmt.Printf("\t\t\t Epoch Time : %d\n",plog.firstimeUnix)
				} else {
					fmt.Printf("\t\t\t No epoch time detected or just didn't try :\n")
				}
			}
			fmt.Println("")
		}
		fmt.Println("")
		fmt.Println("")
	}
}

func detectType(log * Log, settings Settings) {
	detecttype := "unknown"
	detected := false
	
	partssize := len(log.parts)
	for seq := 0;seq < partssize && !detected;seq++  {
		part := log.parts[seq]
		
		f, err := os.Open(part.path)
		defer f.Close()
		errorSoft(fmt.Sprintf("Could not open %s",part.path),err)
		
		
		reader := bufio.NewReader(f)
		
		scandone := false
		for str,errread := reader.ReadString('\n');(!scandone && !detected && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
			if strarrg := jobmatchgeneric.FindStringSubmatch(str);len(strarrg) > 1 {
					detecttype = cleanupName(strarrg[1])
					detected = true
			} else if strarrc := jobmatchcmdline.FindStringSubmatch(str);len(strarrc) > 1  {
				if(strarrc[1] != "backupjob") {
					detecttype = cleanupName(strarrc[1])
					detected = true
				}
			}
			
			if (errread == io.EOF) {
				scandone = true
			}
		}
	}
	
	if(detected) {
		log.logtype = detecttype
		//fmt.Printf("Detected %s : %s\n",log.logtype,log.path)
	} else {
		log.logtype = detecttype
		fmt.Printf("Unknown logtype %s\n",log.path)
	}
}


func detectFirstTime(log * Log, settings Settings) {

	partssize := len(log.parts)
	for seq := 0;seq < partssize;seq++  {
		part := log.parts[seq]
		
		f, err := os.Open(part.path)
		defer f.Close()
		errorSoft(fmt.Sprintf("Could not open %s",part.path),err)
		
		reader := bufio.NewReader(f)
		
		detected := false
		detectedtime := time.Unix(0,0)
		
		scandone := false
		
		for str,errread := reader.ReadString('\n');(!scandone && !detected && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
			if strarrg := jobtimeline.FindStringSubmatch(str);len(strarrg) > 1 {
					t,v :=  logtimeToEpoch(&strarrg[1],settings.utchours,settings.utcminutes)
					if(t != -1) {
						detected = true
						detectedtime = *v
					}
					//fmt.Println(strarrg[4])
			} 
			if (errread == io.EOF) {
				scandone = true
			}
		}
		part.firsttime = &detectedtime
		part.firstimeUnix = detectedtime.Unix()
		if(!detected) {
			fmt.Printf("Unknown firsttime detection (is this a veeam log?) assigning fake %s \n\t %s\n",part.firsttime.Format(time.RFC3339),part.path)
		} else {
			//fmt.Printf("Detected %s : \n\t%s\n",part.firsttime.Format(time.RFC3339),part.path)
		}
	}
	
	sort.Sort(LogByFirstTime(log.parts))
}

//settings is passed without point so it is thread safe (copy of a very small struct)

func detectConcurrent(collection * LogCollection,detectBase bool,prefixFilter *[]string,  detectionFN func(*Log,Settings), settingsptr *Settings) {
	prefixFilterOn := false
	if(prefixFilter != nil) {
		prefixFilterOn = true
	}
	
	var wg sync.WaitGroup
	for _,tree := range collection.all {
		for i,_ := range tree.logs {
			log := tree.logs[i]
			shouldScan := true
			
			if(!detectBase && tree.base) {
				shouldScan = false
			}
			if(prefixFilterOn) {
				prefixMatch := false
				for _,filter := range (*prefixFilter) {
					if strings.EqualFold(filter,log.prefix) {
						prefixMatch = true
					}
				}
				
				if(!prefixMatch) {
					shouldScan = false
				}
			}
			
			if(shouldScan) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					detectionFN(log,*settingsptr)
				}()
			}
		}
	}
	wg.Wait()		
}

func findLogByPrefixAndName(collection *LogCollection,prefix string,name string)  (bool,*Log) {
	var svcbackuplog *Log
	found := false
	
	for i :=0;i<len(collection.all) && !found;i++ {
		tree := collection.all[i]
		if tree.base {
			for j :=0;j<len(tree.logs) && !found;j++ {
				log := tree.logs[j]
				if(strings.EqualFold(log.name,name) && strings.EqualFold(log.prefix,prefix)) {
					found = true
					svcbackuplog = log
				}
			}
		}
	}
	
	return found,svcbackuplog
}

func detectUTC(col *LogCollection) (int,int) {
	lutcvar := 0
	lutcminvar := 0
	
	found,svcbackuplog := findLogByPrefixAndName(col,"svc","veeambackup")
	
	
	
	if(found){
		
		detected := false
		
		utcdetect := regexp.MustCompile("Time zone has been set to \\(UTC([-+0-9]+):([0-9]+)\\)")
		
		partssize := len(svcbackuplog.parts)
		for seq := 0;seq < partssize && !detected;seq++  {
			
		
			part := svcbackuplog.parts[seq]
			
			
			

			var reader *bufio.Reader
			var err error
			if part.compressed && part.compressedType == "gz" {
				f, errf := os.Open(part.compressedParentPath)
				errorSoft(fmt.Sprintf("Could not open %s",part.compressedParentPath),errf)
				defer f.Close()
				
				if (errf == nil) {
					gz, errg := gzip.NewReader(f)
					errorSoft(fmt.Sprintf("Could not open %s",part.compressedParentPath),errg)
					defer gz.Close()
					if(errg == nil) {
						reader = bufio.NewReader(gz)
					} else {
						err = errg
					}
				} else {
					err = errf
				}
				
			} else if part.compressed && part.compressedType == "zip" {
				f, errf := zip.OpenReader(part.compressedParentPath)
				errorSoft(fmt.Sprintf("Could not open %s",part.compressedParentPath),errf)
					
				defer f.Close()
				if (errf == nil) {	
					for x := 0;x < len(f.File);x++ {
						zf := f.File[x]
						if part.path == zf.Name {
							zfr, errz := zf.Open()
							errorSoft(fmt.Sprintf("Could not open %s",part.compressedParentPath),errz)
							defer zfr.Close()
							if(errz == nil) {
								reader = bufio.NewReader(zfr)
							} else {
								err = errz
							}
						}
					}
				} else {
					err = errf
				}
			}else {
				f, errf := os.Open(part.path)
				errorSoft(fmt.Sprintf("Could not open %s",part.path),errf)
				defer f.Close()
				
				if (errf == nil) {
					reader = bufio.NewReader(f)
				} else {
					err = errf
				}
			}
					
			
			
			
			if err == nil && reader != nil {
				fmt.Printf("UTC detect on %s logs\n",part.path)
				scandone := false
				for str,errread := reader.ReadString('\n');(!scandone && !detected && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
					if strarrg := utcdetect.FindStringSubmatch(str);len(strarrg) > 1 {
							fmt.Printf("Got match now process\n")
							
							lutcvar,err = strconv.Atoi(strarrg[1])
							if (err == nil) {
								lutcminvar,err = strconv.Atoi(strarrg[2])
								if(err != nil ) {
									lutcminvar = 0
								} else {
									fmt.Printf("Autodected UTC %d:%d offset\n",lutcvar,lutcminvar)
								}
							} else {
								lutcvar = 0
								lutcminvar =0			
							}
							
							detected = true
					} 
					if (errread == io.EOF) {
						scandone = true
					}
				}
			} else {
				fmt.Printf("Unable to open %s log \n",part.path)
				fmt.Printf("%s\n",err)
			}
		}
		
		
		
	}
	return lutcvar,lutcminvar
}