package connector

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/blabu/egeonC2cService/dto"
	"github.com/blabu/egeonC2cService/parser"
)

const proto = 1

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

//ConfConnection - конфигурация соединения
type ConfConnection struct {
	User        string
	Pass        string
	СhunkSize   uint64
	IsNew       bool // true - будет сделана попытка регистрации пользователя
	PingTimeout time.Duration
}

//Connection - структура реализующая интерфейс IConnection
type Connection struct {
	conn        net.Conn
	cnf         ConfConnection
	p           parser.Parser
	stop        chan bool
	receiveBuff []byte
}

//IConnection - интерфейс работы с соединением
type IConnection interface {
	Read() (from string, command uint16, data []byte, err error)
	Write(to string, command uint16, data []byte) error
	Close() error
}

//NewC2cConnection - create new c2c connection register or init than
func NewC2cConnection(conn net.Conn, cnf ConfConnection) (IConnection, error) {
	p := parser.CreateEmptyParser(cnf.СhunkSize)
	res := &Connection{
		conn: conn,
		cnf:  cnf,
		p:    p,
		stop: make(chan bool),
	}
	if cnf.IsNew {
		err := res.register()
		if err != nil {
			return nil, err
		}
	}
	err := res.init()
	if err != nil {
		return nil, err
	}
	go func() {
		dt := time.NewTicker(cnf.PingTimeout)
		defer dt.Stop()
		for {
			select {
			case <-res.stop:
				return
			case <-dt.C:
				res.ping()
			}
		}
	}()
	return res, err
}

func (c *Connection) Write(to string, command uint16, data []byte) error {
	buf, err := c.p.FormMessage(dto.Message{
		Command: command,
		Proto:   proto,
		Jmp:     3,
		From:    c.cnf.User,
		To:      to,
		Content: data,
	})
	if err != nil {
		return err
	}
	_, err = c.conn.Write(buf)
	return err
}

func (c *Connection) Read() (from string, command uint16, data []byte, err error) {
	reader := bufio.NewReader(c.conn)
	receiveBuff, err := c.p.ReadPacketHeader(reader)
	if err != nil {
		return "", 0, nil, err
	}
	restSize, err := c.p.IsFullReceiveMsg(receiveBuff)
	if err != nil {
		return "", 0, nil, err
	}
	if restSize > 0 {
		resp := make([]byte, restSize)
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(restSize) * 10 * time.Millisecond))
		_, err := io.ReadFull(reader, resp)
		if err != nil {
			return "", 0, nil, err
		}
		receiveBuff = append(receiveBuff, resp...)
	}
	m, err := c.p.ParseMessage(receiveBuff)
	if err != nil {
		return "", 0, nil, err
	}
	return m.From, m.Command, m.Content, nil
}

func (c *Connection) register() error {
	sign := sha256.Sum256([]byte(c.cnf.User + c.cnf.Pass))
	signature := base64.StdEncoding.EncodeToString(sign[:])
	if err := c.Write("0", dto.RegisterCOMMAND, []byte(signature)); err != nil {
		return err
	}
	_, cmd, data, err := c.Read()
	if err != nil {
		return err
	}
	if data == nil || cmd != dto.RegisterCOMMAND {
		return errors.New("Can not register. Error while read")
	}
	return nil
}

func (c *Connection) init() error {
	temp := sha256.Sum256([]byte(c.cnf.User + c.cnf.Pass))
	credentials := base64.StdEncoding.EncodeToString(temp[:])
	salt := randStringRunes(32)
	resSign := sha256.Sum256([]byte(c.cnf.User + salt + credentials))
	signature := base64.StdEncoding.EncodeToString(resSign[:])
	if err := c.Write("0", dto.InitByNameCOMMAND, []byte(salt+";"+signature)); err != nil {
		return err
	}
	_, cmd, data, err := c.Read()
	if err != nil {
		return err
	}
	if data == nil || cmd != dto.InitByNameCOMMAND {
		return errors.New("Can not init. Errors while read")
	}
	if bytes.Index(data, []byte("INIT OK")) < 0 {
		return fmt.Errorf("Bad init %s", data)
	}
	return nil
}

func (c *Connection) connect(name string) error {
	if err := c.Write(name, dto.ConnectByNameCOMMAND, nil); err != nil {
		return err
	}
	_, cmd, data, err := c.Read()
	if err != nil {
		return err
	}
	if data == nil || cmd != dto.ConnectByNameCOMMAND {
		return errors.New("Can not connect. Errors while read")
	}
	if bytes.Index(data, []byte("CONNECT OK")) < 0 {
		return fmt.Errorf("Bad connection error")
	}
	return nil
}

func (c *Connection) ping() error {
	err := c.Write("0", dto.PingCOMMAND, nil)
	return err
}

func (c *Connection) Close() error {
	close(c.stop)
	return c.conn.Close()
}
