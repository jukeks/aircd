package protocol

import (
    "testing"
    "github.com/stretchr/testify/assert"
)


func TestParseMessageF(t *testing.T) {
    ircmessage := ParseMessage("PING :jees")
    assert.Equal(t, ircmessage.getType(), PING,
                 "Message type parsed incorrectly")
    ping := ircmessage.(PingMessage)
    assert.Equal(t, ping.Token, "jees", "Ping token parsed incorrectly")


    ircmessage = ParseMessage("NICK juke")
    assert.Equal(t, ircmessage.getType(), NICK,
                 "Message type parsed incorrectly")
    nick := ircmessage.(NickMessage)
    assert.Equal(t, nick.Nick, "juke", "Nick message parsed incorrectly")


    ircmessage = ParseMessage("USER juke 0 * :Real Juke")
    assert.Equal(t, ircmessage.getType(), USER, "Message type parsed incorrectly")
    user := ircmessage.(UserMessage)
    assert.Equal(t, user.Username, "juke", "User message parsed incorrectly")
    assert.Equal(t, user.Realname, "Real Juke",
                 "User message parsed incorrectly")
    assert.Equal(t, user.Mode, uint8(0), "User message parsed incorrectly")

    ircmessage = ParseMessage("USER juke 1024 * :Real Juke")
    assert.Equal(t, ircmessage.getType(), UNKNOWN,
                 "Message type parsed incorrectly")

    ircmessage = ParseMessage("USER juke -1 * :Real Juke")
    assert.Equal(t, ircmessage.getType(), UNKNOWN,
                 "Message type parsed incorrectly")


    ircmessage = ParseMessage("PRIVMSG juke :hello there")
    assert.Equal(t, ircmessage.getType(), PRIVATE,
                 "Message type parsed incorrectly")
    private := ircmessage.(PrivateMessage)
    assert.Equal(t, private.Target, "juke",
                 "Private message parsed incorrectly")
    assert.Equal(t, private.Message, "hello there",
                 "Private message parsed incorrectly")
}
