# http2
## 感想
- コネクション開きっぱなしで複数httpストリームを持てる感じ.
- リクエスト毎にコネクションを閉じないで何度も平行するのでTCP的にも速度でる.
- サーバから事前にデータをpushしてくるなど未来的な仕様がある
- TLS前提なので, isuconではhttpsな外部サーバにリクエストを投げるような感じの構成の場合にしか使え無さそう
- ベンチマーカーがhttp2喋ってきたらどうしよう..

## http2対応かどうか確認する
h2iというコマンドがある. https://godoc.org/golang.org/x/net/http2/h2i
```sh
$h2i example.com
```
これでエラーが出ず接続できたらhttp2で喋れる.

リクエストを送ってみるサンプル
```sh
$ h2i google.com
Connecting to google.com:443 ...
Connected to 172.217.25.110:443
Negotiated protocol "h2"
Sending: []
           [FrameHeader SETTINGS len=24]
                                          [MAX_CONCURRENT_STREAMS = 100]
                                                                          [INITIAL_WINDOW_SIZE = 1048576]
                                                                                                           [MAX_FRAME_SIZE = 16384]
                                                                                                                                     [MAX_HEADER_LIST_SIZE = 16384]
                                                                                                                                                                   [FrameHeader WINDOW_UPDATE len=4]
                                                                                                                                                                                                      Window-Increment = 983041

                                                                                                                                                                                                                               [FrameHeader SETTINGS flags=ACK len=0]
                       h2i> ping foooooooooooo
[FrameHeader PING flags=ACK len=8]
                                    Data = "fooooooo"
                                                     h2i> 
h2i> 
h2i> header
(as HTTP/1.1)> GET https://google.com HTTP/1.1
(as HTTP/1.1)> 
Opening Stream-ID 1:
                     :authority = google.com
                                             :method = GET
                                                           :path = /
                                                                     :scheme = https
                                                                                    [FrameHeader HEADERS flags=END_HEADERS stream=1 len=156]
                                                                                                                                              :status = "302"
                                                                                                                                                               cache-control = "private"
                                                                                                                                                                                          content-type = "text/html; charset=UTF-8"
                                                                                                                                                                                                                                     location = "https://www.google.co.jp/?gfe_rd=cr&ei=Tgv-V4atHerd8AfXkrioDg"
                                                                   content-length = "262"
                                                                                           date = "Wed, 12 Oct 2016 10:07:10 GMT"
                                                                                                                                   alt-svc = "quic=\":443\"; ma=2592000; v=\"36,35,34,33,32\""
                                                                                                                                                                                              [FrameHeader DATA flags=END_STREAM stream=1 len=262]
      "<HTML><HEAD><meta http-equiv=\"content-type\" content=\"text/html;charset=utf-8\">\n<TITLE>302 Moved</TITLE></HEAD><BODY>\n<H1>302 Moved</H1>\nThe document has moved\n<A HREF=\"https://www.google.co.jp/?gfe_rd=cr&amp;ei=Tgv-V4atHerd8AfXkrioDg\">here</A>.\r\n</BODY></HTML>\r\n"
                                              [FrameHeader PING len=8]
                                                                        Data = "\x00\x00\x00\x00\x00\x00\x00\x00"
```

## goでhttp2クライアント
- http.Client.Transportをhttp2.Transportに置き換えたhttpクライアントを用意してあげればいい
- オレオレ証明書などで検証する場合は InsecureSkipVerify を true にしてあげる
```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

func main() {
    client := http.Client{
        Transport: &http2.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },  
    }
	resp, err := client.Get("https://google.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("HEADER:", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	_ = body
	//fmt.Println(string(body))
}
```
