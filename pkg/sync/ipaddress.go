// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sync

import (
	"fmt"
	"log"
	"time"

	externalip "gitlab.com/vocdoni/go-external-ip"
)

type IPType uint

const (
	IP4 IPType = 4
	IP6 IPType = 6
)

var webCheck = []string{
	"http://checkip.dyndns.org/",
	"http://ipdetect.dnspark.com/",
	"http://dns.loopia.se/checkip/checkip.php",
}

// IPAddressPoller is a poller used to check the value
// of the current public internet IP address.
type IPAddressPoller struct {
	channels     []chan string
	pollInterval time.Duration
	consensus    *externalip.Consensus
	iptype       IPType
}

func NewIPAddressPoller(iptype IPType, pollInterval time.Duration, consensus *externalip.Consensus) *IPAddressPoller {
	if consensus == nil {
		return &IPAddressPoller{
			pollInterval: pollInterval,
			consensus:    externalip.DefaultConsensus(nil, nil),
			iptype:       iptype,
		}
	}
	return &IPAddressPoller{
		pollInterval: pollInterval,
		consensus:    consensus,
		iptype:       iptype,
	}
}

// Channel returns a channel that receives data whenever an
// IP address value is received.
func (i *IPAddressPoller) Channel() <-chan string {
	c := make(chan string, 1)
	i.channels = append(i.channels, c)
	return c
}

// poll() runs a single polling event and retrieving the internet IP.
func (i *IPAddressPoller) poll() error {
	// Make a request to each url and send to the
	// channels if a consensus is achieved
	ip, err := i.consensus.ExternalIP(uint(i.iptype))
	if err != nil {
		return fmt.Errorf("could not obtain IP address: %w", err)
	}
	for _, c := range i.channels {
		select {
		case c <- ip.String():
		default:
		}
	}
	return nil
}

// request() makes a request to a URL to get the internet IP address.
// func html_parser(url string) (string, error) {
// 	z := html.NewTokenizer(resp.Body)
// 	for {
// 		tt := z.Next()
// 		switch tt {
// 		case html.ErrorToken:
// 			return "", z.Err()
// 		case html.TextToken:
// 			text := strings.Trim(string(z.Text()), " \n\t")
// 			if text != "" {
// 				ip := ""
// 				fmt.Sscanf(text, "Current IP Address: %s", &ip)
// 				if ip != "" {
// 					return strings.Trim(ip, " \n\t"), nil
// 				}
// 			}
// 		}
// 	}
// }

// Run starts the main loop for the poller.
func (i *IPAddressPoller) Run(stopCh <-chan struct{}) error {
	if err := i.poll(); err != nil {
		log.Printf("Error polling for IP: %v", err)
	}
	for {
		select {
		case <-time.After(i.pollInterval):
			if err := i.poll(); err != nil {
				log.Printf("Error polling for IP: %v", err)
			}
		case <-stopCh:
			return nil
		}
	}
}
