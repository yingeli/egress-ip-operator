package controllers

import (
	"strings"
)

type SNATRule struct {
	OutputInterface   string
	Source            string
	InvertDestination string
	ToSource          string
}

const (
	ruleComment = "egressip.yingeli.github.com/v1alpha1"
)

func NewSNATRule(source string, investDst string, toSource string) SNATRule {
	return SNATRule{
		OutputInterface:   "eth0",
		Source:            source,
		InvertDestination: investDst,
		ToSource:          toSource,
	}
}

func ParseSNATRule(rule string) (SNATRule, bool) {
	sr := SNATRule{}
	if !strings.Contains(rule, ruleComment) || !strings.Contains(rule, "-j SNAT") {
		return sr, false
	}
	tokens := strings.Split(rule, " ")
	i := 0
	for i < len(tokens) {
		switch tokens[i] {
		case "-o", "--out-interface":
			i += 1
			sr.OutputInterface = tokens[i]
		case "-s", "--source", "--src":
			i += 1
			sr.Source = tokens[i]
		case "-d", "--destination", "--dst":
			i += 1
			sr.InvertDestination = tokens[i]
		case "--to", "--to-source":
			i += 1
			sr.ToSource = tokens[i]
		default:
		}
		i += 1
	}
	return sr, true
}

func (r *SNATRule) Spec() []string {
	return []string{
		"-o", r.OutputInterface,
		"-s", r.Source,
		"!", "-d", r.InvertDestination,
		"-j", "SNAT", "--to", r.ToSource,
		"-m", "comment", "--comment", ruleComment,
	}
}
