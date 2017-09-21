package discuz

import (
    "os"
    "fmt"
    "bytes"
    "strings"
    "strconv"
    "crypto/md5"
    "encoding/base64"
)

// 从Discuz的config_global.php中获取的AuthKey, 用意生成解码用的key
const (
    CookiePrefix = "UqpT_c401_auth"
    SaltKey      = "UqpT_c401_saltkey"
    AuthKey      = "e4fdcbMjMKQ1oOp8"
)

var ErrNoUidInCookie = os.NewError("discus: no uid found in cookie")

//Percent-encoding reserved characters
var percMap = map[string]string{
    "%21": "!",
    "%23": "#",
    "%24": "$",
    "%26": "&",
    "%27": "'",
    "%28": "(",
    "%29": ")",
    "%2A": "*",
    "%2B": "+",
    "%2C": ",",
    "%2F": "/",
    "%3A": ":",
    "%3B": ";",
    "%3D": "=",
    "%3F": "?",
    "%40": "@",
    "%5B": "[",
    "%5D": "]",
}

// 生成MD5 string的函数
func md5String(origin string) string {
    hash := md5.New()
    hash.Write([]byte(origin))
    encode := bytes.NewBufferString("")
    fmt.Fprintf(encode, "%x", hash.Sum())
    return encode.String()
}

// 通过SaltKey和AuthKey，计算解码用的Key
func CalAuthKey(saltKey string) string {
    hash1 := md5String(AuthKey + saltKey)
    hash2 := md5String(hash1)
    return hash2
}

// 把Discuz Cookie里面的非URI字符如"%2B"等，转化为"+"
func percCookie(origin string) string {
    result := origin
    for k, v := range percMap {
        result = strings.Replace(result, k, v, -1)
    }
    return result
}

// 从Cookie中解码UID
func DecodeCookie(cookie, key string) (uid int, err os.Error) {
    cookie = percCookie(cookie)

    keya := md5String(key[0:16])
    keyc := cookie[0:4]

    cryptKey := keya + md5String(keya+keyc)
    cryptKeyLen := len(cryptKey)

    encodedCookie := cookie[4:]
    if len(encodedCookie)%4 != 0 {
        need := 4 - len(encodedCookie)%4
        for i := 0; i < need; i++ {
            encodedCookie += "=" //用"="补足，长度为4的倍数
        }
    }
    decodedCookie, _ := base64.StdEncoding.DecodeString(encodedCookie)
    decodedCookieLen := len(decodedCookie)

    box := make([]int, 256)
    for i := 0; i < 256; i++ {
        box[i] = i
    }

    rndkey := make([]uint8, 256)
    for i := 0; i < 256; i++ {
        index := int(i % cryptKeyLen)
        rndkey[i] = cryptKey[index]
    }

    j := 0
    for i := 0; i < 256; i++ {
        j = (j + box[i] + int(rndkey[i])) % 256
        box[i], box[j] = box[j], box[i]
    }

    a := 0
    j = 0

    result := ""
    for i := 0; i < decodedCookieLen; i++ {
        a = (a + 1) % 256
        j = (j + box[a]) % 256
        box[a], box[j] = box[j], box[a]
        bit := box[(box[a]+box[j])%256]
        result = result + string(int(decodedCookie[i])^bit)
    }

    t := strings.Split(result, "\t")

    if len(t) >= 2 {
        uid, err = strconv.Atoi(t[1])
        if err != nil {
            return 0, ErrNoUidInCookie
        }
    } else {
        return 0, ErrNoUidInCookie
    }

    return
}