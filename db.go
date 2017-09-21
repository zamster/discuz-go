package discuz

import (
    "json"
    "github.com/Philio/GoMySQL"
)

const (
    DB_HOST    = "172.16.3.78"
    DB_USER    = "root"
    DB_PASS    = "xx"
    DB_NAME    = "xx"
    MAX_WORKER = 8
)

// 总线，输入的为SQL字符串，输出的为查询结果序列化后的字符串
type Bus struct {
    In  chan string
    Out chan []byte
}

// 总线总数
var ch [MAX_WORKER]*Bus

// 轮训用的计数器
var rb int

func SpawnWoker() (b *Bus) {
    pool := new(Bus)

    pool.In = make(chan string)
    pool.Out = make(chan []byte)

    go func() {
        db, err := mysql.DialTCP(DB_HOST, DB_USER, DB_PASS, DB_NAME)

        if err != nil {
            Logger.Printf("[Batabase - Error] %v", err)
        }

        db.Reconnect = true // 允许断开自动重连

        defer db.Close() // goutine结束时再close

        for {
            sql := <-pool.In

            db.Query("SET NAMES utf8")

            err = db.Query(sql)

            if err != nil {
                Logger.Printf("[Batabase - Error] %v", err)
            }

            result, err := db.StoreResult()
            if err != nil {
                Logger.Printf("[Batabase - Error] %v", err)
            }

            /*rc := result.RowCount() // 结果数量*/
            /*Logger.Printf("[Batabase - Query - %v] %v", rc, sql)*/

            rows := result.FetchRows()

            data, err := json.Marshal(rows)

            if err != nil {
                Logger.Printf("[Batabase - Error] %v", err)
                pool.Out <- nil
            } else {
                pool.Out <- data
            }

            err = result.Free()
            if err != nil {
                Logger.Printf("[Batabase - Error] %v", err)
            }
        }
    }()

    return pool
}

// 轮询
func RoundRobin() (b *Bus) {
    rb = (rb + 1) % MAX_WORKER
    return ch[rb]
}

// 同步获取SQL数据接口
func Exec(sql string) (data []byte) {
    c := RoundRobin()
    c.In <- sql
    return <-c.Out
}

// 初始化Goroutine
func init() {
    for i := 0; i < MAX_WORKER; i++ {
        ch[i] = SpawnWoker()
    }
}

//Load Config File and Init DB
func ConnectDB() (c *mysql.Client) {
    c, err := mysql.DialTCP(DB_HOST, DB_USER, DB_PASS, DB_NAME)

    if err != nil {
        Logger.Printf("[Batabase - Error] %v", err)
    }
    return
}