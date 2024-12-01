package model

type EnumTyped int

const (
	EnumTypedOne EnumTyped = iota + 1
	EnumTypedTwo
)

const (
	EnumUntypedOne = iota + 1
	EnumUntypedTwo
)
