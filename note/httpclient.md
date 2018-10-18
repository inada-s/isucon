go http client
---
# 参考
- http://qiita.com/ono_matope/items/60e96c01b43c64ed1d18

# 感想
- 基本的に DefaultClient を使う (http.Getやhttp.Postなどは中でこれを使う)
- 特殊な設定したclientを持ちたい場合や複数用意したい場合はhttp.Clientを作る
- よく使うのは client.Get, client.PostForm
- `err != nil` の場合は レスポンスは必ず最後まで読み, Closeを呼び出すこと

# サンプルコード
```go
client := &http.Client{
  Jar: jar,
  Timeout: 5 * time.Second,
}

// GET
resp, err := client.Get(baseURL + "/foo")[
defer resp.Body.Close()

// PostForm ("application/x-www-form-urlencoded")
data := url.Values{"name": {UserID}, "password": {Pass}}
resp, err := client.PostForm(URL+"/login", data)
defer resp.Body.Close()

// Post
data := url.Values{"name": {UserID}, "password": {Pass}}
resp, err := client.Post("http://example.com/upload", "image/jpeg", &buf)
defer resp.Body.Close()
```

