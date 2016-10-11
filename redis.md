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

# サンプルコード
```go
package main

import (
	"bytes"
	"encoding/gob"
	"log"
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

func main() {
	waitUntilRedisEstablish()

	c := redisPool.Get()
	defer c.Close()
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

	log.Println("end")
}
```
