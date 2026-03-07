package model

type DomainType int

const (
	DomainFull    DomainType = iota
	DomainSuffix
	DomainKeyword
	DomainRegexp
)

type DomainRule struct {
	Type       DomainType
	Value      string
	Attributes []string
}

type FileName string
type AttrKey string

type ListInfo struct {
	Name                  FileName
	HasInclusion          bool
	InclusionAttributeMap map[FileName][]AttrKey
	FullList              []DomainRule
	SuffixList            []DomainRule
	SuffixUniqueList      []DomainRule
	KeywordList           []DomainRule
	RegexpList            []DomainRule
	AttributeRuleMap      map[AttrKey][]DomainRule
	AttributeRuleList     []DomainRule
}

type ListInfoMap map[FileName]*ListInfo

type RuleMap map[string][]DomainRule

type IPRuleMap map[string][]string

func NewListInfo() *ListInfo {
	return &ListInfo{
		InclusionAttributeMap: make(map[FileName][]AttrKey),
		FullList:              make([]DomainRule, 0, 10),
		SuffixList:            make([]DomainRule, 0, 10),
		SuffixUniqueList:      make([]DomainRule, 0, 10),
		KeywordList:           make([]DomainRule, 0, 10),
		RegexpList:            make([]DomainRule, 0, 10),
		AttributeRuleMap:      make(map[AttrKey][]DomainRule),
		AttributeRuleList:     make([]DomainRule, 0, 10),
	}
}
