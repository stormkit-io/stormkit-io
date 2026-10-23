package analytics

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"gopkg.in/guregu/null.v3"
)

// maxCampaignValueLength bounds each campaign value, since they come straight
// from a visitor-controlled URL.
const maxCampaignValueLength = 200

// Campaign is the attribution a visitor landed with: the UTM parameters and the
// ref parameter that directories and launch sites commonly append.
type Campaign struct {
	Source null.String
	Medium null.String
	Name   null.String
	Ref    null.String
}

// CampaignFromQuery reads the campaign parameters from a landing URL's query.
// Absent, empty or non-UTF-8 values are left invalid.
func CampaignFromQuery(q url.Values) Campaign {
	return Campaign{
		Source: campaignValue(q.Get("utm_source")),
		Medium: campaignValue(q.Get("utm_medium")),
		Name:   campaignValue(q.Get("utm_campaign")),
		Ref:    campaignValue(q.Get("ref")),
	}
}

// CampaignFromRawQuery is CampaignFromQuery for an unparsed query string, with
// or without its leading "?".
func CampaignFromRawQuery(raw string) Campaign {
	q, _ := url.ParseQuery(strings.TrimPrefix(raw, "?"))

	return CampaignFromQuery(q)
}

func campaignValue(v string) null.String {
	v = strings.TrimSpace(strings.ReplaceAll(v, "\x00", ""))

	if v == "" || !utf8.ValidString(v) {
		return null.String{}
	}

	if utf8.RuneCountInString(v) > maxCampaignValueLength {
		v = string([]rune(v)[:maxCampaignValueLength])
	}

	return null.StringFrom(v)
}
