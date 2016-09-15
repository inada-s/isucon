# nginx
## 設定雛形
```
# 動作ユーザー／グループの設定
# user www-data;

# ワーカープロセス数
worker_processes auto;
#worker_processes 1;

# pid
pid /run/nginx.pid;

# ワーカーの扱う最大ファイルディスクリプター数.
# OSレベルで扱えるディスクリプター総数である、/proc/sys/fs/file-maxを参考に設定する.
worker_rlimit_nofile 65536;

events {
  worker_connections  65536;
}

http {
  log_format  isucon '$time_local $msec\t$status\treqtime:$request_time\t'
                       'in:$request_length\tout:$bytes_sent\trequest:$request\t'
                       'acceptencoding:$http_accept_encoding\treferer:$http_referer\t'
                       'ua:$http_user_agent';
  access_log  /var/log/nginx/access.log isucon;
  # access_log  /var/log/nginx/access.log off;

  # カーネル空間内で静的ファイルの読み込みと送信ができるようになる.
  sendfile on;

  # 0以外の値が設定された場合、一つのsendfile() コールで転送することができるデータの量を制限します。
  # 制限が無い場合、一つの高速な接続がworkerプロセスを完全に掴むかもしれません。
  sendfile_max_chunk 0;

  # TCP_CORK TCP_NOPSUH を使用する. 
  tcp_nopush on;
  # TCP_NODELAY を使用する. tcp_nopush onの場合offでいいと思う.
  # tcp_nodelay on;

  # Connection: keep-alive がレスポンスヘッダに付加される. 
  # 数値はサーバ側でTCPコネクションを開きっぱなしにする時間(sec).
  keepalive_timeout 60;
  # 同一keep-alive connection でサービスするリクエストの最大値.
  keepalive_requests 100000;

  # コンテンツをgzip圧縮する. CPUと帯域と相談して on/offをきめる事.
  gzip on;
  # Vary: Accept-Encoding をレスポンスヘッダに付加する.
  gzip_vary on;
  # デフォルトだと nginxは text/html しかgzip圧縮しないので明示的に追加する.
  gzip_types text/css text/javascript;
  # Viaヘッダ(前段にキャッシュサーバやCDNが存在する場合に付く)が付いていてもgzip圧縮する
  gzip_proxied any;
  
  upstream app {
    server 127.0.0.1:8080;
  }

  server {
    location ~* \.(css|js)$ {
      gzip_static always;
      gunzip on;
    }

    location / {
      proxy_set_header Host $host;
      proxy_pass http://app;
    }
  }
}
```

# mysql

## isucon向け設定
```
skip-grant-tables
#innodb_read_io_threads=4
#innodb_write_io_threads=8  #To stress the double write buffer
#innodb_buffer_pool_size=20G
#innodb_buffer_pool_load_at_startup=ON
#innodb_log_file_size = 32M #Small log files, more page flush
#innodb_log_files_in_group=2
#innodb_file_per_table=1
#innodb_log_buffer_size=8M
innodb_flush_method=O_DIRECT
innodb_flush_log_at_trx_commit=0
skip-innodb_doublewrite
```

## インデックス
追加
```
ALTER TABLE posts ADD INDEX created_at_idx(created_at);
```

複合インデックス
```
ALTER TABLE posts ADD INDEX index_posts_on_created_at_updated_at(created_at, updated_at);
```

削除
```
ALTER TABLE posts DROP INDEX created_at_idx;
```


# golang
## コンフリクトしにくい編集
同じパッケージ内に新しくファイルを作って実装する.  
そこで作った関数をメインのgoファイルで呼び出す形にする.

## データのオンメモリ化
データの使い方を良く見て考えてデータ構造を作る.

頻出として map[PK]Value がある.  
mutexとmapで持って, のGet, Set, Updateの関数を提供し直接mapを触らない形がハマらないで済みそう.  
ポインタを使うとメモリやコピーコストを節約できるかもしれないけど, ハマらないで済むので基本オブジェクトをそのまま持つ.  
DBを操作する部分を調べて関数を差しこむなり入れ替えるなり.

