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
			channel.handle_message(action)
		}

		if len(channel.users) == 0 {
			return
		}
	}
}

func (channel *Channel) handle_message(action ClientAction) {
	switch action.message.GetType() {
	case protocol.PRIVATE:
		msg := action.message.(protocol.PrivateMessage)
		channel.handle_private_message(action.user, msg)
	case protocol.JOIN:
		msg := action.message.(protocol.JoinMessage)
		channel.handle_join(action.user, msg)
	case protocol.PART:
		msg := action.message.(protocol.PartMessage)
		channel.handle_part(action.user, msg)
	default:
		log.Printf("Channel message not implemented: %v", action)
	}
}

func (channel *Channel) add_user(user *User) {
	log.Printf("User %s joined channel %s", user.nick, channel.name)
	channel.users = append(channel.users, user)
}

func (channel *Channel) remove_user(user *User) {
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

func (channel *Channel) get_user_names() []string {
	users := []string{}
	for _, u := range channel.users {
		users = append(users, u.nick)
	}

	return users
}

func (channel *Channel) handle_join(joined_user *User,
	message protocol.JoinMessage) {
	channel.add_user(joined_user)

	serialized := get_serialized_message_from(joined_user.hostmask(), message)
	for _, channel_user := range channel.users {
		channel_user.send_serialized_message(serialized)
	}

	joined_user.send_users(channel.get_user_names(), message.Target)
}

func (channel *Channel) handle_part(parted_user *User,
	message protocol.PartMessage) {
	channel.remove_user(parted_user)

	serialized := get_serialized_message_from(parted_user.hostmask(), message)
	for _, channel_user := range channel.users {
		channel_user.send_serialized_message(serialized)
	}

	parted_user.send_message_from(parted_user.hostmask(), message)
}

func (channel *Channel) handle_private_message(sending_user *User,
	message protocol.PrivateMessage) {
	serialized := get_serialized_message_from(sending_user.hostmask(), message)
	for _, channel_user := range channel.users {
		if channel_user == sending_user {
			continue
		}

		channel_user.send_serialized_message(serialized)
	}
}
