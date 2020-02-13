module github.com/AdamSLevy/fct-trackerd

go 1.13

require (
	crawshaw.io/sqlite v0.2.4
	github.com/AdamSLevy/retry v0.0.0-20191017184328-cce921f261f4
	github.com/Factom-Asset-Tokens/base58 v0.0.0-20191118025050-4fa02e92ec20 // indirect
	github.com/Factom-Asset-Tokens/factom v0.0.0-20200212221606-6d5a0a1efb17
	github.com/canonical-ledgers/cryptoprice/v2 v2.0.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/stretchr/testify v1.4.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

replace github.com/canonical-ledgers/cryptoprice/v2 => ../cryptoprice
