package suppressions

type Credentials struct { // want Credentials:`v1:Credentials:`
	//taglock:ignore TAG401 -- internal-only response
	Token string `json:"token"`

	//taglock:ignore TAG301 -- intentionally reserved // want `TAG902 suppression for TAG301 is unused`
	ID string `json:"id"`

	//taglock:ignore TAG999 -- typo // want `TAG901 suppression references unknown rule "TAG999"`
	Hidden string `json:"-"` // want `TAG302 field Hidden is ignored`
}
