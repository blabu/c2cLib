package dto

import "time"

// ClientDescriptor - base entity for client to client messanger
type ClientDescriptor struct {
	ID           uint64    `json:"ID"`
	Name         string    `json:"Name"` /*Начинается ОБЯЗАТЕЛЬНО с буквы латинского алфавита*/
	SecretKey    string    `json:"Key"`
	RegisterDate time.Time `json:"Registered"`
}
