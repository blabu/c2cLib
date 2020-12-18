package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/blabu/c2cLib/dto"
)

// *Протокол.
// *$Vx..x;yy...yy;zz...zz;tt...tt;jj...jj;ss...ss###................
// *Все сообщение разделено на заголовок (выполняется ИСКЛЮЧИТЕЛЬНО в текстовом виде латиница в кодировке ASCI)
// *И сам данные (произволный формат)
// *Заголовок состоит из следующих полей разделенных символом ';':
// *$V - "магическая последовательность" начало посылки
// *x..x - версия протокола (число в шестнадцатиричном представлении)
// *yy...yy - от кого посылка (строка ASCII)
// *zz...zz - кому посылка (строка ASCII)
// *tt...tt - тип сообщения (число в шестнадцатиричном представлении)
// *jj...jj - кол-во прыжков (после которого вернем ошибку)
// *ss...ss - размер передаваемых данных (число в шестнадцатиричном представлении)
// *### - Конец заголовка
// *
// *Пример: $V1;987654321;12345678;5;2;C###MESSAGE DATA

const headerParamSize = 6

var beginHeader = []byte("$V")
var endHeader = []byte("###")
var delim = []byte(";")

type header struct {
	protocolVer uint64 // Версия протокола
	mType       uint64 // Тип сообщения (смотри клиента)
	jumpCnt     uint64 // счетчик прыжков

	headerSize  int // Размер заголовка
	contentSize int // Размер данных

	from string
	to   string
}

// C2cParser - Парсер разбирает сообщения по протоколу
// 1 - клиент-клиент
type C2cParser struct {
	maxPackageSize uint64
	head           header
}

//CreateEmptyParser - создает интерфейс парсера с ограничением максимального размера сообщения maxSize
// Кусок принятого сообщения нужен для создания других видов парсера в будущем
func CreateEmptyParser(maxSize uint64) Parser {
	c2c := new(C2cParser)
	c2c.maxPackageSize = maxSize
	return c2c
}

//FormMessage - from - Content[0], to - Content[1], data - Content[2]
func (c2c *C2cParser) FormMessage(msg dto.Message) ([]byte, error) {
	res := make([]byte, 0, 128+len(msg.Content))
	res = append(res, beginHeader...)
	res = append(res, []byte(strconv.FormatUint(uint64(msg.Proto), 16))...)
	res = append(res, ';')
	res = append(res, msg.From...)
	res = append(res, ';')
	res = append(res, msg.To...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(uint64(msg.Command), 16))...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(uint64(msg.Jmp), 16))...)
	res = append(res, ';')
	res = append(res, []byte(strconv.FormatUint(uint64(len(msg.Content)), 16))...)
	res = append(res, []byte(endHeader)...)
	res = append(res, msg.Content...)
	return res, nil
}

// return position for start header or/and error if not find header or parsing error
func (c2c *C2cParser) parseHeader(data []byte) (int, error) {
	if data == nil {
		return -1, errors.New("Input is empty, nothing to be parsed")
	}
	index := bytes.IndexByte(data, '$')
	if index < 0 {
		return -1, fmt.Errorf("Undefined start symb of package %s", string(data))
	}
	if !bytes.EqualFold(data[index:index+2], beginHeader) {
		return index, fmt.Errorf("Package must be started from %s", beginHeader)
	}
	c2c.head.headerSize = bytes.Index(data, []byte(endHeader)) // Поиск конца заголовка
	if c2c.head.headerSize < index || c2c.head.headerSize >= len(data) {
		return index, fmt.Errorf("Undefined end header %s in message %s", endHeader, string(data))
	}
	parsed := bytes.Split(data[index+2:c2c.head.headerSize], delim)
	if len(parsed) < headerParamSize {
		return index, errors.New("Incorrect header")
	}
	var err error
	if c2c.head.protocolVer, err = strconv.ParseUint(string(parsed[0]), 16, 64); err != nil { //Версия протокола
		return index, errors.New("Icorrect protocol version, it must be a number")
	}
	switch c2c.head.protocolVer {
	case 1: // Для клиент-сервер соединения
		c2c.head.from = string(parsed[1])                                                   // от кого
		c2c.head.to = string(parsed[2])                                                     //кому
		if c2c.head.mType, err = strconv.ParseUint(string(parsed[3]), 16, 64); err != nil { //тип сообщения (команда)
			return index, errors.New("Icorrect message type, it must be a number")
		}
		if c2c.head.jumpCnt, err = strconv.ParseUint(string(parsed[4]), 16, 64); err != nil {
			return index, errors.New("Incorrect message jump type")
		}
		if c2c.head.jumpCnt == 0 {
			return index, errors.New("Jump count is zero")
		}
		var s uint64
		if s, err = strconv.ParseUint(string(parsed[5]), 16, 64); err != nil { //размер сообщения
			return index, errors.New("Icorrect message size, it must be a number")
		}
		if s > c2c.maxPackageSize {
			return index, fmt.Errorf("Income package is too big %d. Overflow internal buffer %d", s, c2c.maxPackageSize)
		}
		c2c.head.contentSize = int(s)
		c2c.head.headerSize += len(endHeader) // Add endHeader
		return index, nil
		// TODO implement another version of protocol
	default:
		return index, errors.New("Error usuported porotocol")
	}
}

//ParseMessage - from - Content[0], to - Content[1], data - Content[2]
func (c2c *C2cParser) ParseMessage(data []byte) (dto.Message, error) {
	var err error
	var i int
	if i, err = c2c.parseHeader(data); err != nil {
		return dto.Message{}, err
	}
	if len(data) < i+c2c.head.headerSize+c2c.head.contentSize {
		return dto.Message{}, errors.New("Not full message")
	}
	defer func() {
		c2c.head = header{}
	}()
	c2c.head.jumpCnt--
	content := make([]byte, c2c.head.contentSize)
	copy(content, data[i+c2c.head.headerSize:i+c2c.head.headerSize+c2c.head.contentSize])
	return dto.Message{
		Command: uint16(c2c.head.mType),
		Proto:   uint16(c2c.head.protocolVer),
		Jmp:     uint16(c2c.head.jumpCnt),
		From:    c2c.head.from,
		To:      c2c.head.to,
		Content: content,
	}, nil
}

// IsFullReceiveMsg - Проверка пришел полный пакет или нет
// TODO каждый раз парсить заголовок не эффективно надо будет переписать
func (c2c *C2cParser) IsFullReceiveMsg(data []byte) (int, error) {
	if _, err := c2c.parseHeader(data); err != nil {
		return -1, err
	}
	if len(data) >= c2c.head.contentSize+c2c.head.headerSize {
		return 0, nil
	}
	return c2c.head.contentSize + c2c.head.headerSize - len(data), nil
}

//ReadPacketHeader - Читает заголовок пакета из входного Reader-а.
// Ждет либо окончания потока io.Eof или любой другой ошибки, которую вернет Reader,
// либо конца заголовка в данном случае ###
func (c2c *C2cParser) ReadPacketHeader(r io.Reader) ([]byte, error) {
	buf := make([]byte, len(beginHeader)+headerParamSize*(len(delim)+1)+len(endHeader))
	header := make([]byte, 0, len(buf))
	for {
		if n, err := r.Read(buf); err == nil {
			header = append(header, buf[:n]...)
		} else {
			return nil, err
		}
		if bytes.Index(header, endHeader) >= 0 {
			break
		}
	}
	return header, nil
}
