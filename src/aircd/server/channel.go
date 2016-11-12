package main

type Channel struct {
    name string
    mode string
    users []*User
}

func (channel *Channel) get_users() []string {
    users := []string{}
    for _, u := range channel.users {
        users = append(users, u.nick)
    }

    return users
}