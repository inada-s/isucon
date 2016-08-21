pprofのメモ
-----------------------------
pprofには色々使い方あるけど、基本的に使いそうなパターンだけメモしておく

## http.pprof
https://golang.org/pkg/net/http/pprof/

### 使用時のテンプレ
```go
import (
  _ "net/http/pprof"
)
// main.mainなどで
go func() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```sh
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof http://localhost:6060/debug/pprof/block
```

## runtime.pprof
https://golang.org/pkg/runtime/pprof/

http.pprofだとcpuプロファイル30秒ブロックしてしまう  
秒数指定できるけど、今から開始したい、今終了したい、といった使い方には向いていない.  
runtime.pprofのほうが柔軟に対応できる

`/startprofile`でプロファイル開始、`/endprofile`でプロファイル終了をする例.  
isuconのベンチマークスクリプトを間に挟めば一回のベンチマーク分を綺麗に引きぬくことができる.

### 使用時のテンプレ
```go
import (
  "os"
  "runtime/pprof"
  "runtime"
)

var (
  cpuProfileFile = "/tmp/cpu.pprof"
  memProfileFile = "/tmp/mem.pprof"
  blockProfileFile = "/tmp/block.pprof"
)
// main.mainなどで
go func() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}()

http.HandleFunc("/startprofile", func(w http.ResponseWriter, r *http.Request) {
        f, err := os.Create(cpuProfileFile)
        if err != nil {
                w.Write([]byte(err.Error()))
                return
        }
        if err := pprof.StartCPUProfile(f); err != nil {
                w.Write([]byte(err.Error()))
                return
        }
        runtime.SetBlockProfileRate(1) // どれくらい遅くなるか確認する
        w.Write([]byte("profile started\n")) 
})

http.HandleFunc("/stopprofile", func(w http.ResponseWriter, r *http.Request) {
        pprof.StopCPUProfile()
        runtime.SetBlockProfileRate(0)
        w.Write([]byte("profile ended\n"))
        
        mf, err := os.Create(memProfileFile)
        if err != nil {
                w.Write([]byte(err.Error()))
                return
        }
        pprof.WriteHeapProfile(mf)
        
        bf, err := os.Create(blockProfileFile)
        if err != nil {
                w.Write([]byte(err.Error()))
                return
        }
        pprof.Lookup("block").WriteTo(bf, 0)
})
```

```sh
go tool pprof /tmp/mem.pprof
go tool pprof /tmp/cpu.pprof
go tool pprof /tmp/block.pprof
```
↑未確認
