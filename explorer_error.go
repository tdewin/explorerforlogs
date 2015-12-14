package main

import (
	"fmt"
)

//generic
func errorPanic(msg string,e error) {
	if e != nil {
		fmt.Printf(msg)
		panic(e)
	}
}
func errorSoft(msg string,e error) {
	if e != nil {
		fmt.Printf("%s\n%s",msg,e)
	}
}