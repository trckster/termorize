package main

import (
	"termorize/src/config"
	"termorize/src/http"
)

func main() {
	config.LoadEnv()
	http.LaunchServer()
}
