package gomailer

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type client struct {
	timeOut time.Duration
}

// getDefaultClient return a default http client
func (c *client) getDefaultClient() *http.Client {
	if c.timeOut == 0 {
		c.timeOut = defaultTimeout
	}
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: c.timeOut,
		}).Dial,
		TLSHandshakeTimeout: c.timeOut * time.Second,
	}

	return &http.Client{
		Timeout:   c.timeOut,
		Transport: netTransport,
	}
}

// toJSON encode data to json and return bytes
func toJSON(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
