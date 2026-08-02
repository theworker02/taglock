package collision

type A struct {
	ID string `json:"id"`
}
type B struct {
	Legacy string `json:"id"`
}
type Combined struct {
	A
	B
}
