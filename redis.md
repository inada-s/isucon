# 参考
- redisドキュメント http://redis.shibu.jp/index.html
- redigoリポジトリ https://github.com/garyburd/redigo
- redigoドキュメント https://godoc.org/github.com/garyburd/redigo/redis

# 感想
永続化もできて, 複数のサーバからアトミックに操作できるメモリストレージとして優秀, さらにpubsub使えばサーバ間で通知が送りあえる.  
goの任意の型をredisに乗っけるためには, gob等を使ってシリアライズ/デシリアライズするコードを書かなければならないのがネック.

pubsubはgorpcで代用できる.
永続化はgobエンコードしてファイルに落とすのとたいしてコスト変わらない, むしろ初期データに復元するなどがredisだとめんどくさそう.

redisのこのデータ構造にピッタリハマる, というような問題が出たときは使うかもしれない.

# redigoメモ

`c.Do()`はブロッキングで処理を実行, 戻り値で結果を返す.  
`c.Send()`をn回呼び出し, `c.Flush()`で一度にコマンドを送る事ができ, `c.Receive()`をn回呼んで結果を受け取る.

```go
c.Send("SET", "foo", "bar")
c.Send("GET", "foo")
c.Flush()
c.Receive() // reply from SET
v, err = c.Receive() // reply from GET
```

Send("MULTI")-Do("Exec")でトランザクションができる
```go
c.Send("MULTI")
c.Send("INCR", "foo")
c.Send("INCR", "bar")
r, err := c.Do("EXEC")
fmt.Println(r) // prints [1, 1]
```

# 永続化
.rdbファイルと.aofファイルの2種類の形式がある.

.aofは書き込みコマンドのダンプのようなもの.  
.rdbはバイナリファイルで.aofより軽い

本来は.rdbに定期的にフルバックアップし, ダウン時の為に.aofにもこまめに書き出しておくような使い方をするらしい.  
ただし再読込したい場合は基本的にredisを落として再起動いなければならない.

isuconでは/initializeが呼ばれると初期データにリストアしたいが, redis-serverは再起動したくない.  
そのためのやりかたとして,  
1. .aofファイルをまっさらにした状態でredis-serverを起動し, 初期データをredisに書き込む.  
2. すると初期データに等しい.aofファイルができあがる. この.aofファイルを保存しておく.  
3. /initializeが呼ばれ初期データをリストアしたくなったときは, flushall コマンドで初期化後, パイプで.aofファイルをredis-cliに流し込む.


# サンプルコード
```go
package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	redisPool = newPool(":6379")
)

// note: exported なフィールドしかgobはシリアライズできない
type User struct {
	Name    string
	Address string
	Age     int
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", server)
		},
	}
}

func waitUntilRedisEstablish() {
	for {
		c := redisPool.Get() // note: 毎回新しいconnで試さないとだめ
		_, err := c.Do("PING")
		c.Close()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second)
	}
}

func generateInitialData() {
	// 事前にappendonly.aofを削除後, redis-serverを再起動した直後に実行すること

	// データベース等から初期データを生成する
	// appendonlyファイルに出力する
	// ファイル名はredis.confで設定する プログラム側からは指定できない.
	// 特に変える意味もないのでappendonly.aofで良いと思う.

	c := redisPool.Get()
	defer c.Close()

	log.Println(redis.String(c.Do("config", "set", "appendonly", "yes")))
	log.Println(redis.String(c.Do("config", "set", "appendfsync", "everysec")))

	// DB等から初期データを読み込み, redisに書き込むことで, appendonly.aofができあがる.
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(User{"Asan", "1-1-1", 20})
	enc.Encode(User{"Bsan", "2-2-2", 30})
	enc.Encode(User{"Csan", "3-3-3", 40})
	c.Do("SET", "FOO", buf.String())

	os.Exit(0)
}

func restoreInitialData() {
	c := redisPool.Get()
	defer c.Close()

	// 復元はスクリプトに書いておいて, それをgoからExecするのが良さそう.
	/*
		#restore_redis.sh
		"/usr/bin/redis-cli flushall"
		"/usr/bin/redis-cli --pipe < /home/isucon/appendonly.aof"
	*/

	log.Println(redis.String(c.Do("config", "set", "appendonly", "no")))
	out, err := exec.Command("/bin/bash", "restore_redis.sh").Output()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(out))
	time.Sleep(time.Second)
}

func main() {
	waitUntilRedisEstablish()

	//generateInitialData()
	restoreInitialData()

	c := redisPool.Get()
	defer c.Close()

	log.Print("initial data:")
	log.Println(redis.String(c.Do("GET", "FOO")))

	log.Println(c.Send("set", "FOOO", "BAR"))
	log.Println(c.Send("get", "FOOO"))
	c.Flush()
	log.Println(c.Receive())
	log.Println(redis.String(c.Receive()))

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(User{"fooman", "1-1-1", 20})
	if err != nil {
		log.Fatalln(err)
	}
	c.Do("SET", "FOO", buf.String())

	bin, err := redis.String(c.Do("GET", "FOO"))
	log.Println(bin)
	if err != nil {
		log.Fatalln(err)
	}
	dec := gob.NewDecoder(strings.NewReader(bin))
	var foo User
	dec.Decode(&foo) // note: アドレスを渡さないとだめ
	log.Println(foo)

	log.Println(c.Do("BGSAVE"))
	log.Println(c.Do("LASTSAVE"))

	log.Println(redis.Strings(c.Do("config", "get", "appendonly")))

	log.Println("end")
}
```
