/* graylog-state
 * Copyright (c) 2018 Tuomas Starck
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dghubble/sling"
	"gopkg.in/yaml.v2"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

type authResponse struct {
	Token     string `json:"session_id"`
	Timestamp string `json:"valid_until"`
}

type Input struct {
	Title  string `json:"title,omitempty"`
	Type   string `json:"type,omitempty"`
	Global bool   `json:"global,omitempty"`
	Conf   struct {
		Allow_override_date       bool   `json:"allow_override_date,omitempty" yaml:"allow_override_date,omitempty"`
		Bind_address              string `json:"bind_address,omitempty" yaml:"bind_address,omitempty"`
		Expand_structured_data    bool   `json:"expand_structured_data,omitempty" yaml:"expand_structured_data,omitempty"`
		Force_rdns                bool   `json:"force_rdns,omitempty" yaml:"force_rdns,omitempty"`
		Max_message_size          int    `json:"max_message_size,omitempty" yaml:"max_message_size,omitempty"`
		Override_source           bool   `json:"override_source,omitempty" yaml:"override_source,omitempty"`
		Port                      int    `json:"port,omitempty" yaml:"port,omitempty"`
		Recv_buffer_size          int    `json:"recv_buffer_size,omitempty" yaml:"recv_buffer_size,omitempty"`
		Store_full_message        bool   `json:"store_full_message,omitempty" yaml:"store_full_message,omitempty"`
		Tcp_keepalive             bool   `json:"tcp_keepalive,omitempty" yaml:"tcp_keepalive,omitempty"`
		Tls_cert_file             string `json:"tls_cert_file,omitempty" yaml:"tls_cert_file,omitempty"`
		Tls_client_auth           string `json:"tls_client_auth,omitempty" yaml:"tls_client_auth,omitempty"`
		Tls_client_auth_cert_file string `json:"tls_client_auth_cert_file,omitempty" yaml:"tls_client_auth_cert_file,omitempty"`
		Tls_enable                bool   `json:"tls_enable,omitempty" yaml:"tls_enable,omitempty"`
		Tls_key_file              string `json:"tls_key_file,omitempty" yaml:"tls_key_file,omitempty"`
		Tls_key_password          string `json:"tls_key_password,omitempty" yaml:"tls_key_password,omitempty"`
		Use_null_delimiter        bool   `json:"use_null_delimiter,omitempty" yaml:"use_null_delimiter,omitempty"`
	} `json:"configuration" yaml:"configuration"`
}

type config struct {
	API struct {
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
		URL  string `yaml:"url"`
	} `yaml:"api,omitempty"`
	Inputs []Input `yaml:"inputs,omitempty"`
}

type inputFromAPI struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Type   string      `json:"type"`
	Global bool        `json:"global,omitempty"`
	Attr   interface{} `json:"attributes,omitempty"`
}

type inputsFromAPI struct {
	Total  int            `json:"total"`
	Inputs []inputFromAPI `json:"inputs"`
}

type createInputResponse struct {
	ID string `json:"id"`
}

type errorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func init() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.LUTC)
}

func toReader(i interface{}) *bytes.Reader {
	j, err := json.Marshal(i)
	if err != nil {
		log.Fatalf("JSON marshaling failed: %v", err)
	}
	return bytes.NewReader(j)
}

func (c *config) read(path string) *config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Read failed: %v", err)
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}
	return c
}

func authenticate(conf config) string {
	apiErr := new(errorResponse)
	apiResp := new(authResponse)
	req := &authRequest{
		Username: conf.API.User,
		Password: conf.API.Pass,
	}
	_, err := sling.New().Base(conf.API.URL).Post("system/sessions").
		BodyJSON(req).Receive(apiResp, apiErr)
	if err != nil || apiResp.Token == "" {
		spew.Dump(err, apiErr, apiResp)
		log.Fatalf("Authentication failed: %q", apiErr.Message)
	}
	return apiResp.Token
}

func makeClient(conf config, token string) *sling.Sling {
	return sling.New().SetBasicAuth(token, "session").
		Base(conf.API.URL).Set("Content-Type", "application/json")
}

func (api *inputsFromAPI) has(i Input) *inputFromAPI {
	for _, ii := range api.Inputs {
		// Using `Title` as an arbitrary key for idempotency (API update
		// allows changing everything based on input ID, I think).
		if i.Title == ii.Title {
			return &ii
		}
	}
	return nil
}

func dynDiff(a, b interface{}) bool {
	va := reflect.ValueOf(a)

	if va.Kind().String() != "struct" {
		log.Fatalf("dynDiff(): 1st arg must be of type struct")
	}
	if reflect.ValueOf(b).Kind().String() != "map" {
		log.Fatalf("dynDiff(): 2nd arg must be of type map")
	}

	empty := func(t reflect.Type, v interface{}) bool {
		if t.Name() == "bool" && v.(bool) == false {
			return true
		} else if t.Name() == "int" && v.(int) == 0 {
			return true
		} else if t.Name() == "string" && v.(string) == "" {
			return true
		}
		return false
	}

	equal := func(k string, v interface{}) bool {
		key := strings.ToLower(k)
		// log.Println("equal(): key =", key)

		switch x := b.(type) {
		case map[string]interface{}:
			for xk, xv := range x {
				_ = xv
				if key == xk {
					// log.Println(":> equal key found:", key, xk)
					nro, ok := xv.(float64)
					if ok && v == int(nro) {
						// log.Println(":> int and float are the same:", v, xv)
						return true
					} else if v == xv {
						// log.Println(":> values are the same:", v, xv)
						return true
					} else {
						// log.Println(":! values differ", v, reflect.TypeOf(v), xv, reflect.TypeOf(xv))
						return false
					}
				}
			}
		default:
			log.Fatalf("dynDiff.equal(): only map[string] supported")
		}

		return false
	}

	for i := 0; i < va.NumField(); i++ {
		if empty(va.Field(i).Type(), va.Field(i).Interface()) {
			continue
		}
		if !equal(va.Type().Field(i).Name, va.Field(i).Interface()) {
			return true
		}
	}

	return false
}

func (i *Input) isDifferent(ai *inputFromAPI) bool {
	if i.Type != ai.Type || i.Global != ai.Global {
		return true
	}

	return dynDiff(i.Conf, ai.Attr)
}

func doInputs(api *sling.Sling, c config) {
	var err error
	apiErr := new(errorResponse)
	apiResp := new(createInputResponse)
	apiState := new(inputsFromAPI)

	_, err = api.New().Get("system/inputs").Receive(apiState, apiErr)
	if err != nil || apiErr.Type != "" {
		spew.Dump(err, apiErr, apiState)
		log.Fatalf("Input GET failed: %q", apiErr.Message)
	}

	for _, input := range c.Inputs {
		apiInput := apiState.has(input)
		if apiInput == nil {
			payload := toReader(input)
			_, err = api.New().Post("system/inputs").Body(payload).Receive(apiResp, apiErr)
			if err != nil || apiResp.ID == "" {
				spew.Dump(err, apiErr, apiResp)
				log.Fatalf("Failed to create input: %q", apiErr.Message)
			}
			log.Printf("Input created %q\n", apiResp.ID)
		} else if input.isDifferent(apiInput) {
			endpoint := fmt.Sprintf("system/inputs/%s", apiInput.ID)
			payload := toReader(input)
			_, err := api.New().Put(endpoint).Body(payload).Receive(apiResp, apiErr)
			if err != nil || apiErr.Type != "" {
				spew.Dump(err, apiErr, apiResp)
				log.Fatalf("Failed to update input: %q", apiErr.Message)
			}
			log.Printf("Input updated %q\n", apiResp.ID)
		}
	}

	for _, ai := range apiState.Inputs {
		remove := true
		for _, i := range c.Inputs {
			if ai.Title == i.Title {
				remove = false
			}
		}
		if remove {
			var ignore interface{}
			endpoint := fmt.Sprintf("system/inputs/%s", ai.ID)
			_, err := api.New().Delete(endpoint).Receive(ignore, apiErr)
			if err != nil || apiErr.Type != "" {
				spew.Dump(err, apiErr)
				log.Fatalf("Delete failed: %q", apiErr.Message)
			}
			log.Printf("Input deleted %q\n", ai.ID)
		}
	}
}