```
var (
	userRepoM sync.Mutex
	userRepo  map[int]User
)

func userAdd(u User) {
	userRepoM.Lock()
	userRepo[u.ID] = u
	userRepoM.Unlock()
}

func userGet(uid int) User {
	userRepoM.Lock()
	u := userRepo[uid]
	userRepoM.Unlock()
	return u
}

func userBan(uid, ban int) {
	userRepoM.Lock()
	u := userRepo[uid]
	u.DelFlg = ban
	userRepo[uid] = u
	userRepoM.Unlock()
}

func usersReset() {
	userRepoM.Lock()
	defer userRepoM.Unlock()

	userRepo = make(map[int]User)
	users := []User{}
	err := db.Select(&users, "SELECT * FROM users")
	if err != nil {
		panic(err)
	}
	for _, u := range users {
		userRepo[u.ID] = u
	}
}
```

## セッションストアのオンメモリ化
"github.com/gorilla/sessions" のオンメモリバージョン  
ちょっと書き換えれば他のライブラリにも対応可能.

メモリに余裕があるならユーザ情報まるまるセッションにも持っておくと良さそう
```
const sessionName = "isucon_session"

type Session struct {
	Key string

	UserId int
	User   User
	Notice string
}

func (s *Session) Save(r *http.Request, w http.ResponseWriter) {
	sessionStore.Set(w, s)
}

type SessionStore struct {
	sync.Mutex
	store map[string]*Session
}

var sessionStore = SessionStore{
	store: make(map[string]*Session),
}

func (self *SessionStore) Get(r *http.Request) *Session {
	cookie, _ := r.Cookie(sessionName)
	if cookie == nil {
		return &Session{}
	}
	key := cookie.Value
	self.Lock()
	s := self.store[key]
	self.Unlock()
	if s == nil {
		s = &Session{}
	}
	return s
}

func (self *SessionStore) Set(w http.ResponseWriter, sess *Session) {
	key := sess.Key
	if key == "" {
		b := make([]byte, 8)
		rand.Read(b)
		key = hex.EncodeToString(b)
		sess.Key = key
	}

	cookie := sessions.NewCookie(sessionName, key, &sessions.Options{})
	http.SetCookie(w, cookie)

	self.Lock()
	self.store[key] = sess
	self.Unlock()
}
```


## レンダリング結果のキャッシュ
例えば非ログインユーザの場合同じ結果を返すリクエストの場合, レンダリング結果をキャッシュしておくことができる.  
他にもコンテンツ1つ1つに対してレンダリング結果を保持しておき, レンダリング回数を減らすことができるケースがある.  

また, isuconのレギュレーションではしばしば, 更新系リクエストの結果が他のGETに反映されるまでに1秒程度猶予がある.  
この場合対象となるGETリクエストでは, 余裕持って0.5秒に1回レンダリングすればよく, ベンチマーク1分だとすると120回しかレンダリングしなくて済む.

1秒猶予のレギュレーションが無い場合, 毎回レンダリングすることになるが, 更新系リクエストのレスポンスを返す前にレンダリングしてしまえばいい.

```
var (
	indexRenderedMtx sync.Mutex
	indexRendered    []byte
)

func updateIndexRendered() {
	var b bytes.Buffer
	indexTemplate.Execute(&b, ...)
}

...
// 0.5秒毎に更新する場合
func updateIndexRenderedLoop() {
	for {
		time.Sleep(500 * time.Millisecond)
		updateIndexRendered()
	}
}
```


## テンプレートエンジン撲滅
DB周りの基本的なチューニングが済んでくるとたいてい, テンプレートエンジンのレンダリング時間が占める割合が増えてくる.  
便利スクリプトを用意しておき, 機械的に置換してく.

go run tpl_conv.go < index.html > index.gotpl

あとは手修正でがんばる.



