package lpgen

// 索引字母 I, J, K, L, M, N

type lingoProcessResult struct {
	Tokens []lingoToken
}

func lingoProcessor(extractResult *lingoExtractResult) *lingoProcessResult {
	result := &lingoProcessResult{}
	result.Tokens = extractResult.Tokens
	return result
}
