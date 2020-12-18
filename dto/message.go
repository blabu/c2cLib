package dto

// Message - это данные от устройства прошедшие валидацию и разделенные на содержимое и команду
type Message struct {
	ID      uint64
	Proto   uint16
	Jmp     uint16
	Command uint16
	From    string
	To      string
	Content []byte
}

type UnSendedMsg struct {
	ID      uint64 `json:"ID"`
	Proto   uint16 `json:"Proto"`
	Command uint16 `json:"Cmd"`
	From    string `json:"From"`
	Content []byte `json:"Content"`
}
