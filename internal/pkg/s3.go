package pkg

import (
	"encoding/json"
	"slices"

	"github.com/daffadon/digihome/internal/schema"
)

func IsBucketPublic(policyStr string) (bool, error) {
	var policy schema.Policy
	err := json.Unmarshal([]byte(policyStr), &policy)
	if err != nil {
		return false, err
	}

	for _, stmt := range policy.Statement {
		if stmt.Effect != "Allow" {
			continue
		}

		principal := stmt.Principal["AWS"]
		if principal != "*" {
			continue
		}

		var actions []string
		switch a := stmt.Action.(type) {
		case string:
			actions = []string{a}
		case []interface{}:
			for _, item := range a {
				if str, ok := item.(string); ok {
					actions = append(actions, str)
				}
			}
		}
		if slices.Contains(actions, "s3:GetObject") {
			return true, nil
		}
	}

	return false, nil
}
