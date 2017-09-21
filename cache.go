package discuz

import (
    "fmt"
    "time"
    "json"
    "bytes"
    "strconv"
)

const (
    SECOND           = 1E9     //纳秒
    DELTA_CACHE_TIME = 60 * 10 // 10分钟
    PAGE_TO_CACHE    = 3
)

var (
    userCache  = make(map[string]User)
    topicCache = make(map[string]Topic)
    replyCache = make(map[string]Reply)
    forumCache = make(map[string]Forum)

    missedTopicChan = make(chan int)
    missedReplyChan = make(chan int)
)

// ====================================================================
// =                               接口                               =
// ====================================================================

func GetTopicFromCache(ids []int) (t []Topic) {
    length := len(ids)
    for i := 0; i < length; i++ {
        key := strconv.Itoa(ids[i])
        _, hasKey := topicCache[key]

        if hasKey {
            t = append(t, topicCache[key])
        } else {
            missedTopicChan <- ids[i]
        }
    }
    return
}

func GetReplyFromCache(ids []int) (r []Reply) {
    length := len(ids)
    for i := 0; i < length; i++ {
        key := strconv.Itoa(ids[i])
        _, hasKey := replyCache[key]

        if hasKey {
            r = append(r, replyCache[key])
        } else {
            missedReplyChan <- ids[i]
        }
    }
    return
}

// 等待SQL数据返回
func GetUserFromCache(uid int) (u User) {
    key := strconv.Itoa(uid)
    _, hasKey := userCache[key]

    if !hasKey {
        sql := bytes.NewBufferString("")
        fmt.Fprintf(sql, "SELECT uid, username FROM pre_common_member WHERE uid = %d", uid)
        buf := Exec(string(sql.Bytes()))
        cacheUserFromBuf(buf)
    }
    return userCache[key]
}

func GetForumFromCache() (f map[string]Forum) {
    return forumCache
}

// ====================================================================
// =                               User                               =
// ====================================================================

/*
从buf里生成User，并且Cache
:param buf: 字符串数组
*/
func cacheUserFromBuf(buf []byte) {
    var f []interface{}
    err := json.Unmarshal(buf, &f)

    if err != nil {
        fmt.Println(err)
    }

    number := len(f)

    for i := 0; i < number; i++ {
        tmp := f[i].([]interface{})

        uid := int(tmp[0].(float64))

        var username string

        // 有的用户名为nil，需要做一个switch去判断...
        switch v := tmp[1].(type) {
        case string:
            username = tmp[1].(string)
        case nil:
            username = "匿名用户"
        }

        key := strconv.Itoa(uid)
        userCache[key] = User{uid, username}
    }
}

// ====================================================================
// =                              Forum                               =
// ====================================================================

/*
主论坛板块缓存,无增量，无被动缓存
*/
func initMainForumCache() {
    buf := Exec("SELECT f.fid, f.name FROM pre_forum_forum f LEFT JOIN pre_forum_forumfield ff USING ( fid ) WHERE f.status = '1' and f.type='forum'")

    var f []interface{}
    err := json.Unmarshal(buf, &f)

    if err != nil {
        fmt.Println(err)
    }

    number := len(f)

    for i := 0; i < number; i++ {
        tmp := f[i].([]interface{})

        fid := int(tmp[0].(float64))
        name := tmp[1].(string)

        key := strconv.Itoa(fid)

        forumCache[key] = Forum{fid, name}
    }
}

// ====================================================================
// =                              Topic                               =
// ====================================================================

/*
从字符串buf里生成User，并且Cache
:param buf: 字符串数组
*/
func cacheTopicFromBuf(buf []byte) {
    var f []interface{}
    err := json.Unmarshal(buf, &f)

    if err != nil {
        fmt.Println(err)
    }

    number := len(f)

    for i := 0; i < number; i++ {
        tmp := f[i].([]interface{})

        tid := int(tmp[0].(float64))
        authorid := int(tmp[1].(float64))

        // 有的用户名为nil，需要做一个switch去判断...
        var author string
        switch v := tmp[2].(type) {
        case string:
            author = tmp[2].(string)
        case nil:
            author = "匿名用户"
        }

        // 有的message为nil，需要做一个switch去判断...
        var subject string
        switch v := tmp[3].(type) {
        case string:
            subject = tmp[3].(string)
        case nil:
            subject = ""
        }

        // 有的message为nil，需要做一个switch去判断...
        var message string
        switch v := tmp[4].(type) {
        case string:
            message = tmp[4].(string)
        case nil:
            message = ""
        }

        dateline := int(tmp[5].(float64))

        key := strconv.Itoa(tid)

        topicCache[key] = Topic{tid, authorid, author, subject, message, dateline}
    }
}

