package main

import (
 "os"
 "fmt"
 //"html/template"
)



func writeTimeStampHtml(logsfordump *LogCollection, filename *string) {
	htmltemplate := `
<html>
	<head><title>{{.Title}}</title></head>
	<body>
		
	</body>
</html>
	`

    f, err := os.Create((*filename))
	errorPanic(fmt.Sprintf("Can not open output file %s",filename),err)

	f.Write([]byte(htmltemplate))

    defer f.Close()
}
