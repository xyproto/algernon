//go:build trace

package engine

import (
	"flag"
	"os"
	"runtime/pprof"
	"runtime/trace"

	"github.com/felixge/fgtrace"
	log "github.com/sirupsen/logrus"
)

var (
	cpuProfileFilename *string
	memProfileFilename *string
	traceFilename      *string
	fgtraceFilename    *string
)

func init() {
	cpuProfileFilename = flag.String("cpuprofile", "", "write CPU profile to `file`")
	memProfileFilename = flag.String("memprofile", "", "write memory profile to `file`")
	traceFilename = flag.String("tracefile", "", "write trace to `file`")
	fgtraceFilename = flag.String("fgtrace", "", "write fgtrace to `file`")
}

func traceStart() {
	// Output CPU profile information, if a filename is given
	if *cpuProfileFilename != "" {
		f, err := os.Create(*cpuProfileFilename)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		log.Info("Profiling CPU usage")
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		AtShutdown(func() {
			pprof.StopCPUProfile()
			log.Info("Done profiling CPU usage")
			f.Close()
		})
	}
	// Profile memory at server shutdown, if a filename is given
	if *memProfileFilename != "" {
		AtShutdown(func() {
			f, errProfile := os.Create(*memProfileFilename)
			if errProfile != nil {
				// Fatal is okay here, since it's inside the anonymous shutdown function
				log.Fatal("could not create memory profile: ", errProfile)
			}
			defer f.Close()
			log.Info("Saving heap profile to ", *memProfileFilename)
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		})
	}
	if *traceFilename != "" {
		f, errTrace := os.Create(*traceFilename)
		if errTrace != nil {
			panic(errTrace)
		}
		go func() {
			log.Info("Tracing")
			if err := trace.Start(f); err != nil {
				panic(err)
			}
		}()
		AtShutdown(func() {
			pprof.StopCPUProfile()
			trace.Stop()
			log.Info("Done tracing")
			f.Close()
		})
	}
	if *fgtraceFilename != "" {
		AtShutdown(func() {
			fgtrace.Config{Dst: fgtrace.File(*fgtraceFilename)}.Trace().Stop()
		})
	}
}
