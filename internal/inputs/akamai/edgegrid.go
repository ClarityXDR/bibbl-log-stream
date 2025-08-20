package akamai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Credentials holds Akamai EdgeGrid credential pieces.
type Credentials struct {
    Host         string
    ClientToken  string
    ClientSecret string // raw secret (not base64)
    AccessToken  string
}

// sign applies EG1-HMAC-SHA256 Authorization header to req.
func sign(req *http.Request, creds Credentials, bodyHash string) error {
    ts := time.Now().UTC().Format("20060102T15:04:05+0000")
    nonce := randSeq(16)
    base := fmt.Sprintf("EG1-HMAC-SHA256 client_token=%s;access_token=%s;timestamp=%s;nonce=%s;", creds.ClientToken, creds.AccessToken, ts, nonce)
    host := creds.Host
    if host == "" { host = req.Host }
    var canon string
    var names []string
    for k, vv := range req.Header {
        lk := strings.ToLower(k)
        if strings.HasPrefix(lk, "x-akamai-") {
            names = append(names, lk)
            canon += lk + ":" + strings.Join(vv, ",") + "\t"
        }
    }
    sort.Strings(names)
    data := strings.Join([]string{req.Method, host, req.URL.Path, req.URL.RawQuery, canon, bodyHash, base}, "\t")
    key := []byte(creds.ClientSecret)
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(data))
    sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
    req.Header.Set("Authorization", base+"signature="+sig)
    return nil
}

func bodySHA256Base64(b []byte) string { h := sha256.Sum256(b); return base64.StdEncoding.EncodeToString(h[:]) }

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
func randSeq(n int) string { b := make([]rune, n); for i := range b { b[i] = letters[rand.Intn(len(letters))] }; return string(b) }
