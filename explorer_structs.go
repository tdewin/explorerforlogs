package main
import ( 
	"time"
)
//total collection
type LogCollection struct {
	Basepath string
	All []*LogTree
}
//files related to eachother (job files and task files are essentially a split)
type LogTree struct {
	Name string
	Base bool
	Logs []*Log
}
//a log file consist maybe out of multiple segments
type Log struct {
	Name string
	Prefix string
	Directory string
	Path string
	Logtype string
	Parts []*PartialLog
	Genericname string
}
type PartialLog struct {
  Seq int
  Prefix string
  Name string
  Filename string
  Path string
  Directory string
  Firsttime *time.Time
  FirstimeUnix int64
  Compressed bool
  CompressedType string
  CompressedParentPath string
  Events []*Event
}
type Event struct {
	LineNumber int
	Description string
	OriginalLine string
}



type LogByFirstTime []*PartialLog
func (a LogByFirstTime) Len() int           { return len(a) }
func (a LogByFirstTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LogByFirstTime) Less(i, j int) bool { return a[i].FirstimeUnix < a[j].FirstimeUnix }

type Settings struct {
	utchours int
	utcminutes int
	skew int64
	fastSkipping int
}