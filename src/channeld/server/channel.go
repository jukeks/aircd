package main

import (
	"channeld/protocol"
	"log"
)

type Channel struct {
	name  string
	mode  string
	users []*User

	incoming chan ClientAction
}

func NewChannel(name string) *Channel {
	c := new(Channel)
	c.name = name
	c.users = []*User{}

	c.incoming = make(chan ClientAction, 1000)

	return c
}

func (channel *Channel) Serve() {
	for {
		select {
		case action := <-channel.incoming:
			channel.handleMessage(action)
		}

		if len(channel.users) == 0 {
			return
		}
	}
}

func (channel *Channel) handleMessage(action ClientAction) {
	switch action.message.GetType() {
	case protocol.PRIVATE:
		msg := action.message.(protocol.PrivateMessage)
		channel.handlePrivateMessage(action.user, msg)
	case protocol.JOIN:
		msg := action.message.(protocol.JoinMessage)
		channel.handleJoin(action.user, msg)
	case protocol.PART:
		msg := action.message.(protocol.PartMessage)
		channel.handlePart(action.user, msg)
	default:
		log.Printf("Channel message not implemented: %v", action)
	}
}

func (channel *Channel) addUser(user *User) {
	log.Printf("User %s joined channel %s", user.nick, channel.name)
	channel.users = append(channel.users, user)
}

func (channel *Channel) removeUser(user *User) {
	log.Printf("User %s left channel %s", user.nick, channel.name)
	for i, i_u := range channel.users {
		if i_u.nick == user.nick {
			a := channel.users
			a[i] = a[len(a)-1]
			channel.users = a[:len(a)-1]

			log.Printf("Channel has %d users", len(channel.users))
			return
		}
	}
}

func (channel *Channel) getUserNames() []string {
	users := []string{}
	for _, u := range channel.users {
		users = append(users, u.nick)
	}

	return users
}

func (channel *Channel) handleJoin(joined_user *User,
	message protocol.JoinMessage) {
	channel.addUser(joined_user)

	serialized := getSerializedMessageFrom(joined_user.hostmask(), message)
	for _, channel_user := range channel.users {
		channel_user.sendSerializedMessage(serialized)
	}

	joined_user.sendUsers(channel.getUserNames(), message.Target)
}

func (channel *Channel) handlePart(parted_user *User,
	message protocol.PartMessage) {
	channel.removeUser(parted_user)

	serialized := getSerializedMessageFrom(parted_user.hostmask(), message)
	for _, channel_user := range channel.users {
		channel_user.sendSerializedMessage(serialized)
	}

	parted_user.sendMessageFrom(parted_user.hostmask(), message)
}

func (channel *Channel) handlePrivateMessage(sending_user *User,
	message protocol.PrivateMessage) {
	serialized := getSerializedMessageFrom(sending_user.hostmask(), message)
	for _, channel_user := range channel.users {
		if channel_user == sending_user {
			continue
		}

		channel_user.sendSerializedMessage(serialized)
	}
}
