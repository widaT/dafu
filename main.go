package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/dgraph-io/badger"
	"github.com/miekg/dns"
)

var db *badger.DB

func init() {
	var err error
	options := badger.DefaultOptions("data/")
	options.Logger = nil
	db, err = badger.Open(options)
	if err != nil {
		log.Fatal(err)
	}

	if err := Replay(); err != nil {
		log.Fatal(err)
	}
}

func NewRR(s string) dns.RR { r, _ := dns.NewRR(s); return r }

func main() {
	port := flag.Int("port", 8053, "port to run on")
	flag.Parse()

	go func() {
		srv := &dns.Server{Addr: "0.0.0.0:" + strconv.Itoa(*port), Net: "udp"}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	//没有对应找到域名解析，一般要镜像forward
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {

		qDomain := ""

		if len(r.Question) > 0 {
			qDomain = r.Question[0].Name
			log.Printf("remote addr %s %#v", w.RemoteAddr(), r.Question[0].Name)
		}

		m := new(dns.Msg)

		//这边可以回空
		//m.Ns = []dns.RR{}

		//这边可以写逻辑
		str := qDomain + " IN  A  " + "127.0.0.1"
		rr := NewRR(str)
		m.SetReply(r)
		m.Ns = []dns.RR{rr}
		w.WriteMsg(m)
	})

	go func() {

		http.HandleFunc("/add", func(wr http.ResponseWriter, rq *http.Request) {

			rq.ParseForm()

			domain := rq.PostFormValue("d")
			ip := rq.PostFormValue("ip")
			tp := rq.PostFormValue("tp")

			if domain == "" || ip == "" {
				wr.Write([]byte("error"))
				return
			}

			if tp == "" {
				tp = "A"
			}
			if tp != "A" && tp != "AAAA" {
				wr.Write([]byte("error"))
				return
			}

			domain += "."

			str := domain + " IN " + tp + " " + ip
			rr := NewRR(str)

			switch rr.(type) {
			case *dns.A:
			case *dns.AAAA:
			default:
				wr.Write([]byte("error"))
				return
			}

			log.Printf("add domain:%s", domain)
			Save(domain, str)

			dns.HandleFunc(domain, func(w dns.ResponseWriter, r *dns.Msg) {
				//这边还可以根据的udp的remoteaddr 动态调度解析地址
				//w.RemoteAddr()
				log.Println("remote addr ", w.RemoteAddr())
				m := new(dns.Msg)
				m.SetReply(r)
				m.Ns = []dns.RR{rr}
				w.WriteMsg(m)
			})

			wr.Write([]byte("ok"))
		})
		http.ListenAndServe(":9898", nil)
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)
}

// Save 保存解析记录
func Save(domain, value string) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(domain), []byte(value))
	})
}

// Replay 解析记录重放
func Replay() error {
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := string(item.Key())
			var val string
			item.Value(func(v []byte) error {
				val = string(v)
				return nil
			})
			log.Println("rr ", val)
			dns.HandleFunc(k, func(w dns.ResponseWriter, r *dns.Msg) {
				m := new(dns.Msg)
				m.SetReply(r)

				m.Ns = []dns.RR{NewRR(val)}
				w.WriteMsg(m)
			})

		}
		return nil
	})
}
