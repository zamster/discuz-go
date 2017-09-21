package main

import (
    "fmt"
    "http"
    "json"
    "strconv"
    "./discuz"
)

/*
从Cookie里面的xxx_auth解码出UID，如果有UID，则用户是登陆状态
*/
func GetUidFromRequest(r *http.Request) (uid int) {

    auth, err := r.Cookie(discuz.CookiePrefix)
    if err == http.ErrNoCookie {
        return 0
    }
    cookie := auth.Value

    salt, err := r.Cookie(discuz.SaltKey)
    if err == http.ErrNoCookie {
        return 0
    }
    saltKey := salt.Value

    key := discuz.CalAuthKey(saltKey)
    uid, err = discuz.DecodeCookie(cookie, key)
    return
}

/*
浏览首页板块
*/
func index(w http.ResponseWriter, r *http.Request) {
    forums := discuz.GetForumFromCache()
    data, _ := json.Marshal(forums) // dump 成json
    fmt.Fprintf(w, "%s", string(data))
}

/*
浏览板块话题/发表板块话题
*/
func forum(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        fid, _ := strconv.Atoi(r.FormValue("fid"))
        page, _ := strconv.Atoi(r.FormValue("page"))
        t := discuz.ListTopic(fid, page)
        data, _ := json.Marshal(t)
        fmt.Fprintf(w, "%v", string(data))
    } else if r.Method == "POST" {
        uid := GetUidFromRequest(r)
        if uid != 0 {
            fid, _ := strconv.Atoi(r.FormValue("fid"))
            subject := r.FormValue("subject")
            message := r.FormValue("message")
            discuz.PostTopic(uid, fid, subject, message)
        }
    }
}

/*
查看话题/回复话题
*/
func topic(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        tid, _ := strconv.Atoi(r.FormValue("tid"))
        page, _ := strconv.Atoi(r.FormValue("page"))
        result := discuz.ListReply(tid, page)
        data, _ := json.Marshal(result)
        fmt.Fprintf(w, "%v", string(data))
    } else if r.Method == "POST" {
        uid := GetUidFromRequest(r)
        if uid != 0 {
            fid, _ := strconv.Atoi(r.FormValue("fid"))
            tid, _ := strconv.Atoi(r.FormValue("tid"))
            message := r.FormValue("message")
            discuz.PostReply(uid, fid, tid, message)
        }
    }
}

/*
浏览用户
*/
func user(w http.ResponseWriter, r *http.Request) {
    uid, _ := strconv.Atoi(r.FormValue("uid"))
    user := discuz.GetUserFromCache(uid)
    data, _ := json.Marshal(user)
    fmt.Fprintf(w, "%v", string(data))
}

/*
通过WAP站登陆，写入Cookie
*/
func login(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "xx", http.StatusFound)
}

/*
通过WAP站注册，写入Cookie
*/
func register(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "xx", http.StatusFound)
}

func main() {
    // 初始化缓存
    discuz.InitCache()

    // HTTP 接口
    http.HandleFunc("/", index)
    http.HandleFunc("/forum", forum)
    http.HandleFunc("/topic", topic)
    http.HandleFunc("/user", user)
    http.HandleFunc("/login", login)
    http.HandleFunc("/register", register)

    http.ListenAndServe("172.16.1.189:80", nil)
}