package main

import (
 "sync"
 "strings"
 "time"
 "io"
 "fmt"
 "sort"
 "regexp"
 "strconv"
)

func testDetectionLogCollection(collection *LogCollection) {
	fmt.Printf("Start Path %s\n####################\n",collection.Basepath)
	for _,tree := range collection.All {

		fmt.Printf("\tTree %s\n",tree.Name)
		for _,log := range tree.Logs {
			
			fmt.Printf("\t\tLog %s %s type : %s\n",log.Prefix,log.Name,log.Logtype)
			for _,plog := range log.Parts {
				fmt.Printf("\t\t Part %d %s %s \n",plog.Seq,plog.Prefix,plog.Name)
				if(plog.FirstimeUnix != 0) {
					fmt.Printf("\t\t\t Epoch Time : %d\n",plog.FirstimeUnix)
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
	
	partssize := len(log.Parts)
	for seq := 0;seq < partssize && !detected;seq++  {
		part := log.Parts[seq]
		
		plr,err := NewPartialLogReader(part)
		defer plr.Close()
		reader := plr.reader
		
		if  err == nil && reader != nil {
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
		} else {
				fmt.Printf("Unable to open %s log \n",part.Path)
				fmt.Printf("%s\n",err)
		}
	}
	
	if(detected) {
		log.Logtype = detecttype
		//fmt.Printf("Detected %s : %s\n",log.Logtype,log.Path)
	} else {
		log.Logtype = detecttype
		fmt.Printf("Unknown logtype %s\n",log.Path)
	}
}


func detectFirstTime(log * Log, settings Settings) {

	partssize := len(log.Parts)
	for seq := 0;seq < partssize;seq++  {
		part := log.Parts[seq]
		
		plr,err := NewPartialLogReader(part)
		defer plr.Close()
		reader := plr.reader
		
		if  err == nil && reader != nil {
			detected := false
			detectedtime := time.Unix(0,0)
			
			scandone := false
			
			for str,errread := reader.ReadString('\n');(!scandone && !detected && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
				if strarrg := jobtimelineglobal.FindStringSubmatch(str);len(strarrg) > 1 {
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
			part.Firsttime = &detectedtime
			part.FirstimeUnix = detectedtime.Unix()
			if(!detected) {
				fmt.Printf("Unknown firsttime detection (is this a veeam log?) assigning fake %s \n\t %s\n",part.Firsttime.Format(time.RFC3339),part.Path)
			} else {
				//fmt.Printf("Detected %s : \n\t%s\n",part.Firsttime.Format(time.RFC3339),part.Path)
			}
		} else {
				fmt.Printf("Unable to open %s log \n",part.Path)
				fmt.Printf("%s\n",err)
		}
	}
	
	sort.Sort(LogByFirstTime(log.Parts))
}

//settings is passed without point so it is thread safe (copy of a very small struct)
//filter is prefix not type (cause this might not be detected)
func detectConcurrent(collection * LogCollection,detectBase bool,prefixFilter *[]string,  detectionFN func(*Log,Settings), settingsptr *Settings) {
	prefixFilterOn := false
	if(prefixFilter != nil) {
		prefixFilterOn = true
	}
	
	var wg sync.WaitGroup
	for _,tree := range collection.All {
		for i,_ := range tree.Logs {
			log := tree.Logs[i]
			shouldScan := true
			
			if(!detectBase && tree.Base) {
				shouldScan = false
			}
			if(prefixFilterOn) {
				prefixMatch := false
				for _,filter := range (*prefixFilter) {
					if strings.EqualFold(filter,log.Prefix) {
						prefixMatch = true
					} else {
						//fmt.Println("Ignoring "+log.Prefix)
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
	
	for i :=0;i<len(collection.All) && !found;i++ {
		tree := collection.All[i]
		if tree.Base {
			for j :=0;j<len(tree.Logs) && !found;j++ {
				log := tree.Logs[j]
				if(strings.EqualFold(log.Name,name) && strings.EqualFold(log.Prefix,prefix)) {
					found = true
					svcbackuplog = log
				}
			}
		}
	}
	
	return found,svcbackuplog
}

func detectUTC(col *LogCollection,settingsptr *Settings) {
	lutcvar := 0
	lutcminvar := 0
	detected := false
	
	found,svcbackuplog := findLogByPrefixAndName(col,"svc","veeambackup")
	
	
	if(found){
		
		
		utcdetect := regexp.MustCompile("Time zone has been set to \\(UTC([-+0-9]+):([0-9]+)\\)")
		
		partssize := len(svcbackuplog.Parts)
		for seq := 0;seq < partssize && !detected;seq++  {
			
		
			part := svcbackuplog.Parts[seq]
			
			
			plr,err := NewPartialLogReader(part)
			defer plr.Close()
			reader := plr.reader
					
			
			
			if err == nil && reader != nil {
				//fmt.Printf("UTC detect on %s logs\n",part.Path)
				scandone := false
				for str,errread := reader.ReadString('\n');(!scandone && !detected && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
					if strarrg := utcdetect.FindStringSubmatch(str);len(strarrg) > 1 {
							//fmt.Printf("Got match now process\n")
							
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
				fmt.Printf("Unable to open %s log \n",part.Path)
				fmt.Printf("%s\n",err)
			}
		}	
	}
	if(!detected) {
		fmt.Printf("Could not detect utc, defaulting to +0:00 \n")
	}
	settingsptr.utchours = lutcvar
	settingsptr.utcminutes = lutcminvar
}