package protocol

type ConnectionInitiationError int

const (
	NO_ERROR ConnectionInitiationError = iota
	NICK_IN_USE
)

type ClientAction struct {
	Connection *IrcConnection
	Message    IrcMessage
}

type ConnectionInitiationAction struct {
	UserMessage  UserMessage
	NickMessage  NickMessage
	Hostname     string
	Conn         *IrcConnection
	ResponseChan chan ConnectionInitiationActionResponse
}

type ConnectionInitiationActionResponse struct {
	Success   bool
	ErrorCode ConnectionInitiationError
	Reply     IrcMessage
}

type ChannelAction struct {
	OriginHostMask string
	OriginNick     string
	OriginConn     *IrcConnection
	Message        IrcMessage
}
