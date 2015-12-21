package main
import (
	"fmt"
	"sync"
	"strings"
	"io"
	"time"
)

func testStampingLogCollection(collection *LogCollection) {
	fmt.Printf("Start Path %s\n####################\n",collection.basepath)
	for _,tree := range collection.all {

		fmt.Printf("\tTree %s\n",tree.name)
		for _,log := range tree.logs {
			
			fmt.Printf("\t\tLog %s %s type : %s\n",log.prefix,log.name,log.logtype)
			for _,plog := range log.parts {
				fmt.Printf("\t\t Part %d %s %s \n",plog.seq,plog.prefix,plog.name)
				for _,event := range plog.events {
					fmt.Printf("\t\t\t Event %10d %s \n",event.lineNumber,event.description)
				}
			}
			fmt.Println("")
		}
		fmt.Println("")
		fmt.Println("")
	}
}


func timeStampLogs(tree *LogTree,settings Settings) {
	var wg sync.WaitGroup
	
	
	for i,_ := range tree.logs {
		wg.Add(1)
		log := tree.logs[i]
		go func(log *Log) {
			defer wg.Done()
			partssize := len(log.parts)
			
			
			for seq := 0;seq < partssize;seq++  {
				prevtimestamp := int64(0)
				part := log.parts[seq]
			
				plr,err := NewPartialLogReader(part)
				defer plr.Close()
				reader := plr.reader
				
				if  err == nil && reader != nil {
					scandone := false
					lineno := 0
					for str,errread := reader.ReadString('\n');(!scandone && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
						lineno++
						if(lineno%settings.fastSkipping == 0) {
							if strarrg := jobtimeline.FindStringSubmatch(str);len(strarrg) > 1 {
								t,v :=  logtimeToEpoch(&strarrg[1],settings.utchours,settings.utcminutes)
								if(t != -1 ) {
									if (t-prevtimestamp) > settings.skew {
										prevtimestamp = t
										event := Event{lineNumber:lineno,description:fmt.Sprintf("Time Stamp at : %s",v.Format(time.RFC3339)),originalLine:str}
										part.events = append(part.events,&event)
									}
								}
							} 
							if (errread == io.EOF) {
								scandone = true
							}
						}
					}
				}
			} 
		}(log)
	}
	wg.Wait()
	fmt.Printf("Parsed %s\n",tree.name)
}


func doNothing(tree *LogTree,settings Settings) { }
func parseLogTree(collection * LogCollection,detectBase bool,treeFilter *[]string,  detectionFN func(*LogTree,Settings), settingsptr *Settings) (* LogCollection) {
	subcol := LogCollection{basepath:collection.basepath}
	
	treeFilterOn := false
	if(treeFilter != nil) {
		treeFilterOn = true
	}
	
	var wg sync.WaitGroup
	
	for a,_ := range collection.all {
		tree := collection.all[a]
		shouldScan := true
		
		if(!detectBase && tree.base) {
			shouldScan = false
		} else if treeFilterOn {
			gotMatch := false
			for i,_ := range tree.logs {
				log := tree.logs[i]
				
				for _,filter := range (*treeFilter) {
					if strings.EqualFold(filter,log.logtype) {
						gotMatch = true
						//fmt.Printf("%s %s\n",filter,log.logtype)
					}
				}
			}
			if(!gotMatch) {
				shouldScan = false
			}
		}
		
		if shouldScan {
			subcol.all = append(subcol.all,tree)
			wg.Add(1)
	
			go func() {
					defer wg.Done()
					detectionFN(tree,*settingsptr)
			}()
			
		}
	}
	wg.Wait()
	return &subcol
}