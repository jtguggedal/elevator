
package main

import "flag"

// note, that variables are pointers
var strFlag = flag.String("long-string", "", "Description")
var boolFlag = flag.Bool("bool", false, "Description of flag")

func init() {
    // example with short version for long flag
    flag.StringVar(strFlag, "s", "", "Description")
}

func main() {
    flag.Parse()
    println(*strFlag, *boolFlag)
}
