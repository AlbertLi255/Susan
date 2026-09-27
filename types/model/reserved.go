package model

import "strings"

// reservedNamespaces is the exact namespace blacklist. It must stay identical
// to the Susan platform's NamespaceValidator.RESERVED_NAMES
// (backend/app/core/namespace_validator.py); changes must be made in both
// places at the same time.
var reservedNamespaces = map[string]struct{}{
	"library":       {},
	"models":        {},
	"search":        {},
	"featured":      {},
	"latest":        {},
	"trending":      {},
	"cloud":         {},
	"local":         {},
	"hf":            {},
	"api":           {},
	"admin":         {},
	"account":       {},
	"docs":          {},
	"blog":          {},
	"signin":        {},
	"signup":        {},
	"device":        {},
	"download":      {},
	"system":        {},
	"susan":         {},
	"official":      {},
	"susanofficial": {},
	"pricing":       {},
	"settings":      {},
	"support":       {},
	"connect":       {},
	"susancloud":    {},
	"chat":          {},
}

// IsReservedNamespace reports whether ns is a reserved namespace that may not
// be used as a push owner, protecting official names such as "susan" and
// "official" from being squatted.
//
// The rules are case-insensitive: an exact match against the reserved
// blacklist, or a name prefixed with "susan" or "official". An empty string is
// not reserved; callers are responsible for handling empty namespaces.
func IsReservedNamespace(ns string) bool {
	ns = strings.ToLower(strings.TrimSpace(ns))
	if ns == "" {
		return false
	}

	if _, ok := reservedNamespaces[ns]; ok {
		return true
	}

	return strings.HasPrefix(ns, "susan") || strings.HasPrefix(ns, "official")
}
