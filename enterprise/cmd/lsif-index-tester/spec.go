package main

type Position struct {
	Line      int64 `json:"line"`
	Character int64 `json:"character"`
}

type DefinitionRequest struct {
	TextDocument string   `json:"textDocument"`
	Position     Position `json:"position"`
}

type DefinitionResponse struct{}

type DefinitionTest struct {
	Request  DefinitionRequest  `json:"request"`
	Response DefinitionResponse `json:"response"`
}

type LsifTest struct {
	Definitions map[string]DefinitionTest `json:"textDocument/definition"`
}
