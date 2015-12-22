package main

import (
 "os"
 "fmt"
 "html/template"
 "crypto/md5"
 "io"
)

type TimeStampHTML struct {
	Title string
	JqueryCode string
	StartPath string
	All []*LogTree 
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
return `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}} : {{.StartPath}}</title>
		<style>
			body {
				font-family: Arial, Verdana;
				font-size:12px;
			}
			div.container {
				width:100%
				margin-bottom: 15px;
				background-color:#eee;
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
				color: #111;
			}

		</style>
	<script type="text/javascript">
		{{rawjs .JqueryCode}}
	</script>
	</head>
	<body>
		<p>Output of {{.StartPath}}</p>
		<div class="container">
		{{range .All}}
			<div class="title" onclick="$('#id-{{ uniqid .Name }}').toggle()">+ {{ .Name }}</div>
			<div class="tree" id="id-{{ uniqid .Name }}" >
			Contains : {{range $index, $value := .Logs}} {{ if $index }} | {{end}}<a href="#id-{{uniqid .Path}}">{{.Name}}</a> {{end}}<br><br>
			{{range .Logs}}
					<a name="id-{{uniqid .Path}}"></a>
					<div class="log">
						{{ localFile .Path .Name }} {{ .Logtype }}</a> 
						{{range .Parts}}
						<div class="partiallog">
							{{ if .Compressed }}
								{{ localFile .CompressedParentPath .Name }} (File {{ .Path }} is compressed)
							{{ else }}
								{{ localFile .Path .Name }}
							{{ end }}
							{{range .Events}}
							<div class="event">
								{{ .LineNumber }} : {{ .Description }}
								<div class="eventsrc">
									{{ .OriginalLine }}
								</div>
							</div>
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
		StartPath: logsfordump.Basepath,
		All: logsfordump.All,
		JqueryCode:string(jquerydata),
	}
	
	t, err := template.New("timestamp").Funcs(*tshtml.getFunctionMap()).Parse(tshtml.getTemplate())
	errorPanic(fmt.Sprintf("Templating did not work"),err)
	
	err = t.Execute(f, tshtml)
	errorPanic("Write failed",err)
}
