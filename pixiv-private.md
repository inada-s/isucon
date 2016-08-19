Pixiv Private Isucon を使って練習
------------

# 2016-08-19

リポジトリ 
https://github.com/catatsuy/private-isu

とりあえずインスタンス立てる  
SSD 20GBでいいのかしら

ローカルベンチを動かせるようにする

遅いのでスコア算出式を読む
```
成功レスポンス数(GET) x 1 + 成功レスポンス数(POST) x 2 + 成功レスポンス数(画像投稿) x 5 - (サーバエラー(error)レスポンス数 x 10 + リクエスト失敗(exception)数 x 20 + 遅延POSTレスポンス数 x 100)
```

80番webサービスが見えるらしいが見えない  
セキュリティグループを変更してみえた

twitterかfacebookっぽい何か  
ログインすると画像つきのpostを投稿できる  
postにはコメントが付けられる  
ban機能があるっぽい

これ色々つかうツール群テンプレ作っとこう
vimrcも準備しよう
```
sudo apt-get update
sudo apt-get install vim tree screen unzip git 
```

isuconユーザでログインできるようにする
rootになってauthorized_keysを/home/isucon/.ssh/にコピー
```
chown isucon:isucon authorized_keys
```

とりあえずデフォルトでベンチマーカー走らせよう
う、ターミナルに色がつかない
bashrcあるけどbash_profileがなくて色つかなかったのでbash_profileを用意
bashrcも事前用意しておこう

ベンチマーカーがリポジトリにしかないっぽいので持ってくる
ビルドしたけど動かない(すぐ終了しちゃう)
ami-53ef0e32から抜いてくるか
userdataディレクトリはダウンロードするやつとマージしないとだめだった。

```
/opt/go/bin/benchmarker -t http://52.196.125.238/ -u /opt/go/src/github.com/catatsuy/private-isu/benchmarker/userdata
```

とりあえずリモートベンチが動いた
```
{"pass":true,"score":2355,"success":1989,"fail":0,"messages":[]}
```
ローカルでも動かそう
```
{"pass":true,"score":2422,"success":2035,"fail":0,"messages":[]}
```
ローカルベンチの結果

goに切り替え
```
{"pass":true,"score":4000,"success":3728,"fail":0,"messages":[]}
```
はやーい


コードの変更分が分かるようにgitに置こう

goがない？
env.shで環境変数よみこんだらいけた
```
sucon@ip-10-0-1-77:~$ which go
/home/isucon/.local/go/bin/go
```
なるほど




大事なコマンドとファイル
```
/etc/systemd/system/isu-go.service 
sudo systemctl daemon-reload
sudo systemctl restart isu-go
sudo systemctl restart mysql
```

かるくwebappを見る

initializeは競技用か

```
func dbInitialize() {
	sqls := []string{
		"DELETE FROM users WHERE id > 1000",
		"DELETE FROM posts WHERE id > 10000",
		"DELETE FROM comments WHERE id > 100000",
		"UPDATE users SET del_flg = 0",
		"UPDATE users SET del_flg = 1 WHERE id % 50 = 0",
	}

	for _, sql := range sqls {
		db.Exec(sql)
	}
}
```

このidまでの奴が最初から入っているということか
userが1000件, postsが10000件, commentsが100000件  
DBにマッピングされるオブジェクトがある。これは嬉しい


nginxのシンタックスハイライト
https://github.com/vim-scripts/nginx.vim
あとで入れるところまでやっちゃおう

## 構成の分析

アプリは8080で待ち受けてる
前段にnginxが80番でいるっぽい

ドキュメントによると
MySQL 
3306番ポートでMySQLが起動しています。初期状態では以下のユーザが設定されています。
ユーザ名: root, パスワードなし

memcached 
11211番ポートでmemcachedが起動しています。

まずはCPUメモリディスクどれがボトルネックなのか見る

```
 PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND 
  741 mysql     20   0 1282552 211604   9620 S 136.8 20.7   4:31.89 mysqld
24194 isucon    20   0  342996  88444   8844 S  43.3  8.6   0:03.65 app
24260 isucon    20   0  144984  21908   6836 S   9.7  2.1   0:00.84 benchmarker
  160 root      20   0   28872   2844   2564 S   2.0  0.3   0:01.45 systemd-journal
  420 www-data  20   0   91908   4904   3084 S   2.0  0.5   0:01.42 nginx
```

mysqlがCPU食いまくってる メモリは大丈夫そう ディスクはどうやって見ればいいんだ..後で調べよう

slowクエリとかいうのがあったな
```
mysql> show variables like 'slow%';
+---------------------+--------------------------------------+
| Variable_name       | Value                                |
+---------------------+--------------------------------------+
| slow_launch_time    | 2                                    |
| slow_query_log      | OFF                                  |
| slow_query_log_file | /var/lib/mysql/ip-10-0-1-77-slow.log |
+---------------------+--------------------------------------+
3 rows in set (0.00 sec)
```

low_query_log_fileはあるけどOFFになってる

```
mysql> show variables like 'long%';
+-----------------+-----------+
| Variable_name   | Value     |
+-----------------+-----------+
| long_query_time | 10.000000 |
+-----------------+-----------+
1 row in set (0.01 sec)
```

とりあえずONにしてこれ0.1秒とかにしよう

