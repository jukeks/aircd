package main

import (
	"log"
)

type Channel struct {
	name  string
	mode  string
	users []*User
}

func NewChannel(name string) *Channel {
	c := new(Channel)
	c.name = name
	c.users = []*User{}

	return c
}

func (channel *Channel) add_user(user *User) {
	channel.users = append(channel.users, user)
}

func (channel *Channel) remove_user(user *User) {
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

func (channel *Channel) get_users() []*User {
	users := []*User{}
	for _, u := range channel.users {
		users = append(users, u)
	}

	return users
}
