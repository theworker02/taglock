//go:build taglock_evolution_example

package v1

type Metadata struct {
	RequestID string `json:"request_id"`
}
type User struct {
	Metadata
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
	//taglock:deprecated since=1.4.0 remove-after=2.0.0 replacement=ContactEmail
	Email string `json:"email,omitempty"`
}
