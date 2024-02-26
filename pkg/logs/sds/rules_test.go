//go:build sds

package sds

import (
	"testing"

	sds "github.com/DataDog/sds-go-bindings"
)

func testdata() RulesConfig {
	return RulesConfig{
		Rules: []RuleConfig{
			{
				Id:          "0",
				Name:        "Zero",
				Description: "Zero desc",
				Pattern:     "zero",
				Tags:        []string{"tag:zero"},
				MatchAction: sds.MatchActionRedact,
				Enabled:     true,
			},
			{
				Id:          "1",
				Name:        "One",
				Description: "One desc",
				Pattern:     "one",
				Tags:        []string{"tag:one"},
				MatchAction: sds.MatchActionHash,
				Enabled:     false,
			},
			{
				Id:          "2",
				Name:        "Two",
				Description: "Two desc",
				Pattern:     "two",
				Tags:        []string{"tag:two"},
				MatchAction: sds.MatchActionRedact,
				Enabled:     true,
			},
		},
	}
}

func TestGetById(t *testing.T) {
	rules := testdata()

	two := rules.GetById("2")
	if two == nil {
		t.Error("rule two exists, should be returned")
	}
	if two.Id != "2" {
		t.Error("not the good rule")
	}
	if two.Name != "Two" {
		t.Error("not the good rule")
	}
	if two.Description != "Two desc" {
		t.Error("not the good rule")
	}
	if two.Pattern != "two" {
		t.Error("not the good rule")
	}

	zero := rules.GetById("0")
	if zero == nil {
		t.Error("rule zero exists, should be returned")
	}
	if zero.Name != "Zero" {
		t.Error("not the good rule")
	}

	unknown := rules.GetById("meh")
	if unknown != nil {
		t.Error("rule doesn't exist, nothing should be returned")
	}
}

func TestOnlyEnabled(t *testing.T) {
	rules := testdata()

	onlyEnabled := rules.OnlyEnabled()
	if len(onlyEnabled.Rules) != 2 {
		t.Error("only two rules are enabled")
	}

	if onlyEnabled.GetById("0").Name != "Zero" {
		t.Error("zero should be part of the returned rules")
	}
	if onlyEnabled.GetById("2").Name != "Two" {
		t.Error("two should be part of the returned rules")
	}
}
