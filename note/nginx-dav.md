nginx dav
-----

# 参考
- http://mogile.web.fc2.com/nginx/http/ngx_http_dav_module.html
- http://server-setting.info/centos/nginx-webdav-howto.html

# 対応しているか確認
` nginx -V`で`--with-http_dav_module`が入っていればok
拡張も使いたい場合は`nginx-dav-ext-module`も入っているか確認

```
$nginx -V
nginx version: nginx/1.6.2
TLS SNI support enabled
configure arguments: --with-cc-opt='-g -O2 -fstack-protector-strong -Wformat -Werror=format-security -D_FORTIFY_SOURCE=2' --with-ld-opt=-Wl,-z,relro --prefix=/usr/share/nginx --conf-path=/etc/nginx/nginx.conf --http-log-path=/var/log/nginx/access.log --error-log-path=/var/log/nginx/error.log --lock-path=/var/lock/nginx.lock --pid-path=/run/nginx.pid --http-client-body-temp-path=/var/lib/nginx/body --http-fastcgi-temp-path=/var/lib/nginx/fastcgi --http-proxy-temp-path=/var/lib/nginx/proxy --http-scgi-temp-path=/var/lib/nginx/scgi --http-uwsgi-temp-path=/var/lib/nginx/uwsgi --with-debug --with-pcre-jit --with-ipv6 --with-http_ssl_module --with-http_stub_status_module --with-http_realip_module --with-http_auth_request_module --with-http_addition_module --with-http_dav_module --with-http_geoip_module --with-http_gzip_static_module --with-http_image_filter_module --with-http_spdy_module --with-http_sub_module --with-http_xslt_module --with-mail --with-mail_ssl_module --add-module=/build/nginx-T5fW9e/nginx-1.6.2/debian/modules/nginx-auth-pam --add-module=/build/nginx-T5fW9e/nginx-1.6.2/debian/modules/nginx-dav-ext-module --add-module=/build/nginx-T5fW9e/nginx-1.6.2/debian/modules/nginx-echo --add-module=/build/nginx-T5fW9e/nginx-1.6.2/debian/modules/nginx-upstream-fair --add-module=/build/nginx-T5fW9e/nginx-1.6.2/debian/modules/ngx_http_substitutions_filter_module
```
ubuntu で `apt-get install nginx` したら対応しているnginxが入るはず.
つよいnginx入れたいなら `apt-get install nginx-extras`.

# 設定
webdav使いたいlocationだけ別に設定すると良さそう.
```
  location /webdav/ {
    root /var/www;
    dav_methods PUT DELETE MKCOL COPY MOVE;
    dav_access user:rw group:rw all:rw;
    create_full_put_path on;
    client_body_temp_path /tmp/nginx/webdav;
  } 
```
- `/var/www/webdav` 以下がwebdavで参照される.
- `/tmp/nginx/webdav` に一時ファイルを置く.

パーミッションの問題がにハマりたくないなら浅いディレクトリにしておき, 必要なディレクトリは作っておくとよい.
各ディレクトリがwww-dataさんが書き込みできるか確認しておく.


# GETの確認
webdavディレクトリに適当なファイルを設置してブラウザで開く
エラーが出る場合はログを見て修正

# PUTの確認
```
$echo "testfile" > test
$curl http://52.198.202.223/webdav/test --upload-file test
$curl http://52.198.202.223/webdav/test
```

# DELETEの確認
```
$curl http://52.198.202.223/webdav/test
$curl http://52.198.202.223/webdav/test -X "DELETE"
```

# GOからリクエスト
hoge.go
```go
package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func Put(url string, r io.Reader) {
	req, err := http.NewRequest("PUT", url, r)
	must(err)
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	must(err)
	log.Println("PUT ok:", string(body))
}

func Get(url string) {
	resp, err := http.Get(url)
	must(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	must(err)
	log.Println("GET ok:", string(body))
}

func Delete(url string) {
	req, err := http.NewRequest("DELETE", url, nil)
	must(err)
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	must(err)
	log.Println("DELETE ok:", string(body))
}

func main() {
	url := "http://52.198.202.223/webdav/testgo"

	f, err := os.Open("hoge.go")
	must(err)
	Put(url, f)
	Get(url)
	Delete(url)
	Get(url)
}
```
