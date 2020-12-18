package dto

/*
Набор поддерживаемых команд
протого моста между клиентами
Обработка всех команд происходит в Write методе
*/
const (
	ErrorCOMMAND         uint16 = 1
	PingCOMMAND          uint16 = 2
	RegisterCOMMAND      uint16 = 3
	GenerateCOMMAND      uint16 = 4
	InitByIDCOMMAND      uint16 = 5
	InitByNameCOMMAND    uint16 = 6
	ConnectByIDCOMMAND   uint16 = 7
	ConnectByNameCOMMAND uint16 = 8
	DataCOMMAND          uint16 = 9
	DestroyConCOMMAND    uint16 = 10
	PropertiesCOMMAND    uint16 = 11
	SaveDataCOMMAND      uint16 = 12
)
