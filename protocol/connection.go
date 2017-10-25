package protocol

type Connection interface {
	Send([]Message) error
	SendJsonp([]Message, string) error
	IsConnected() bool
	IsSingleShot() bool
	Close()
	Priority() int
}
