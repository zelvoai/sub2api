package dto

import (
	"reflect"
	"strings"
	"testing"
)

// TestSystemSettings_CoversPublicSettings makes drift obvious when a user-facing
// settings field is added to public settings but not admin settings, or vice versa.
//
// Intentional exclusions:
// - public-only convenience fields that admin does not expose
// - admin-only secrets / configured flags that public must never expose
func TestSystemSettings_CoversPublicSettings(t *testing.T) {
	adminKeys := jsonTags(reflect.TypeOf(SystemSettings{}))
	publicKeys := jsonTags(reflect.TypeOf(PublicSettings{}))

	publicOnlyFields := map[string]string{
		"force_email_on_third_party_signup": "public auth UX field, not part of admin system settings payload",
		"sora_client_enabled":             "upstream-only field, not surfaced in this fork's admin payload",
		"version":                         "public response includes runtime version, admin payload is settings-oriented",
	}

	var missing []string
	for key := range publicKeys {
		if _, ok := adminKeys[key]; ok {
			continue
		}
		if _, allowed := publicOnlyFields[key]; allowed {
			continue
		}
		missing = append(missing, key)
	}

	if len(missing) > 0 {
		t.Fatalf("dto.SystemSettings is missing JSON fields present on dto.PublicSettings: %s\n"+
			"if the field should stay public-only, document it in publicOnlyFields with a reason.", strings.Join(missing, ", "))
	}
}
