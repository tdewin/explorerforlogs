package main

import ( 
	"bufio"
	"io"
	"archive/zip"
	"compress/gzip"
	"os"
	"fmt"
)

type PartialLogReader struct {
	reader *bufio.Reader
	file *os.File
	gzipReader *gzip.Reader
	zipReader *zip.ReadCloser
	zipFileReader *io.ReadCloser
}
func (plr PartialLogReader) Close() {
	if plr.zipFileReader != nil {
		defer (*(plr.zipFileReader)).Close()
		//fmt.Println("closing zip file")
	}
	if plr.zipReader != nil {
		defer (*(plr.zipReader)).Close()
		//fmt.Println("closing zip")
	}
	if plr.gzipReader != nil {
		defer (*(plr.gzipReader)).Close()
		//fmt.Println("closing gzip")
	}
	if plr.file != nil {
		defer (*(plr.file)).Close()
		//fmt.Println("closing file")
	}
}
func NewPartialLogReader(part *PartialLog) (*PartialLogReader,error) {
			plr := PartialLogReader{}
			var err error
			
			if part.Compressed && part.CompressedType == "gz" {
				f, errf := os.Open(part.CompressedParentPath)
				errorSoft(fmt.Sprintf("Could not open %s",part.CompressedParentPath),errf)
				plr.file = f
				
				if (errf == nil) {
					gz, errg := gzip.NewReader(f)
					errorSoft(fmt.Sprintf("Could not open %s",part.CompressedParentPath),errg)
					plr.gzipReader = gz
					
					if(errg == nil) {
						plr.reader = bufio.NewReader(gz)
					} else {
						err = errg
					}
				} else {
					err = errf
				}
				
			} else if part.Compressed && part.CompressedType == "zip" {
				f, errf := zip.OpenReader(part.CompressedParentPath)
				errorSoft(fmt.Sprintf("Could not open %s",part.CompressedParentPath),errf)
				plr.zipReader = f
				
				if (errf == nil) {	
					for x := 0;x < len(f.File);x++ {
						zf := f.File[x]
						if part.Path == zf.Name {
							zfr, errz := zf.Open()
							errorSoft(fmt.Sprintf("Could not open %s",part.CompressedParentPath),errz)
							plr.zipFileReader = &zfr
							
							if(errz == nil) {
								plr.reader = bufio.NewReader(zfr)
							} else {
								err = errz
							}
						}
					}
				} else {
					err = errf
				}
			}else {
				f, errf := os.Open(part.Path)
				errorSoft(fmt.Sprintf("Could not open %s",part.Path),errf)
				plr.file = f
				
				if (errf == nil) {
					plr.reader = bufio.NewReader(f)
				} else {
					err = errf
				}
			}
			return &plr,err
}
