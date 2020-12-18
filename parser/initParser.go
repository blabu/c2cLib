package parser

const (
	startSymb        byte = '$'
	versionAttribute byte = 'V'
)

func InitParser(rec []byte, size uint64) (Parser, error) {
	return CreateEmptyParser(size), nil
}