http://linuxserver.jp/%E3%82%B5%E3%83%BC%E3%83%90%E6%A7%8B%E7%AF%89/db/mysql/%E3%82%B9%E3%83%AD%E3%83%BC%E3%82%AF%E3%82%A8%E3%83%AA%E3%83%AD%E3%82%B0
```
log_slow_queries	スロークエリログの出力の有効/無効の選択	log_slow_queries(※パラメータなしで記載)	mysqld
long_query_time	スロークエリログの出力対象となる秒単位で指定。
※MySQL5.1からは小数点をつけると1秒未満でも設定可能	long_query_time = 1	mysqld
slow_query_log_file	スロークエリログを出力する場所	slow_query_log_file=mysql-slow.log	mysqld
log-queries-not-using-indexes	インデックスを使わない検索をスロークエリログとして出力する	log-queries-not-using-indexes(※パラメータなしで記載)	mysqld
```


なるほど、全部有効にするか

mysqlは5.5.49 新しそうだ。
[VerUP］MySQL 5.5.49（リリース日：2016/04/11）

my.confで設定するらしいぞ
これもシンタックス欲しい

あ、goのappのpathいじったからビルドしないと..
gorootが違うせいかなんかエラーでまくる。go入れなおしちゃおう
```
rm -rf /home/isucon/.local/go
```

```
wget https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.7.linux-amd64.tar.gz
.bash_profileにexport PATH=$PATH:/usr/local/go/binを追加
```
ビルドできた

```
slow_query_log_file = /var/log/mysql/mysql-slow.log
slow_query_log = 1
long_query_time = 2
log_queries_not_using_indexes
```

ベンチマーク走らせる
{"pass":true,"score":4132,"success":3880,"fail":0,"messages":[]}


slow_logみれた

SELECT COUNT(*) が沢山ひっかっかる

```
SELECT * FROM `comments` WHERE `post_id` = 11 ORDER BY `created_at` DESC LIMIT 3;
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.011549  Lock_time: 0.000011 Rows_sent: 1  Rows_examined: 100000
SET timestamp=1471601308;
SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = 904;
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.002261  Lock_time: 0.000010 Rows_sent: 13  Rows_examined: 10000
SET timestamp=1471601308;
SELECT `id` FROM `posts` WHERE `user_id` = 904;
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.014767  Lock_time: 0.000012 Rows_sent: 1  Rows_examined: 100000
SET timestamp=1471601308;
SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (11, 1009, 1139, 2495, 4021, 4755, 5000, 5150, 6465, 7092, 7977, 9611, 9678);
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.011579  Lock_time: 0.000011 Rows_sent: 1  Rows_examined: 100001
SET timestamp=1471601308;
SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = 9678;
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.015589  Lock_time: 0.000011 Rows_sent: 8  Rows_examined: 100009
SET timestamp=1471601308;
SELECT * FROM `comments` WHERE `post_id` = 9678 ORDER BY `created_at` DESC;
# User@Host: root[root] @ localhost [127.0.0.1]
```

comments:
user_id post_id created_at

posts:
user_id created_at

にインデックス張る

やばそう
```
SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` ORDER BY `created_at` DESC;
```

インデックス春
複合インデックスのほうが良いのかな。。あとで勉強する
```
ALTER TABLE `isuconp`.`comments` ADD INDEX `user_id_idx` (`user_id`);
ALTER TABLE `isuconp`.`comments` ADD INDEX `post_id_idx` (`post_id`);
ALTER TABLE `isuconp`.`comments` ADD INDEX `created_at_idx` (`created_at`);

ALTER TABLE `isuconp`.`posts` ADD INDEX `user_id_idx` (`user_id`);
ALTER TABLE `isuconp`.`posts` ADD INDEX `created_at_idx` (`created_at`);
```

slowlogバックアップしてsudo systemctl restart mysql

```
 PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND 
26966 isucon    20   0  576412 241192   6360 S  78.9 23.6   1:07.75 app                                                                                                                
28448 mysql     20   0 1344496 202720  10572 S  54.3 19.8   0:25.06 mysqld                                                                                                             
28602 isucon    20   0  168232  36236   5896 S  40.6  3.5   0:21.73 benchmarker                    
```

おお、appのほうがcpuくってる
```
{"pass":true,"score":12242,"success":11248,"fail":0,"messages":[]}
```
スコアアップ

まだslow_logになんかいる
```
# User@Host: root[root] @ localhost [127.0.0.1]
# Query_time: 0.078341  Lock_time: 0.000037 Rows_sent: 10057  Rows_examined: 20114
SET timestamp=1471602941;
SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` ORDER BY `created_at` DESC;
```
WHEREないんだがｗ

postsをオンメモリにできれば高速化できそう。

これからはappを見ていく

journalctl ってなんやねん。。覚えとこう
```
        log_format  isucon '$time_local $msec\t$status\treqtime:$request_time\t'
                           'in:$request_length\tout:$bytes_sent\trequest:$request\t'
                           'acceptencoding:$http_accept_encoding\treferer:$http_referer\t'
                           'ua:$http_user_agent';
        access_log /var/log/nginx/access.log isucon;
```

去年のisuconのディレクトリから拝借してきた


とりあえずimageのリクエストがやばそう
nginxから静的に返すようにする

なにこれDBに画像データあるの...

全部ファイルに落とすようにした、静的ファイルのルーティングがうまくいかない

nginxの静的ファイルの配信、アプリとどっちが優先されるんだ？

静的ファイル配信できた。
{"pass":true,"score":16055,"success":15191,"fail":0,"messages":[]}

jsとかも静的ファイルにした
{"pass":true,"score":18090,"success":17310,"fail":0,"messages":[]}

時間切れ。


