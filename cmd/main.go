package main

import (
	"github.com/ncyborgse/go-template/internal/cli"
	"github.com/ncyborgse/go-template/pkg/build"
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
