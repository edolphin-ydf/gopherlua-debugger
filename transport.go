package lua_debugger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/edolphin-ydf/gopherlua-debugger/proto"
	"io"
	"log"
	"net"
	"strconv"
)

type Transport struct {
	c       net.Conn
	Handler func(int, interface{})
}

func (t *Transport) Connect(host string, port int) error {
	var err error
	t.c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	go t.parseMsg()

	return nil
}

func (t *Transport) parseMsg() {
	var lineBuf bytes.Buffer
	r := bufio.NewReader(t.c)
	readHead := true
	cmd := 0

	for {
		tmpLine, isPrefix, err := r.ReadLine()
		if err != nil {
			t.Handler(proto.MsgIdActionReq, &proto.ActionReq{Action: proto.Stop})
			break
		}
		lineBuf.Write(tmpLine)
		if isPrefix {
			continue
		}

		if readHead {
			cmd, err = strconv.Atoi(lineBuf.String())
			if err != nil {
				log.Println("parse cmd error:", err)
				_ = t.c.Close()
				break
			}
			lineBuf = bytes.Buffer{}
			readHead = false
			continue
		}

		msg := proto.GetMsg(cmd)
		if err = json.Unmarshal(lineBuf.Bytes(), msg); err != nil {
			log.Println("unmarshal json fail", err)
			break
		}

		if t.Handler != nil {
			t.Handler(cmd, msg)
		}

		lineBuf = bytes.Buffer{}
		readHead = true
	}
}

func (t *Transport) Send(cmd int, msg interface{}) {
	if t.c == nil {
		return
	}
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("%d\n", cmd))
	data, _ := json.Marshal(msg)
	buf.Write(data)
	buf.WriteString("\n")

	if _, err := io.Copy(t.c, &buf); err != nil {
		log.Println("send msg fail:", string(data), err)
	}
}
