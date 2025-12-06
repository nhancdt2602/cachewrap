package rules

// CachewrapRule represents a function that needs caching instrumentation
type CachewrapRule struct {
	FuncName      string
	ReceiverType  string
	ParamNames    []string
	PackageName   string
	ImportPath    string
	CacheInstance string
	KeyPrefix     string
	IsEvict       bool
	IsClear       bool
}

func (r *CachewrapRule) String() string {
	receiver := ""
	if r.ReceiverType != "" {
		receiver = "(" + r.ReceiverType + ")."
	}
	return r.PackageName + "." + receiver + r.FuncName
}
