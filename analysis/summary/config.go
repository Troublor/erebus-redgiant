package summary

type Config struct {
	IncludeDef      bool `json:"includeDef"`
	IncludeUse      bool `json:"includeUse"`
	IncludeTransfer bool `json:"includeTransfer"`
	IncludeProfit   bool `json:"includeProfit"`
	IncludeTrace    bool `json:"includeTrace"`
}

var IncludeAllConfig = Config{
	IncludeDef:      true,
	IncludeUse:      true,
	IncludeTransfer: true,
	IncludeProfit:   true,
	IncludeTrace:    true,
}
