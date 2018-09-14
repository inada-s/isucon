package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	gsm "github.com/bradleypeabody/gorilla-sessions-memcache"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

var (
	db    *sqlx.DB
	store *gsm.MemcacheStore
)

const (
	postsPerPage   = 20
	ISO8601_FORMAT = "2006-01-02T15:04:05-07:00"
	UploadLimit    = 10 * 1024 * 1024 // 10mb
	PublicDir      = "../../../public"

	// CSRF Token error
	StatusUnprocessableEntity = 422
)

var (
	cMtx          = sync.Mutex{}
	cUser         = map[int]*User{}  // UserID => User
	cPost         = map[int]*Post{}  // PostID => Post
	cUserNameToID = map[string]int{} // UserAccountName => UserID
	cUserComCount = map[int]int{}    // UserID => CommentCount
	cPostComCount = map[int]int{}    // PostID => CommentCount
)

func InitCache() {
	cUser = map[int]*User{}
	cPost = map[int]*Post{}          // PostID => Post
	cUserNameToID = map[string]int{} // UserAccountName => UserID
	cUserComCount = map[int]int{}
	cPostComCount = map[int]int{}
}

func must(err error) {
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

// User
func GetPost(postID int) *Post {
	cMtx.Lock()
	defer cMtx.Unlock()
	return cPost[postID]
}

func AddPost(post *Post) {
	cMtx.Lock()
	defer cMtx.Unlock()
	if post == nil {
		panic("nil post")
	}
	cPost[post.ID] = post
}

func InitPost() {
	cMtx.Lock()
	defer cMtx.Unlock()
	t := time.Now()

	rows, err := db.Queryx("SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` ORDER BY `created_at` ASC")
	must(err)
	for rows.Next() {
		var post Post
		err = rows.StructScan(&post)
		must(err)

		post.User = cUser[post.UserID]
		cPost[post.ID] = &post
	}
	log.Println("InitPost", time.Since(t).Seconds())
}

// User
func GetUserByName(userName string) *User {
	cMtx.Lock()
	defer cMtx.Unlock()
	id, ok := cUserNameToID[userName]
	if !ok {
		return nil
	}
	return cUser[id]
}

func GetUser(userID int) *User {
	cMtx.Lock()
	defer cMtx.Unlock()
	return cUser[userID]
}

func AddUser(user *User) {
	cMtx.Lock()
	defer cMtx.Unlock()
	if user == nil {
		panic("nil user")
	}
	cUser[user.ID] = user
}

func InitUser() {
	cMtx.Lock()
	defer cMtx.Unlock()
	t := time.Now()

	rows, err := db.Queryx("SELECT * FROM `users`")
	must(err)
	for rows.Next() {
		var user User
		err = rows.StructScan(&user)
		must(err)

		cUser[user.ID] = &user
		cUserNameToID[user.AccountName] = user.ID
	}
	log.Println("InitUser", time.Since(t).Seconds())
}

// PostCommentCount

func GetPostCommentCount(postID int) int {
	cMtx.Lock()
	defer cMtx.Unlock()
	return cPostComCount[postID]
}

func IncPostCommentCount(postID int) {
	cMtx.Lock()
	defer cMtx.Unlock()
	cPostComCount[postID] += 1
}

func InitPostCommentCount() {
	cMtx.Lock()
	defer cMtx.Unlock()
	t := time.Now()

	posts := []Post{}
	err := db.Select(&posts, "SELECT id FROM `posts`")
	must(err)
	for _, post := range posts {
		var c int
		err := db.Get(&c, "SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` = ?", post.ID)
		must(err)
		cPostComCount[post.ID] = c
	}
	log.Println("InitPostCommentCount", time.Since(t).Seconds())
}

// UserCommentCount

func GetUserCommentCount(userID int) int {
	cMtx.Lock()
	defer cMtx.Unlock()
	return cUserComCount[userID]
}

func InitUserCommentCount() {
	cMtx.Lock()
	defer cMtx.Unlock()
	t := time.Now()

	//users := []User{}
	//err := db.Select(&users, "SELECT id FROM `users`")
	//err := db.Get(&c, "SELECT * FROM `comments`)
	rows, err := db.Queryx("SELECT * FROM `comments`")
	must(err)
	for rows.Next() {
		var com Comment
		err = rows.StructScan(&com)
		must(err)
		cUserComCount[com.UserID] += 1
	}
	log.Println("InitUserCommentCount", time.Since(t).Seconds())
}

func IncUserCommentCount(userID int) {
	cMtx.Lock()
	defer cMtx.Unlock()
	cUserComCount[userID] += 1
}

type User struct {
	ID          int       `db:"id"`
	AccountName string    `db:"account_name"`
	Passhash    string    `db:"passhash"`
	Authority   int       `db:"authority"`
	DelFlg      int       `db:"del_flg"`
	CreatedAt   time.Time `db:"created_at"`
}

type Post struct {
	ID           int       `db:"id"`
	UserID       int       `db:"user_id"`
	Imgdata      []byte    `db:"imgdata"`
	Body         string    `db:"body"`
	Mime         string    `db:"mime"`
	CreatedAt    time.Time `db:"created_at"`
	CommentCount int
	Comments     []*Comment
	User         *User
	CSRFToken    string
}

type Comment struct {
	ID        int       `db:"id"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	User      *User
}

func init() {
	memcacheClient := memcache.New("localhost:11211")
	store = gsm.NewMemcacheStore(memcacheClient, "isucogram_", []byte("sendagaya"))
}

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

func memInitialize() {
	InitCache()
	InitUserCommentCount()
	InitPostCommentCount()
	InitUser()
	InitPost()
}

func tryLogin(accountName, password string) *User {
	log.Println("tryLogin", accountName, password)

	u := User{}
	err := db.Get(&u, "SELECT * FROM users WHERE account_name = ? AND del_flg = 0", accountName)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Println(err)
	if &u != nil && calculatePasshash(u.AccountName, password) == u.Passhash {
		return &u
	} else if &u == nil {
		return nil
	} else {
		return nil
	}
}

var accountRegxp = regexp.MustCompile("\\A[0-9a-zA-Z_]{3,}\\z")
var passwordRegxp = regexp.MustCompile("\\A[0-9a-zA-Z_]{6,}\\z")

func validateUser(accountName, password string) bool {
	if !(accountRegxp.MatchString(accountName) && passwordRegxp.MatchString(password)) {
		return false
	}

	return true
}

// 今回のGo実装では言語側のエスケープの仕組みが使えないのでOSコマンドインジェクション対策できない
// 取り急ぎPHPのescapeshellarg関数を参考に自前で実装
// cf: http://jp2.php.net/manual/ja/function.escapeshellarg.php
func escapeshellarg(arg string) string {
	return "'" + strings.Replace(arg, "'", "'\\''", -1) + "'"
}

// TODO
func digest(src string) string {
	// opensslのバージョンによっては (stdin)= というのがつくので取る
	out, err := exec.Command("/bin/bash", "-c", `printf "%s" `+escapeshellarg(src)+` | openssl dgst -sha512 | sed 's/^.*= //'`).Output()
	if err != nil {
		log.Println(err)
		return ""
	}

	return strings.TrimSuffix(string(out), "\n")
}

func calculateSalt(accountName string) string {
	return digest(accountName)
}

func calculatePasshash(accountName, password string) string {
	return digest(password + ":" + calculateSalt(accountName))
}

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "isuconp-go.session")
	return session
}

func getSessionUser(r *http.Request) User {
	session := getSession(r)
	uid, ok := session.Values["user_id"]
	if !ok || uid == nil {
		return User{}
	}

	u := User{}
	err := db.Get(&u, "SELECT * FROM `users` WHERE `id` = ?", uid)
	if err != nil {
		return User{}
	}

	return u
}

func getFlash(w http.ResponseWriter, r *http.Request, key string) string {
	session := getSession(r)
	value, ok := session.Values[key]

	if !ok || value == nil {
		return ""
	} else {
		delete(session.Values, key)
		session.Save(r, w)
		return value.(string)
	}
}

func makePosts(results []Post, CSRFToken string, allComments bool) ([]Post, error) {
	var posts []Post

	for _, p := range results {
		/*
			err := db.Get(&p.CommentCount, "SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?", p.ID)
			if err != nil {
				return nil, err
			}
		*/
		p.CommentCount = GetPostCommentCount(p.ID)

		query := "SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC"
		if !allComments {
			query += " LIMIT 3"
		}
		var comments []*Comment
		cerr := db.Select(&comments, query, p.ID)
		if cerr != nil {
			return nil, cerr
		}

		for i := 0; i < len(comments); i++ {
			/*
				uerr := db.Get(&comments[i].User, "SELECT * FROM `users` WHERE `id` = ?", comments[i].UserID)
				if uerr != nil {
					return nil, uerr
				}
			*/
			comments[i].User = GetUser(comments[i].UserID)
		}

		// reverse
		for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
			comments[i], comments[j] = comments[j], comments[i]
		}

		p.Comments = comments

		/*
			perr := db.Get(&p.User, "SELECT * FROM `users` WHERE `id` = ?", p.UserID)
			if perr != nil {
				return nil, perr
			}
		*/
		p.User = GetUser(p.UserID)
		p.CSRFToken = CSRFToken

		if p.User.DelFlg == 0 {
			posts = append(posts, p)
		}
		if len(posts) >= postsPerPage {
			break
		}
	}

	return posts, nil
}

func mime2ext(mime string) string {
	ext := ""
	if mime == "image/jpeg" {
		ext = ".jpg"
	} else if mime == "image/png" {
		ext = ".png"
	} else if mime == "image/gif" {
		ext = ".gif"
	}
	return ext
}

func imageURL(p Post) string {
	ext := ""
	if p.Mime == "image/jpeg" {
		ext = ".jpg"
	} else if p.Mime == "image/png" {
		ext = ".png"
	} else if p.Mime == "image/gif" {
		ext = ".gif"
	}

	return "/image/" + strconv.Itoa(p.ID) + ext
}

func isLogin(u User) bool {
	return u.ID != 0
}

func getCSRFToken(r *http.Request) string {
	session := getSession(r)
	csrfToken, ok := session.Values["csrf_token"]
	if !ok {
		return ""
	}
	return csrfToken.(string)
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := io.ReadFull(crand.Reader, k); err != nil {
		panic("error reading from random source: " + err.Error())
	}
	return hex.EncodeToString(k)
}

func getTemplPath(filename string) string {
	return path.Join("templates", filename)
}

func getInitialize(w http.ResponseWriter, r *http.Request) {
	dbInitialize()
	memInitialize()

	noprofile := r.URL.Query().Get("noprofile")
	if noprofile == "" {
		StartProfile(time.Minute)
	}
	w.WriteHeader(http.StatusOK)
}

func getLogin(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)

	if isLogin(me) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	template.Must(template.ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("login.html")),
	).Execute(w, struct {
		Me    User
		Flash string
	}{me, getFlash(w, r, "notice")})
}

func postLogin(w http.ResponseWriter, r *http.Request) {
	if isLogin(getSessionUser(r)) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	u := tryLogin(r.FormValue("account_name"), r.FormValue("password"))

	if u != nil {
		session := getSession(r)
		session.Values["user_id"] = u.ID
		session.Values["csrf_token"] = secureRandomStr(16)
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		session := getSession(r)
		session.Values["notice"] = "アカウント名かパスワードが間違っています"
		session.Save(r, w)

		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func getRegister(w http.ResponseWriter, r *http.Request) {
	if isLogin(getSessionUser(r)) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	template.Must(template.ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("register.html")),
	).Execute(w, struct {
		Me    User
		Flash string
	}{User{}, getFlash(w, r, "notice")})
}

func postRegister(w http.ResponseWriter, r *http.Request) {
	if isLogin(getSessionUser(r)) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	accountName, password := r.FormValue("account_name"), r.FormValue("password")

	validated := validateUser(accountName, password)
	if !validated {
		session := getSession(r)
		session.Values["notice"] = "アカウント名は3文字以上、パスワードは6文字以上である必要があります"
		session.Save(r, w)

		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	exists := 0
	// ユーザーが存在しない場合はエラーになるのでエラーチェックはしない
	db.Get(&exists, "SELECT 1 FROM users WHERE `account_name` = ?", accountName)

	if exists == 1 {
		session := getSession(r)
		session.Values["notice"] = "アカウント名がすでに使われています"
		session.Save(r, w)

		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	query := "INSERT INTO `users` (`account_name`, `passhash`) VALUES (?,?)"
	result, eerr := db.Exec(query, accountName, calculatePasshash(accountName, password))
	if eerr != nil {
		log.Println(eerr)
		return
	}

	session := getSession(r)
	uid, lerr := result.LastInsertId()
	if lerr != nil {
		log.Println(lerr.Error())
		return
	}

	var user User
	err := db.Get(&user, "SELECT * FROM `users` WHERE `id` = ?", uid)
	if err != nil {
		log.Println(err)
		return
	}
	AddUser(&user)

	session.Values["user_id"] = uid
	session.Values["csrf_token"] = secureRandomStr(16)
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func getLogout(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	delete(session.Values, "user_id")
	session.Options = &sessions.Options{MaxAge: -1}
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)

	results := []Post{}

	err := db.Select(&results, "SELECT posts.id, `user_id`, `body`, `mime`, posts.created_at FROM `posts` "+
		"INNER JOIN `users` ON posts.user_id = users.id WHERE users.del_flg = 0 ORDER BY `created_at` DESC LIMIT 20")
	if err != nil {
		log.Println(err)
		return
	}

	posts, merr := makePosts(results, getCSRFToken(r), false)
	if merr != nil {
		log.Println(merr)
		return
	}

	fmap := template.FuncMap{
		"imageURL": imageURL,
	}

	template.Must(template.New("layout.html").Funcs(fmap).ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("index.html"),
		getTemplPath("posts.html"),
		getTemplPath("post.html"),
	)).Execute(w, struct {
		Posts     []Post
		Me        User
		CSRFToken string
		Flash     string
	}{posts, me, getCSRFToken(r), getFlash(w, r, "notice")})
}

func getAccountName(c web.C, w http.ResponseWriter, r *http.Request) {
	/*
			user := User{}
			uerr := db.Get(&user, "SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0", c.URLParams["accountName"])
		if uerr != nil {
			log.Println(uerr)
			return
		}

		if user.ID == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	*/

	user := GetUserByName(c.URLParams["accountName"])
	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	results := []Post{}

	rerr := db.Select(&results, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC", user.ID)
	if rerr != nil {
		log.Println(rerr)
		return
	}

	posts, merr := makePosts(results, getCSRFToken(r), false)
	if merr != nil {
		log.Println(merr)
		return
	}

	/*
		commentCount := 0
		cerr := db.Get(&commentCount, "SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?", user.ID)
		if cerr != nil {
			fmt.Println(cerr)
			return
		}
	*/
	commentCount := GetUserCommentCount(user.ID)

	postIDs := []int{}
	perr := db.Select(&postIDs, "SELECT `id` FROM `posts` WHERE `user_id` = ?", user.ID)
	if perr != nil {
		log.Println(perr)
		return
	}
	postCount := len(postIDs)

	commentedCount := 0
	for _, postID := range postIDs {
		commentedCount += GetPostCommentCount(postID)
	}

	/*
		if postCount > 0 {
				s := []string{}
				for range postIDs {
					s = append(s, "?")
				}
				placeholder := strings.Join(s, ", ")

				// convert []int -> []interface{}
				args := make([]interface{}, len(postIDs))
				for i, v := range postIDs {
					args[i] = v
				}

				ccerr := db.Get(&commentedCount, "SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN ("+placeholder+")", args...)
				if ccerr != nil {
					log.Println(ccerr)
					return
				}
		}
	*/

	me := getSessionUser(r)

	fmap := template.FuncMap{
		"imageURL": imageURL,
	}

	template.Must(template.New("layout.html").Funcs(fmap).ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("user.html"),
		getTemplPath("posts.html"),
		getTemplPath("post.html"),
	)).Execute(w, struct {
		Posts          []Post
		User           User
		PostCount      int
		CommentCount   int
		CommentedCount int
		Me             User
	}{posts, *user, postCount, commentCount, commentedCount, me})
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	m, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(parseErr)
		return
	}
	maxCreatedAt := m.Get("max_created_at")
	if maxCreatedAt == "" {
		return
	}

	t, terr := time.Parse(ISO8601_FORMAT, maxCreatedAt)
	if terr != nil {
		log.Println(terr)
		return
	}

	results := []Post{}
	rerr := db.Select(&results, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC LIMIT 20", t.Format(ISO8601_FORMAT))
	if rerr != nil {
		log.Println(rerr)
		return
	}

	posts, merr := makePosts(results, getCSRFToken(r), false)
	if merr != nil {
		log.Println(merr)
		return
	}

	if len(posts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fmap := template.FuncMap{
		"imageURL": imageURL,
	}

	template.Must(template.New("posts.html").Funcs(fmap).ParseFiles(
		getTemplPath("posts.html"),
		getTemplPath("post.html"),
	)).Execute(w, posts)
}

func getPostsID(c web.C, w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.Atoi(c.URLParams["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	results := []Post{}
	rerr := db.Select(&results, "SELECT * FROM `posts` WHERE `id` = ?", pid)
	if rerr != nil {
		log.Println(rerr)
		return
	}

	posts, merr := makePosts(results, getCSRFToken(r), true)
	if merr != nil {
		log.Println(merr)
		return
	}

	if len(posts) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	p := posts[0]

	me := getSessionUser(r)

	fmap := template.FuncMap{
		"imageURL": imageURL,
	}

	template.Must(template.New("layout.html").Funcs(fmap).ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("post_id.html"),
		getTemplPath("post.html"),
	)).Execute(w, struct {
		Post Post
		Me   User
	}{p, me})
}

var dummy []byte = make([]byte, 1)

func postIndex(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)
	if !isLogin(me) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.FormValue("csrf_token") != getCSRFToken(r) {
		w.WriteHeader(StatusUnprocessableEntity)
		return
	}

	file, header, ferr := r.FormFile("file")
	if ferr != nil {
		session := getSession(r)
		session.Values["notice"] = "画像が必須です"
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	mime := ""
	if file != nil {
		// 投稿のContent-Typeからファイルのタイプを決定する
		contentType := header.Header["Content-Type"][0]
		if strings.Contains(contentType, "jpeg") {
			mime = "image/jpeg"
		} else if strings.Contains(contentType, "png") {
			mime = "image/png"
		} else if strings.Contains(contentType, "gif") {
			mime = "image/gif"
		} else {
			session := getSession(r)
			session.Values["notice"] = "投稿できる画像形式はjpgとpngとgifだけです"
			session.Save(r, w)

			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	filedata, rerr := ioutil.ReadAll(file)
	if rerr != nil {
		log.Println(rerr.Error())
	}

	if len(filedata) > UploadLimit {
		session := getSession(r)
		session.Values["notice"] = "ファイルサイズが大きすぎます"
		session.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	query := "INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)"
	result, eerr := db.Exec(
		query,
		me.ID,
		mime,
		//filedata,
		dummy,
		r.FormValue("body"),
	)
	if eerr != nil {
		log.Println(eerr.Error())
		return
	}

	pid, lerr := result.LastInsertId()
	if lerr != nil {
		log.Println(lerr.Error())
		return
	}

	var post Post
	err := db.Get(&post, "SELECT id, `user_id`, `body`, `mime`, created_at FROM `posts` WHERE id = ?", pid)
	if err != nil {
		log.Println(err)
		return
	}
	AddPost(&post)

	ioutil.WriteFile(PublicDir+"/image/"+fmt.Sprint(pid)+mime2ext(mime), filedata, 0777)

	http.Redirect(w, r, "/posts/"+strconv.FormatInt(pid, 10), http.StatusFound)
	return
}

func getImage(c web.C, w http.ResponseWriter, r *http.Request) {
	pidStr := c.URLParams["id"]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	post := Post{}
	derr := db.Get(&post, "SELECT * FROM `posts` WHERE `id` = ?", pid)
	if derr != nil {
		log.Println(derr.Error())
		return
	}

	ext := c.URLParams["ext"]

	if ext == "jpg" && post.Mime == "image/jpeg" ||
		ext == "png" && post.Mime == "image/png" ||
		ext == "gif" && post.Mime == "image/gif" {

		ioutil.WriteFile(PublicDir+"/image/"+fmt.Sprint(post.ID)+"."+ext, post.Imgdata, 0777)

		w.Header().Set("X-APP-RESPONSE", "yes")
		w.Header().Set("Content-Type", post.Mime)
		_, err := w.Write(post.Imgdata)
		if err != nil {
			log.Println(err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func postComment(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)
	if !isLogin(me) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.FormValue("csrf_token") != getCSRFToken(r) {
		w.WriteHeader(StatusUnprocessableEntity)
		return
	}

	postID, ierr := strconv.Atoi(r.FormValue("post_id"))
	if ierr != nil {
		log.Println("post_idは整数のみです")
		return
	}

	query := "INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)"
	db.Exec(query, postID, me.ID, r.FormValue("comment"))

	IncPostCommentCount(postID)
	IncUserCommentCount(me.ID)

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusFound)
}

func getAdminBanned(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)
	if !isLogin(me) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if me.Authority == 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	users := []User{}
	err := db.Select(&users, "SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC")
	if err != nil {
		log.Println(err)
		return
	}

	template.Must(template.ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("banned.html")),
	).Execute(w, struct {
		Users     []User
		Me        User
		CSRFToken string
	}{users, me, getCSRFToken(r)})
}

func postAdminBanned(w http.ResponseWriter, r *http.Request) {
	me := getSessionUser(r)
	if !isLogin(me) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if me.Authority == 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.FormValue("csrf_token") != getCSRFToken(r) {
		w.WriteHeader(StatusUnprocessableEntity)
		return
	}

	query := "UPDATE `users` SET `del_flg` = ? WHERE `id` = ?"

	r.ParseForm()
	for _, id := range r.Form["uid[]"] {
		db.Exec(query, 1, id)
	}

	http.Redirect(w, r, "/admin/banned", http.StatusFound)
}

func cmd() {
}

func main() {
	flag.Parse()

	log.Println("Hello ISUCON")

	host := os.Getenv("ISUCONP_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ISUCONP_DB_PORT")
	if port == "" {
		port = "3306"
	}
	_, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Failed to read DB port number from an environment variable ISUCONP_DB_PORT.\nError: %s", err.Error())
	}
	user := os.Getenv("ISUCONP_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("ISUCONP_DB_PASSWORD")
	dbname := os.Getenv("ISUCONP_DB_NAME")
	if dbname == "" {
		dbname = "isuconp"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	log.Println(dsn)
	db, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %s.", err.Error())
	}

	for db.Ping() != nil {
		log.Println("sleep1")
		time.Sleep(time.Second)
	}

	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

	log.Println("memInitialize")
	memInitialize()
	log.Println("memInitialize done")

	defer db.Close()

	os.MkdirAll(PublicDir+"/image", 0777)
	goji.Get("/initialize", getInitialize)
	goji.Get("/login", getLogin)
	goji.Post("/login", postLogin)
	goji.Get("/register", getRegister)
	goji.Post("/register", postRegister)
	goji.Get("/logout", getLogout)
	goji.Get("/", getIndex)
	goji.Get(regexp.MustCompile(`^/@(?P<accountName>[a-zA-Z]+)$`), getAccountName)
	goji.Get("/posts", getPosts)
	goji.Get("/posts/:id", getPostsID)
	goji.Post("/", postIndex)
	goji.Get("/image/:id.:ext", getImage)
	goji.Post("/comment", postComment)
	goji.Get("/admin/banned", getAdminBanned)
	goji.Post("/admin/banned", postAdminBanned)
	goji.Get("/*", http.FileServer(http.Dir(PublicDir)))
	goji.Serve()
}
