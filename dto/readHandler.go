package dto

// ClientReadHandler - стандартный обработчик для чтения пакета между узлами внутри системы
type ClientReadHandler func(msg Message, err error) error

//ServerReadHandler - обработчик чтания данных с системы для отправки в интернет
type ServerReadHandler func([]byte, error) error