/*
未命中时更新缓存
*/
func cacheMissedTopicRoutine() {
    for {
        id := <-missedTopicChan

        sql := bytes.NewBufferString("")
        fmt.Fprintf(sql, "SELECT tid, authorid, author, subject, message, dateline FROM pre_forum_post WHERE tid = %d AND first = true", id)
        buf := Exec(string(sql.Bytes()))
        cacheTopicFromBuf(buf)
    }
}

/*
增量缓存
*/

func cacheDeltaTopicRoutine() {
    for {
        time.Sleep(SECOND * DELTA_CACHE_TIME)

        end := time.Seconds()
        begin := end - DELTA_CACHE_TIME

        sql := bytes.NewBufferString("")
        fmt.Fprintf(sql, "SELECT tid, authorid, author, subject, message, dateline FROM pre_forum_post WHERE dateline >= %d AND dateline <= %d AND first = true", begin, end)

        buf := Exec(string(sql.Bytes()))
        cacheTopicFromBuf(buf)
    }
}

// ====================================================================
// =                              Reply                               =
// ====================================================================

/*
从buf里生成User，并且Cache
:param buf: MYSQL返回结果的JSON byte数组
*/
func cacheReplyFromBuf(buf []byte) {

    var f []interface{}
    err := json.Unmarshal(buf, &f)

    if err != nil {
        fmt.Println(err)
    }

    number := len(f)

    for i := 0; i < number; i++ {
        tmp := f[i].([]interface{})

        pid := int(tmp[0].(float64))
        authorid := int(tmp[1].(float64))

        // 有的用户名为nil，需要做一个switch去判断...
        var author string
        switch v := tmp[2].(type) {
        case string:
            author = tmp[2].(string)
        case nil:
            author = "匿名用户"
        }

        // 有的reply为nil，需要做一个switch去判断...
        var message string
        switch v := tmp[3].(type) {
        case string:
            message = tmp[3].(string)
        case nil:
            message = ""
        }

        dateline := int(tmp[4].(float64))

        key := strconv.Itoa(pid)

        replyCache[key] = Reply{authorid, author, message, dateline}
    }
}

/*
回复未命中时缓存
*/
func cacheMissedReplyRoutine() {
    for {
        id := <-missedReplyChan
        sql := bytes.NewBufferString("")
        fmt.Fprintf(sql, "SELECT pid, authorid, author, message, dateline FROM pre_forum_post WHERE pid = %d", id)
        buf := Exec(string(sql.Bytes()))
        cacheReplyFromBuf(buf)
    }
}

/*
增量缓存
*/
func cacheDeltaReplyRoutine() {
    for {
        time.Sleep(SECOND * DELTA_CACHE_TIME)

        end := time.Seconds()
        begin := end - DELTA_CACHE_TIME

        sql := bytes.NewBufferString("")
        fmt.Fprintf(sql, "SELECT pid, authorid, author, message, dateline FROM pre_forum_post WHERE dateline >= %d AND dateline <= %d AND invisible = '0' AND first = false", begin, end)
        buf := Exec(string(sql.Bytes()))
        cacheReplyFromBuf(buf)
    }
}

func InitCache() {
    Logger.Printf("[Cache - All] Initializing Main Cache.")
    initMainForumCache()

    go cacheMissedTopicRoutine() // 必须有
    //go cacheDeltaTopicRoutine()

    go cacheMissedReplyRoutine() // 必须有
    //go cacheDeltaReplyRoutine()

    for _, f := range forumCache {
        fid := f.Fid
        for page := 0; page < PAGE_TO_CACHE; page++ {
            ListTopic(fid, page)
        }
    }

    for _, t := range topicCache {
        tid := t.Tid
        for page := 0; page < PAGE_TO_CACHE; page++ {
            ListReply(tid, page)
        }
    }
    Logger.Printf("[Cache - All] Main Cache Was Initialized.")
}