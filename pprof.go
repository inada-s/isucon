package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

var (
	enableProfile    = true
	isProfiling      = false
	cpuProfileFile   = "/tmp/cpu.pprof"
	memProfileFile   = "/tmp/mem.pprof"
	blockProfileFile = "/tmp/block.pprof"
)

func StartProfile(duration time.Duration) error {
	f, err := os.Create(cpuProfileFile)
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	runtime.SetBlockProfileRate(1)
	isProfiling = true
	if 0 < duration.Seconds() {
		go func() {
			time.Sleep(duration)
			err := EndProfile()
			if err != nil {
				log.Println(err)
			}
		}()
	}
	log.Println("Profile start")
	return nil
}

func EndProfile() error {
	if !isProfiling {
		return nil
	}
	isProfiling = false
	pprof.StopCPUProfile()
	runtime.SetBlockProfileRate(0)
	log.Println("Profile end")

	mf, err := os.Create(memProfileFile)
	if err != nil {
		return err
	}
	pprof.WriteHeapProfile(mf)

	bf, err := os.Create(blockProfileFile)
	if err != nil {
		return err
	}
	pprof.Lookup("block").WriteTo(bf, 0)
	return nil
}

func init() {
	log.Println("add pprof start end func")
	http.HandleFunc("/startprof", func(w http.ResponseWriter, r *http.Request) {
		err := StartProfile(time.Minute)
		if err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte("profile started\n"))
		}
	})

	http.HandleFunc("/endprof", func(w http.ResponseWriter, r *http.Request) {
		err := EndProfile()
		w.Write([]byte("profile ended\n"))
		if err != nil {
			w.Write([]byte(err.Error() + "\n"))
		}
	})
}
