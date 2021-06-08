package main

import (
	"flag"
	"fmt"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/face"

	gocommon "github.com/liuhengloveyou/go-common"
)

var (
	BuildTime string
	CommitID  string
	GitTag    string

	showVer = flag.Bool("version", false, "show version")
	initSys = flag.Bool("init", false, "初始化系统.")

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	if *showVer {
		fmt.Printf("%s\t%s\t%s\n", GitTag, CommitID, BuildTime)
		os.Exit(0)
	}

	gocommon.SingleInstane(common.ServConfig.PidFile)
	rand.Seed(time.Now().UnixNano())

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {

		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
		f.Close()
	}

	switch common.ServConfig.Face {
	case "http":
		face.InitAndRunHttpApi(&common.ServConfig)
	case "grpc":
		//face.GrpcFace()
	default:
		fmt.Println("face: [http | grpc]")
	}
}
