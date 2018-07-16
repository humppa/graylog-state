/* graylog-state */

package main

import (
	"io/ioutil"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/dghubble/sling"
	"gopkg.in/yaml.v2"
)

type config struct {
	API  string `yaml:"api"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

type authResponse struct {
	Token     string `json:"session_id"`
	Timestamp string `json:"valid_until"`
}

type errorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func init() {
	log.SetFlags(log.Llongfile | log.Ldate | log.Lmicroseconds | log.LUTC)
}

func (c *config) read() *config {
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}
	return c
}

func authenticate(c config) string {
	server := sling.New().Base(c.API).Post("system/sessions")
	req := &authRequest{
		Username: c.User,
		Password: c.Pass,
	}
	apiResp := new(authResponse)
	apiErr := new(errorResponse)
	_, err := server.BodyJSON(req).Receive(apiResp, apiErr)
	if err != nil {
		spew.Dump(err)
	}
	if apiErr.Type != "" {
		spew.Dump(apiErr)
	}
	if apiResp.Token == "" {
		spew.Dump(err, apiErr, apiResp)
	}

	return apiResp.Token
}

func main() {
	var config config
	config.read()
	spew.Dump(config)
	token := authenticate(config)
	spew.Dump(token)
}
