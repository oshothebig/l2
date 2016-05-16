package flexswitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"l2/lldp/utils"
)

const (
	CLIENTS_FILE_NAME = "clients.json"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func getClient(fileName string, process string) (*ClientJson, error) {
	var allClients []ClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to open dhcpd config file",
			err, fileName))
		return nil, err
	}
	json.Unmarshal(data, &allClients)
	for _, client := range allClients {
		if client.Name == process {
			return &client, nil
		}
	}
	return nil, errors.New("couldn't find dhcprelay port info")
}
