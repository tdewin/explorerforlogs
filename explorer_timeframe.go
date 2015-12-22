package main
import (
	"fmt"
	"sync"
	"strings"
	"io"
	"time"
)

func testStampingLogCollection(collection *LogCollection) {
	fmt.Printf("Start Path %s\n####################\n",collection.Basepath)
	for _,tree := range collection.All {

		fmt.Printf("\tTree %s\n",tree.Name)
		for _,log := range tree.Logs {
			
			fmt.Printf("\t\tLog %s %s type : %s\n",log.Prefix,log.Name,log.Logtype)
			for _,plog := range log.Parts {
				fmt.Printf("\t\t Part %d %s %s \n",plog.Seq,plog.Prefix,plog.Name)
				for _,event := range plog.Events {
					fmt.Printf("\t\t\t Event %10d %s \n",event.LineNumber,event.Description)
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
	
	
	for i,_ := range tree.Logs {
		wg.Add(1)
		log := tree.Logs[i]
		go func(log *Log) {
			defer wg.Done()
			partssize := len(log.Parts)
			
			
			for seq := 0;seq < partssize;seq++  {
				prevtimestamp := int64(0)
				part := log.Parts[seq]
			
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
										event := Event{LineNumber:lineno,Description:fmt.Sprintf("Time Stamp at : %s",v.Format(time.RFC3339)),OriginalLine:str}
										part.Events = append(part.Events,&event)
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
	fmt.Printf("Parsed %s\n",tree.Name)
}


func doNothing(tree *LogTree,settings Settings) { }
func parseLogTree(collection * LogCollection,detectBase bool,treeFilter *[]string,  detectionFN func(*LogTree,Settings), settingsptr *Settings) (* LogCollection) {
	subcol := LogCollection{Basepath:collection.Basepath}
	
	treeFilterOn := false
	if(treeFilter != nil) {
		treeFilterOn = true
	}
	
	var wg sync.WaitGroup
	
	for a,_ := range collection.All {
		tree := collection.All[a]
		shouldScan := true
		
		if(!detectBase && tree.Base) {
			shouldScan = false
		} else if treeFilterOn {
			gotMatch := false
			for i,_ := range tree.Logs {
				log := tree.Logs[i]
				
				for _,filter := range (*treeFilter) {
					if strings.EqualFold(filter,log.Logtype) {
						gotMatch = true
						//fmt.Printf("%s %s\n",filter,log.Logtype)
					}
				}
			}
			if(!gotMatch) {
				shouldScan = false
			}
		}
		
		if shouldScan {
			subcol.All = append(subcol.All,tree)
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