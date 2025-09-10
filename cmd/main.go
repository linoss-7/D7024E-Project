package main

import (
	"github.com/linoss-7/D7024E-Project/internal/cli"
	"github.com/linoss-7/D7024E-Project/pkg/build"
)

var (
	BuildVersion string = ""
	BuildTime    string = ""
)

func main() {
	build.BuildVersion = BuildVersion
	build.BuildTime = BuildTime
	cli.Execute()
}
