package main

import (
	"encoding/json"
	"fmt"
)

func ParseVcap(src string, tags []string, subkey string) (string, error) {

	var services map[string][]struct {
		Credentials    map[string]interface{} `json:"credentials"`
		Label          string                 `json:"label"`
		Name           string                 `json:"name"`
		Plan           string                 `json:"plan"`
		Provider       interface{}            `json:"provider"`
		SyslogDrainURL interface{}            `json:"syslog_drain_url"`
		Tags           []string               `json:"tags"`
	}

	err := json.Unmarshal([]byte(src), &services)
	if err != nil {
		return "", err
	}

	for _, l := range services {
		for _, service := range l {
			tagged := false
		TAGS:
			for _, actual := range service.Tags {
				for _, tag := range tags {
					if tag == actual {
						tagged = true
						break TAGS
					}
				}
			}

			if !tagged {
				continue
			}

			if v, ok := service.Credentials[subkey]; ok {
				return fmt.Sprintf("%s", v), nil
			}
		}
	}

	return "", fmt.Errorf("no satisfactory service found")
}
