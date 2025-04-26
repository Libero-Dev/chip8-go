package main

type Opcode uint8

const (
	opcode00E0 Opcode = iota
	opcode00EE
	opcode1NNN
	opcode2NNN
	opcode3XNN
	opcode4XNN
	opcode5XY0
	opcode6XNN
	opcode7XNN
	opcode8XY0
	opcode8XY1
	opcode8XY2
	opcode8XY3
	opcode8XY4
	opcode8XY5
	opcode8XY6
	opcode8XY7
	opcode8XYE
	opcode9XY0
	opcodeANNN
	opcodeBNNN
	opcodeCXNN
	opcodeDXYN
	opcodeEX9E
	opcodeEXA1
	opcodeFX07
	opcodeFX0A
	opcodeFX15
	opcodeFX18
	opcodeFX1E
	opcodeFX29
	opcodeFX33
	opcodeFX55
	opcodeFX65
)
