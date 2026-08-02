//go:build taglock_evolution_example

package v2breaking

import "encoding/json"

type Metadata struct {
	RequestID string `json:"requestId"`
}
type User struct {
	Metadata     Metadata `json:"metadata"`
	Legacy       Metadata `json:",inline"`
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	PasswordHash string   `json:"password"`
}

func (User) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]any{"opaque": true}) }
