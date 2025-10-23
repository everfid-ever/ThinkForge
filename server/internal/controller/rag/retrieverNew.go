package rag

import "github.com/everfid-ever/ThinkForge/api/rag"

type ControllerV1 struct{}

func NewV1() rag.IRetrieverV1 {
	return &ControllerV1{}
}
