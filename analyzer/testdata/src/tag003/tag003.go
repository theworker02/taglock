package tag003

type Options struct { // want Options:`v1:Options:`
	Name string `json:"name,omitempty,omitempty"` // want `TAG003 json tag on Name repeats an option`
}
