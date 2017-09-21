package discuz

import (
    "os"
    "fmt"
    "log"
    "json"
    "time"
    "bytes"
)

type Forum struct {
    Fid  int
    Name string
}

type Topic struct {
    Tid      int
    Authorid int
    Author   string
    Subject  string
    Message  string
    Dateline int
}

type Reply struct {
    Authorid int
    Author   string
    Message  string
    Dateline int
}

type User struct {
    Uid      int
    Username string
}

var (
    PERPAGE    = 10
    DELTA_POST = 3
    Logger     = log.New(os.Stdout, "", log.Ldate|log.Ltime)
)

func arrayFromBuf(buf []byte) (arr []int) {
    var f []interface{}
    err := json.Unmarshal(buf, &f)

    if err != nil {
        fmt.Println(err)
    }

    for i := 0; i < len(f); i++ {
        tmp := f[i].([]interface{})

        id := int(tmp[0].(float64))
        arr = append(arr, id)
    }
    return
}

// 读取帖子列表ID
func RangeForum(fid, begin int) (ids []int) {
    sql := bytes.NewBufferString("")
    fmt.Fprintf(sql, "SELECT tid FROM pre_forum_thread WHERE fid = %d AND displayorder IN (0, 1, 2, 3, 4) ORDER BY lastpost DESC LIMIT %d, %d", fid, begin, PERPAGE)
    buf := Exec(string(sql.Bytes()))
    return arrayFromBuf(buf)
}

// 读取帖子列表
func ListTopic(fid, page int) (t []Topic) {
    ids := RangeForum(fid, page*PERPAGE)
    topics := GetTopicFromCache(ids)
    return topics
}

// 查看帖子的回复ID
func RangeReply(tid, begin int) (ids []int) {
    sql := bytes.NewBufferString("")
    fmt.Fprintf(sql, "SELECT pid FROM pre_forum_post WHERE tid= %d AND invisible='0' AND first = false ORDER BY dateline LIMIT %d, %d", tid, begin, PERPAGE)
    buf := Exec(string(sql.Bytes()))
    return arrayFromBuf(buf)
}

func IncTopicView(tid int) {
    db := ConnectDB()
    db.Query("SET NAMES utf8")
    sql := bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE LOW_PRIORITY pre_forum_thread SET views=views+1 WHERE tid = %d", tid)
    db.Query(string(sql.Bytes()))
    db.Close()
}

// 查看帖子的回复
func ListReply(tid, page int) (r []Reply) {
    IncTopicView(tid)
    ids := RangeReply(tid, page*PERPAGE)
    replies := GetReplyFromCache(ids)
    return replies
}

/*
发表帖子
*/
func PostTopic(authorid, fid int, subject, message string) (err os.Error) {

    dateline := time.Seconds() // int

    u := GetUserFromCache(authorid)
    author := u.Username

    db := ConnectDB()
    db.Query("SET NAMES utf8")

    /*Insert Thread*/
    sql := bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_forum_thread (fid, posttableid, readperm, price, typeid, sortid, author, authorid, subject, dateline, lastpost, lastposter, displayorder, digest, special, attachment, moderated, status, isgroup, replycredit, closed) VALUES ('%d', '0', '0', '0', '0', '0', '%s', '%d', '%s', '%d', '%d',  '%s', '0', '0', '0', '0', '0', '32', '0', '0', '0')", fid, author, authorid, subject, dateline, dateline, author)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    tid := db.LastInsertId

    // Insert Post
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_forum_post_tableid SET `pid`=''")
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    pid := db.LastInsertId

    // Insert Post
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_forum_post SET `fid` = '%d', `tid` = '%d', `author` = '%s', `authorid` = '%d', `subject` = '%s', `message` = '%s', `dateline` = '%d', `first` = '1', `invisible`='0', `anonymous`='0',`usesig`='1',`htmlon`='0',`bbcodeoff`='-1',`smileyoff`='-1',`parseurloff`='', `attachment`='0',`status`='0', `pid`='%d'", fid, tid, author, authorid, subject, message, dateline, pid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*Action Log*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_common_member_action_log (`uid`, `action`, `dateline`) VALUES ('%d', '1', '%d')", authorid, dateline)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*Recent Note*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE  pre_common_member_field_home SET `recentnote`='%s' WHERE `uid`='%d'", subject, authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新今日发帖量*/
    lt := time.LocalTime()
    year := lt.Year
    month := lt.Month
    day := lt.Day
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_stat SET `post`=`post`+1 WHERE daytime='%d%d%d'", year, month, day)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新用户发表量*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member_count SET extcredits1=extcredits1+'5',threads=threads+'1',posts=posts+'1' WHERE uid ='%d'", authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新积分*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member SET credits=credits+5 WHERE uid='%d'", authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新“用户最后一次发表”*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member_status SET lastpost='%d' WHERE uid='%d'", dateline, authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新板块最后回复人*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_forum_forum SET lastpost='%s', threads=threads+1, posts=posts+1, todayposts=todayposts+1 WHERE fid='%d'", author, fid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    db.Close()

    var arr []int
    arr = append(arr, int(tid))
    GetTopicFromCache(arr)

    return
}

/*
回复帖子
*/
func PostReply(authorid, fid, tid int, message string) (err os.Error) {

    dateline := time.Seconds() // int

    u := GetUserFromCache(authorid)
    author := u.Username

    db := ConnectDB()
    db.Query("SET NAMES utf8")

    sql := bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_forum_post_tableid SET `pid`=''")
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    pid := db.LastInsertId

    /*Insert Post*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_forum_post SET `fid` = '%d', `tid` = '%d', `author` = '%s', `authorid` = '%d', `message` = '%s', `dateline` = '%d', `first` = '0', `invisible`='0', `anonymous`='0',`usesig`='1',`htmlon`='0',`bbcodeoff`='-1',`smileyoff`='-1',`parseurloff`='', `attachment`='0',`status`='0', `pid`='%d'", fid, tid, author, authorid, message, dateline, pid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*Action Log*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "INSERT INTO pre_common_member_action_log (`uid`, `action`, `dateline`) VALUES ('%d', '1', '%d')", authorid, dateline)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新今日发帖量*/
    lt := time.LocalTime()
    year := lt.Year
    month := lt.Month
    day := lt.Day
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_stat SET `post`=`post`+1 WHERE daytime='%d%d%d'", year, month, day)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*帖子最后回复人*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_forum_thread SET lastposter='%s', lastpost='%d', replies=replies+1 WHERE tid='%d'", author, dateline, tid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新用户发表量*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member_count SET extcredits1=extcredits1+'1', posts=posts+'1' WHERE uid ='%d'", authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新积分*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member SET credits=credits+1 WHERE uid='%d'", authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新“用户最后一次发表”*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_common_member_status SET lastpost='%d' WHERE uid='%d'", dateline, authorid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    /*更新板块最后回复人*/
    sql = bytes.NewBufferString("")
    fmt.Fprintf(sql, "UPDATE pre_forum_forum SET lastpost='%s', posts=posts+1, todayposts=todayposts+1 WHERE fid='%d'", author, fid)
    fmt.Println(string(sql.Bytes()))
    db.Query(string(sql.Bytes()))

    db.Close()

    var arr []int
    arr = append(arr, int(pid))
    GetReplyFromCache(arr)

    return
}