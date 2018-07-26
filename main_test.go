/* main_test.go
 * Copyright (c) 2018 Tuomas Starck
 */

package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dghubble/sling"
	"gopkg.in/yaml.v2"
)

var equalSource = []struct {
	yml []byte
	jsn []byte
}{
	{
		[]byte(`configuration: { bind_address: 0.0.0.0, port: 1514 }`),
		[]byte(`{ "bind_address": "0.0.0.0", "port": 1514 }`),
	},
	{
		[]byte(`configuration: { bind_address: 127.0.0.1, port: 1514 }`),
		[]byte(`{ "bind_address": "127.0.0.1", "port": 1514 }`),
	},
}

var inequalSource = []struct {
	yml []byte
	jsn []byte
}{
	{
		[]byte(`configuration: { bind_address: 0.0.0.0, port: 1514 }`),
		[]byte(`{ "bind_address": "10.0.0.1", "port": 1514 }`),
	},
	{
		[]byte(`configuration: { bind_address: 0.0.0.0, port: 1514 }`),
		[]byte(`{ "bind_address": "0.0.0.0", "port": 4514 }`),
	},
}

func TestUnitDynamicDiffEqual(t *testing.T) {
	var a Input
	var b inputFromAPI

	for _, str := range equalSource {
		yaml.Unmarshal(str.yml, &a)
		json.Unmarshal(str.jsn, &b.Attr)

		if dynDiff(a.Conf, b.Attr) {
			t.Error("dynDiff() should have been `false` for equal inputs")
		}
	}
}

func TestUnitDynamicDiffInequal(t *testing.T) {
	var a Input
	var b inputFromAPI

	for _, str := range inequalSource {
		yaml.Unmarshal(str.yml, &a)
		json.Unmarshal(str.jsn, &b.Attr)

		if !dynDiff(a.Conf, b.Attr) {
			t.Error("dynDiff() should have been `true` for equal inputs")
		}
	}
}

func TestSystem(t *testing.T) {
	var config config
	var token string
	var api *sling.Sling

	inputState := new(inputsFromAPI)

	_ = spew.Sdump(time.Now())

	t.Run("ConfAuth", func(t *testing.T) {
		config.read("testdata/00-auth.yaml")
	})

	t.Run("Auth", func(t *testing.T) {
		token = authenticate(config)
		if len(token) != 36 {
			t.Error("authenticate() should have returned 36 char token")
		}
	})

	t.Run("Sling", func(t *testing.T) {
		api = makeClient(config, token)
	})

	t.Run("Inputs/Create", func(t *testing.T) {
		config.read("testdata/10-inputs.yaml")
		doInputs(api, config)
		api.New().Get("system/inputs").ReceiveSuccess(inputState)
		if inputState.Total != 1 {
			t.Error("API should have exactly 1 input")
		}
	})

	t.Run("Inputs/Update", func(t *testing.T) {
		config.read("testdata/11-inputs.yaml")
		doInputs(api, config)
		api.New().Get("system/inputs").ReceiveSuccess(inputState)
		if inputState.Total != 1 {
			t.Error("API should have exactly 1 input")
		}
	})

	t.Run("Inputs/Create", func(t *testing.T) {
		config.read("testdata/12-inputs.yaml")
		doInputs(api, config)
		api.New().Get("system/inputs").ReceiveSuccess(inputState)
		if inputState.Total != 2 {
			t.Error("API should have exactly 2 inputs")
		}
	})

	t.Run("Inputs/Remove", func(t *testing.T) {
		config.read("testdata/13-inputs.yaml")
		doInputs(api, config)
		api.New().Get("system/inputs").ReceiveSuccess(inputState)
		if inputState.Total != 0 {
			t.Error("API should have no inputs")
		}
	})
}
