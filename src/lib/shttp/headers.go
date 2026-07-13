package shttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type Headers map[string]string

func (h Headers) Make() http.Header {
	headers := make(http.Header)

	for k, v := range h {
		pieces := strings.Split(v, ",")

		for _, piece := range pieces {
			headers.Add(k, piece)
		}
	}

	return headers
}

// String casts the headers into a string by concatenating multiple headers
// with the `;` separator.
func (h Headers) String() string {
	pieces := []string{}

	for k, v := range h {
		pieces = append(pieces, fmt.Sprintf("%s:%s", k, v))
	}

	// Make sure to sort pieces to have the same order all the time
	sort.Strings(pieces)

	return strings.Join(pieces, ";")
}

// UnmarshalJSON implements the json interface.
// This function can handle two possible value types:
// map[string]string and a string.
func (h *Headers) UnmarshalJSON(data []byte) error {
	if *h == nil {
		*h = make(Headers)
	}

	// First try to unmarshal as map
	var mapValue map[string]string

	if err := json.Unmarshal(data, &mapValue); err == nil {
		for k, v := range mapValue {
			(*h)[k] = v
		}

		return nil
	}

	// If that fails, try to unmarshal as string
	var strValue string

	if err := json.Unmarshal(data, &strValue); err != nil {
		return err
	}

	h.fillFromString(strValue)

	return nil
}

// fillFromString fills the given headers object from a headers string
// in the following format: `key1: val1, val2; key2: val3`
func (h *Headers) fillFromString(s string) {
	headers := strings.Split(s, ";")

	for _, header := range headers {
		pieces := strings.SplitN(header, ":", 2)

		if len(pieces) >= 2 {
			(*h)[strings.TrimSpace(pieces[0])] = strings.TrimSpace(pieces[1])
		}
	}
}
