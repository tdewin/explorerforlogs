package main

import (
	"time"
	"regexp"
	"strconv"
	"fmt"
	"strings"
)

var jobtimeformat *regexp.Regexp
var jobtimeline *regexp.Regexp

var jobmatchgeneric *regexp.Regexp
var jobmatchcmdline *regexp.Regexp

func globals() {
	jobtimeformat = regexp.MustCompile("(?:\\[)?([0-9]{1,2}).([0-9]{1,2}).([0-9]{4}) ([0-9]{1,2}):([0-9]{1,2}):([0-9]{1,2})(?:\\])?")
	jobtimeline = regexp.MustCompile("^\\[([0-9]{1,2}.[0-9]{1,2}.[0-9]{4} [0-9]{1,2}:[0-9]{1,2}:[0-9]{1,2})\\]\\s*[<]([0-9]+)[>]\\s*(Info|Warning|Error)\\s*(.*)")
	
	jobmatchgeneric = regexp.MustCompile("^\\[[^\\]]*\\] \\<[0-9]*\\> Info \\s* Job Type: ['\\[]([a-zA-Z ]*)['\\]]")
	jobmatchcmdline = regexp.MustCompile("^CmdLineParams: \\[(?:START|start)([a-zA-Z]+)")
}


func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
func logtimeToEpoch(inputString *string,utc int,utcmin int) (int64,*time.Time) {
	if strarrg := jobtimeformat.FindStringSubmatch(*inputString);len(strarrg) > 1 {
		day,_ := strconv.Atoi(strarrg[1])
		month,_ := strconv.Atoi(strarrg[2])
		year,_ := strconv.Atoi(strarrg[3])
		hour,_ := strconv.Atoi(strarrg[4])
		min,_ := strconv.Atoi(strarrg[5])
		sec,_ := strconv.Atoi(strarrg[6])
		

		t, _ := time.Parse(time.RFC3339,fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%+03d:%02d",year,month,day,hour,min,sec,utc,utcmin))
		return t.Unix(),&t
	}
	t := time.Now()
	return -1,&t
}
func spaceName(str string) (string) {
	return strings.Replace(str,"_"," ",-1)
}
func cleanupName(str string) (string) {
	str = strings.ToLower(strings.Replace(str," ","",-1))
	return str
}