// Package views implements dagobert's server-rendered Templ views.
package views

import (
	"slices"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

// switchSections are the list sub-pages that exist on every real case and carry
// no detail id, so a quick case switch can land on their list page directly.
var switchSections = []string{"events", "assets", "malware", "indicators", "evidences", "tasks", "notes", "summary"}

// SectionSuffix computes the path suffix to preserve when switching from the
// case identified by cid to another case, given the current route. It keeps the
// section list (dropping any detail id), keeps the vis/* pages whole, and falls
// back to summary/ for anything else so a switch never lands on a 404 or
// forbidden page. Used both to build the switch link and (via ValidSection) to
// re-validate the incoming ?to= on the server, so the two always agree.
func SectionSuffix(route, cid string) string {
	rest, ok := strings.CutPrefix(route, "/cases/"+cid+"/")
	if cid == "" || !ok {
		return "summary/"
	}

	if rest == "vis/network" || rest == "vis/mitre" {
		return rest
	}

	section, _, _ := strings.Cut(rest, "/")
	if slices.Contains(switchSections, section) {
		return section + "/"
	}

	return "summary/"
}

// ValidSection re-validates an incoming ?to= suffix against the same allow-list
// SectionSuffix produces, falling back to summary/ so a hand-crafted query
// string can't inject an arbitrary path.
func ValidSection(to string) string {
	if to == "vis/network" || to == "vis/mitre" {
		return to
	}
	if section, ok := strings.CutSuffix(to, "/"); ok && slices.Contains(switchSections, section) {
		return to
	}
	return "summary/"
}

// Env is the core view-model injected into template renders via middleware.
// It carries the current case, user, customizable valueLists, active route, and the
// ACL predicate used to gate links and actions.
type Env struct {
	Case             model.Case
	User             model.User
	ValueLists       model.ValueLists
	CustomAttributes []model.CustomAttribute
	Route            string
	Allowed          func(method, url string) (string, bool)
}
