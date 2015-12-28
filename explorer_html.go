package main

import (
 "os"
 "fmt"
 "html/template"
 "crypto/md5"
 "io"
 "net/http"
 "sync"
 "strconv"
 "bytes"
 "time"
)

type TimeStampHTML struct {
	Title string
	JqueryCode string
	StartPath string
	All []*LogTree 
	WebServer bool
}
func (t * TimeStampHTML) getFunctionMap() (*template.FuncMap) {
	funcMap := template.FuncMap{
        "localFile": func(link string,name string) template.HTML {
            return template.HTML(fmt.Sprintf("<a href='file://%s'>%s</a>",link,name))
         },
		 "rawjs": func(s string) template.JS { return template.JS(s) },
		 "uniqid": func(name string) string {
			h := md5.New()
			io.WriteString(h,name)
			return fmt.Sprintf("%x", h.Sum(nil))
		 },
    }
	return &funcMap
}
func (t * TimeStampHTML) getTemplate() (string) {
return `{{ $WebServer := .WebServer }}<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}} : {{.StartPath}}</title>
		<style>
			body {
					background-color: #111111;
					font-family:arial verdana;
					font-size:12px;
					color:#eeeeee; 
			}
			div.container {
				width:100%
				margin-bottom: 15px;
				background-color:#eee;
				color:#111111;
			}
			div.tree {
				display:none;
			}
			div.log {
				margin-left: 20px;
			}
			div.partiallog {
				margin-left: 20px;
			}
			div.event {
				margin-left: 20px;
			}
			div.eventsrc {
				margin-left: 10px;
				display: none;
			}
			div.title {
				background-color:#238C00;
				color:white;
				font-size:16px;
				padding: 2px 2px 2px 2px;
			}
			
			a:link, a:visited, a:hover, a:active{
				color: green;
			}

		</style>
	<script type="text/javascript">
		{{rawjs .JqueryCode}}
	</script>
	</head>
	<body>
		<p>Output of {{.StartPath}}</p>
		<div class="container">
		{{range $treeindex, $tree := .All}}
			<div class="title" onclick="$('#id-tree{{ $treeindex }}').toggle()">+ {{ .Name }}</div>
			<div class="tree" id="id-tree{{ $treeindex }}" >
			Contains : {{range $logindex, $log := .Logs}} {{ if $logindex }} | {{end}}<a href="#id-log{{$logindex}}">{{.Name}}</a> {{end}}<br><br>
			{{range $logindex, $log := .Logs}}
					<a name="id-log{{$logindex}}"></a>
					<div class="log">
						{{ if $WebServer }}
							{{ .Name }} {{ .Logtype }} 
						{{ else }}
							{{ localFile .Directory .Name }} {{ .Logtype }} 
						{{ end}}
						{{range $partindex, $part := .Parts}}
						<div class="partiallog">
							{{ if .Compressed }}
								{{ if $WebServer }}
									<a href="/log?tree={{ $treeindex }}&log={{ $logindex }}&part={{ $partindex }}">{{ .Name }}</a> (File {{ .Path }} is compressed)
								{{ else }}
									{{ localFile .CompressedParentPath .Name }} (File {{ .Path }} is compressed)
								{{ end}}
							{{ else }}
								{{ if $WebServer }}
									<a href="/log?tree={{ $treeindex }}&log={{ $logindex }}&part={{ $partindex }}">{{ .Name }}</a>
								{{ else }}
									{{ localFile .Path .Name }}
								{{ end}}
								
							{{ end }}
							{{range .Events}}
							<div class="event">
								{{ .LineNumber }} : {{ .Description }}
								<div class="eventsrc">
									{{ .OriginalLine }}
								</div>
							</div>
							{{else}}
								{{ if .Firsttime }}
								<div class="event">
									{{ $part.Firsttime }}
								</div>
								{{ end }}
							{{ end }}
						</div>
						{{end}}
					</div>
				 {{end}}
			</div>
		{{else}}<div><strong>no data</strong></div>{{end}}
		</div>
	</body>
</html>`
}

