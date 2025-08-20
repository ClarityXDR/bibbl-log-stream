package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func main(){
	addr := flag.String("addr", "127.0.0.1:6514", "Syslog TLS address")
	insecure := flag.Bool("insecure", true, "Skip TLS cert verification")
	count := flag.Int("count", 100, "Number of messages to send")
	interval := flag.Duration("interval", 250*time.Millisecond, "Interval between messages")
	hostname := flag.String("host", "sender01", "Hostname to include in message")
	app := flag.String("app", "demo", "App name")
	flag.Parse()

	cfg := &tls.Config{InsecureSkipVerify: *insecure}
	conn, err := tls.Dial("tcp", *addr, cfg)
	if err != nil { log.Fatalf("dial: %v", err) }
	defer conn.Close()
	w := bufio.NewWriter(conn)
	log.Printf("connected to %s", *addr)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i:=0;i<*count;i++{
		pri := 14 // facility=1 (user-level), severity=6 (info)
		ts := time.Now().Format(time.RFC3339)
		msg := fmt.Sprintf("<%d>1 %s %s %s - - - demo message %d value=%d", pri, ts, *hostname, *app, i+1, r.Intn(1000))
		if _, err := w.WriteString(msg+"\n"); err != nil { log.Fatalf("write: %v", err) }
		if err := w.Flush(); err != nil { log.Fatalf("flush: %v", err) }
		time.Sleep(*interval)
	}
	log.Printf("done")
}
