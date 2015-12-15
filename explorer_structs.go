package main
import ( 
	"time"
)
//total collection
type LogCollection struct {
	basepath string
	all []*LogTree
}
//files related to eachother (job files and task files are essentially a split)
type LogTree struct {
	name string
	base bool
	logs []*Log
}
//a log file consist maybe out of multiple segments
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
  compressed bool
  compressedType string
  compressedParentPath string
}
type LogByFirstTime []*PartialLog
func (a LogByFirstTime) Len() int           { return len(a) }
func (a LogByFirstTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LogByFirstTime) Less(i, j int) bool { return a[i].firstimeUnix < a[j].firstimeUnix }

type Settings struct {
	utchours int
	utcminutes int
}