func writeTimeStampHtml(logsfordump *LogCollection, filename *string) {
    f, err := os.Create((*filename))
	defer f.Close()
	errorPanic(fmt.Sprintf("Can not open output file %s",filename),err)

	jquerydata, err := Asset("jquery-2.1.4.min.js")
	
	tshtml := TimeStampHTML{
		Title:"Parsed ",
		WebServer:false,
		StartPath: logsfordump.Basepath,
		All: logsfordump.All,
		JqueryCode:string(jquerydata),
	}
	
	t, err := template.New("timestamp").Funcs(*tshtml.getFunctionMap()).Parse(tshtml.getTemplate())
	errorPanic(fmt.Sprintf("Templating did not work"),err)
	
	err = t.Execute(f, tshtml)
	errorPanic("Write failed",err)
}

type LogMutex struct {
	logs *LogCollection
	settings *Settings
	mut sync.RWMutex
}
type LogHandler struct {
	logMutex *LogMutex
}

func (l *LogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jquerydata, err := Asset("jquery-2.1.4.min.js")
	
	l.logMutex.mut.RLock()
	defer l.logMutex.mut.RUnlock()
	
	
	tshtml := TimeStampHTML{
		Title:"Parsed ",
		WebServer:true,
		StartPath: l.logMutex.logs.Basepath,
		All: l.logMutex.logs.All,
		JqueryCode:string(jquerydata),
	}
	
	t, err := template.New("webmain").Funcs(*tshtml.getFunctionMap()).Parse(tshtml.getTemplate())
	errorPanic(fmt.Sprintf("Templating did not work"),err)
	
	err = t.Execute(w, tshtml)
	errorPanic("Could not write to stream",err)
	
}
type LogViewHandler struct {
	logMutex *LogMutex
}
func (l *LogViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//jquerydata, err := Asset("jquery-2.1.4.min.js")
	
	l.logMutex.mut.RLock()
	defer l.logMutex.mut.RUnlock()
	
	convert := func(str string,def int) (int,error) {
		rint,rerr := strconv.Atoi(str)
		if rerr == nil { if rint < 0 { rint = -rint}} else { rint = def }
		return rint,rerr
	}
	treei,e1 := convert(r.FormValue("tree"),0)
	logi,e2 := convert(r.FormValue("log"),0)
	parti,e3 := convert(r.FormValue("part"),0)
	page,_ := convert(r.FormValue("page"),0)
	lines,_ := convert(r.FormValue("lines"),1000)
	
	if e1 == nil && e2 == nil && e3 == nil {

		
		all := l.logMutex.logs.All
		if len(all) > treei {
			tree := all[treei]
			if len(tree.Logs) > logi {
				log := tree.Logs[logi]
				if len(log.Parts) > parti {
					part := log.Parts[parti]
					
					
					plr,err := NewPartialLogReader(part)
					defer plr.Close()
					reader := plr.reader
					
					if  err == nil && reader != nil {
						header := `<html><head><title>Log Part %s %d %d</title>
							<style>
								a:link, a:visited, a:hover, a:active{
									color: #99ff99;
								}
								body {
									background-color: #111111;
									font-family:arial verdana;
									font-size:12px;
									color:#eeeeee; 
								}
								.stats {
									font-family:arial verdana;
									font-size:20px;
									background-color:#238C00;
									color:white;
								}
								.top {
									font-size:16px;
								}
								.ib {
									display:inline-block;
									margin-right: 5px;
								}
								.numblock {
									width : 80px;
									color:#eeeeee;
									background-color: #222222;
								}
								.timeblock {
									width : 200px;
									color:#eeeeee;
									background-color: #222222;
								}
								.line {
									color:#eeeeee;
									background-color: #111111;
								} 
								.red {
									color:black;
									background-color: red;
								}
								.orange {
									color:black;
									background-color: orange;
								}
							</style>
							</head><body>`
						fmt.Fprintf(w,header,part.Path,page,lines)
						
						npart := ""
						if parti > 0 {
							spart := log.Parts[parti-1]
							npart += fmt.Sprintf(" <a href='/log?tree=%d&log=%d&part=%d&page=%d&lines=%d'>Previous Part (%d)</a> ",treei,logi,(parti-1),0,lines,spart.Seq)
						} 
						if (parti+1) < len(log.Parts) {
							spart := log.Parts[parti+1]
							npart += fmt.Sprintf(" <a href='/log?tree=%d&log=%d&part=%d&page=%d&lines=%d'>Next Part (%d)</a> ",treei,logi,(parti+1),0,lines,spart.Seq)
						}
						
						fmt.Fprintf(w,"<div class='top'>Log Part %s %d %d %s</div>",part.Path,page,lines,npart)
						fmt.Fprintf(w,"<div><a href='/'>...return to main</a></div><br><br>")
						scandone := false
						
						ln := 0
						
						start := lines*page
						stop := lines*(page+1)
						
						
						
						timestamp := ""
						firststamp := ""
						interesting := 0
						var b bytes.Buffer
						
						for str,errread := reader.ReadString('\n');(!scandone && (errread == nil || errread == io.EOF));str,errread = reader.ReadString('\n') {
							ln++
							
							if(ln >= start && ln < stop) {
								if strarrg := jobtimeline.FindStringSubmatch(str);len(strarrg) > 1 {
										t,v :=  logtimeToEpoch(&strarrg[1],(*(l.logMutex.settings)).utchours,(*(l.logMutex.settings)).utcminutes)
										if(t != -1) {
											timestamp = v.Format(time.RFC3339)
											color := ""
											if strarrg[3] == "Error" {
											 color = "red"
											 interesting++
											} else if strarrg[3] == "Normal" {
											 color = "orange"
											 interesting++
											}
											if(firststamp == "") { firststamp = timestamp}
											
											fmt.Fprintf(&b,"<div><div class='ib numblock'>%d</div><div class='ib timeblock'>%s&nbsp;</div><div class='ib line %s'>%s</div></div>",ln,timestamp,color,strarrg[4])
										} else {
											fmt.Fprintf(&b,"<div><div class='ib numblock'>%d</div><div class='ib timeblock'>%s&nbsp;</div><div class='ib line'>%s</div></div>",ln,timestamp,str)
										}
								} else {
									fmt.Fprintf(&b,"<div><div class='ib numblock'>%d</div><div class='ib timeblock'>%s&nbsp;</div><div class='ib line'>%s</div></div>",ln,timestamp,str)
								}
							}
							if (errread == io.EOF) {
								scandone = true
							}
						}
						
						
						
						
						
						np := ""
						if(page > 0) {
							np += fmt.Sprintf("<div><a href='/log?tree=%d&log=%d&part=%d&page=%d&lines=%d'>Previous</a>  ",treei,logi,parti,(page-1),lines)
						} 
						np += fmt.Sprintf("- Current Page %d -",page)
						if (((page+1)*lines) <= ln) { 
							np += fmt.Sprintf("<a href='/log?tree=%d&log=%d&part=%d&page=%d&lines=%d'>Next</a></div>",treei,logi,parti,(page+1),lines)
						}
						
						
						fmt.Fprintf(w,np)
						fmt.Fprintf(w,"<div>")
						for p:=0;(p*lines)<ln;p++  {
						    if p == page {
								fmt.Fprintf(w,"%d, ",p)
							} else {
								fmt.Fprintf(w,"<a href='/log?tree=%d&log=%d&part=%d&page=%d&lines=%d'>%d</a>, ",treei,logi,parti,p,lines,p)
							}
						}
						fmt.Fprintf(w,"</div>")
						
						fmt.Fprintf(w,"<div class='stats'>Start: %s Stop: %s Interesting: %d</div>",firststamp,timestamp,interesting)
						
						b.WriteTo(w)
						fmt.Fprintf(w,"<div class='line'>Total of %d lines in log</div></body></html>",ln)
						fmt.Fprintf(w,np)
					} else {
						fmt.Fprintf(w,"<html><head><title>Error</title></head><body>Problem opening %s</body></html>",part.Path)
					}
					
					
					
					
					
				} else { fmt.Fprintf(w,"<html><head><title>Error</title></head><body>part out of range</body></html>")}
			} else { fmt.Fprintf(w,"<html><head><title>Error</title></head><body>log out of range</body></html>")}
		} else { fmt.Fprintf(w,"<html><head><title>Error</title></head><body>tree out of range</body</html>>") }
	} else {
		fmt.Fprintf(w,"<html><head><title>Error</title></head><body>malformed request, make sure tree,log & part are integers</body></html>")
	}
}

func webServer(logs *LogCollection,settings *Settings) {
	logMutex := LogMutex{logs,settings,sync.RWMutex{}}
	
	logHandler := LogHandler{logMutex:&logMutex}
	logViewHandler := LogViewHandler{logMutex:&logMutex}
	
	http.Handle("/log", &logViewHandler)
	http.Handle("/", &logHandler)
	
	fmt.Printf("Starting server on http://localhost:14486")
	err := http.ListenAndServe(":14486", nil)
	errorPanic("Could not bind to port 14486",err)
}
