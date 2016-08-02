package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/yiduoyunQ/sm/sm-svr/structs"
)

type message struct {
	From   string `json:"from"`
	Method string `json:"method"`
	Domain string `json:"domain"`
	Body   string `json:"body"`
}

type topologyBody struct {
	Version  string `json:"version"`
	Topology string `json:"topology"`
}

var (
	domain      = ""
	name        = "proxyTest"
	ip          = ""
	port        = 0
	defaultFile = "/etc/upproxy/upsql-proxy.conf"
	timeout     = 10 * time.Second

	commands = []cli.Command{
		// health check
		{
			Name:        "proxyHealthCheck",
			ShortName:   "phc",
			Usage:       "upproxy health check",
			Description: "upproxy health check with get_topology",
			Flags:       flags,
			Action: func(c *cli.Context) {
				if c.IsSet("default-file") {
					defaultFile = c.String("default-file")
				}
				if c.Bool("version") {
					cli.ShowVersion(c)
					return
				}

				f, err := os.Open(defaultFile)
				if err == nil {
					r := bufio.NewReader(f)
					for {
						b, _, err := r.ReadLine()
						if err != nil {
							if err == io.EOF {
								break
							}
							panic(err)
						}

						s := strings.TrimSpace(string(b))
						if strings.Index(s, "#") == 0 {
							continue
						}
						index := strings.Index(s, "=")
						if index < 0 {
							continue
						}
						key := strings.TrimSpace(s[:index])
						if len(key) == 0 {
							continue
						}
						val := strings.TrimSpace(s[index+1:])
						if len(val) == 0 {
							continue
						}
						index = strings.Index(val, "#")
						if index > 0 {
							val = strings.TrimSpace(val[:index])
						}
						switch strings.ToLower(key) {
						default:
							continue
						case "proxy-domain":
							domain = val
						case "adm-cli-address":
							host := strings.Split(val, ":")
							strIp, strPort := host[0], host[1]
							ip = strIp
							port, err = strconv.Atoi(strPort)
							if err != nil {
								os.Exit(2)
							}
						}

					}
				}

				if c.IsSet("domain") {
					domain = c.String("domain")
				}
				if c.IsSet("ip") {
					ip = c.String("ip")
				}
				if c.IsSet("port") {
					port = c.Int("port")
				}
				if c.IsSet("time-out") {
					timeout = c.Duration("time-out")
				}

				err = check(timeout)
				if err != nil {
					os.Exit(2)
				}
			},
		},
	}
	flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "version, v",
			Usage: "print app version",
		},
		cli.StringFlag{
			Name:  "default-file, 0",
			Usage: "default config file",
		},
		cli.StringFlag{
			Name:  "domain, d",
			Usage: "upproxy domain",
		},
		cli.StringFlag{
			Name:  "ip, i",
			Usage: "upproxy ip address",
		},
		cli.IntFlag{
			Name:  "port, p",
			Usage: "upproxy port",
		},
		cli.DurationFlag{
			Name:  "time-out",
			Value: 10 * time.Second,
			Usage: "db connect time out",
		},
	}
)

func check(t time.Duration) error {
	conn, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(port), t)
	if err != nil {
		return err
	}
	defer conn.Close()
	var topology structs.Topology
	err = sendMessage(domain, "get_topology", name, "X", conn)
	if err != nil {
		return err
	}

	msg, err := readMessage(conn)
	if err != nil {
		return err
	}
	var tb topologyBody
	err = json.Unmarshal([]byte(msg.Body), &tb)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(tb.Topology), &topology)
	if err != nil {
		return err
	}

	return nil
}

func readMessage(conn net.Conn) (*message, error) {
	var msg message

	var b = make([]byte, 7)
	n, err := conn.Read(b)
	if err != nil || n != 7 {
		return nil, err
	}
	l, err := strconv.Atoi(string(b[:n-1]))
	if err != nil {
		return nil, err
	}

	b1 := make([]byte, l)
	l_start := 0
	l_end := 0
	for {
		b1_tmp := make([]byte, l)
		l_tmp, err := conn.Read(b1_tmp)
		if err != nil {
			return nil, err
		}
		l = l - l_tmp
		if l < 0 {
			log.Panic(l)
		}
		l_end = l_end + l_tmp
		copy(b1[l_start:l_end], b1_tmp)
		l_start = l_start + l_tmp
		if l == 0 {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"body": string(b1),
		"head": string(b),
	}).Debug("Receive Message")

	err = json.Unmarshal(b1, &msg)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &msg, nil
}

func sendMessage(domain, method, from, body string, conn net.Conn) error {
	reply := message{
		Domain: domain,
		Method: method,
		From:   from,
		Body:   body,
	}
	bReply, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	bReply = []byte(prependZero(strconv.Itoa(len(bReply))) + "C" + string(bReply))
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"message": string(bReply),
	}).Debug("Send Message")
	_, err = conn.Write(bReply)
	if err != nil {
		return err
	}
	return nil
}

func prependZero(s string) string {
	var ret string = ""
	for i := 0; i < 6-len(s); i++ {
		ret = ret + "0"
	}
	ret = ret + s
	return ret
}
