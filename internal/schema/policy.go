package schema

type Policy struct {
	Statement []struct {
		Effect    string         `json:"Effect"`
		Principal map[string]any `json:"Principal"`
		Action    any            `json:"Action"`
		Resource  any            `json:"Resource"`
	} `json:"Statement"`
}
