//go:build taglock_evolution_example

package v2compatible

type Metadata struct {
	RequestID string `json:"request_id"`
}
type User struct {
	Metadata
	ID        int64  `json:"id,string"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}
