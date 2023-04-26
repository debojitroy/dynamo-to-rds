package controller

var db = make(map[string]string)

func SetUser(user string, value string) {
	db[user] = value
}

func GetUser(user string) (string, bool) {
	value, ok := db[user]
	return value, ok
}
