package main

import (
	"image"
	"image/color"
	"math/rand"
	"os"
	"time"

	"github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/backends/opengl"
)

const (
	RamStart        uint16 = 0x000
	RamGameStart    uint16 = 0x200
	RamGameStartETI uint16 = 0x600
	RamEnd          uint16 = 0xFFF

	CyclesToExecute = 1
)

type (
	Reg8Bit  uint8
	Reg16Bit uint16
)

type Chip8 struct {
	// General Accessible Memory
	MainMemory [0xFFF]byte

	// General Purpose 8-Bit Registers (V0-VF)
	Vx [16]Reg8Bit

	// Memory Address Store Register
	I Reg16Bit

	// Delay Timer Register
	DT Reg8Bit

	// Sound Timer Register
	ST Reg8Bit

	// Program Counter
	PC Reg16Bit

	// Stack Pointer
	SP Reg8Bit

	Stack [16]Reg16Bit

	isStopped bool

	keyPressed [16]bool

	keyJustReleased [16]bool

	screen *opengl.Window

	screenState [32][64]uint8
}

const (
	SCREEN_WIDTH   = 64
	SCREEN_HEIGHT  = 32
	SCALING_FACTOR = 10
)

var colorOff = color.RGBA{0xd1, 0xd4, 0xcd, 255}

func run() {
	cfg := opengl.WindowConfig{
		Title:     "Pixel Rocks!",
		Bounds:    pixel.R(0, 0, SCREEN_WIDTH*SCALING_FACTOR, SCREEN_HEIGHT*SCALING_FACTOR),
		VSync:     false,
		Resizable: false,
	}
	win, err := opengl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	c := NewChip8()
	c.screen = win
	c.LoadDefaultSprites()
	c.LoadRomFile("./pong.rom")
	c.screen.Clear(colorOff)
	win.SetMatrix(pixel.IM.Scaled(pixel.ZV, 1))

	frameDuration := time.Second / 1000
	for !win.Closed() && !c.isStopped {
		start := time.Now()
		c.ExecuteCPU(CyclesToExecute)

		if c.DT > 0 {
			c.DT--
		}

		if c.ST > 0 {
			c.ST--
		}

		win.Update()

		c.handleInput()
		elapsed := time.Since(start)
		if remaining := frameDuration - elapsed; remaining > 0 {
			time.Sleep(remaining)
		}

	}
}

func main() {
	opengl.Run(run)
}

func NewChip8() *Chip8 {
	return &Chip8{}
}

func (c *Chip8) LoadDefaultSprites() {
	copy(c.MainMemory[:RamGameStart], defaultSprites)
}

func (c *Chip8) ExecuteCPU(cyclesToExecute int) {
	for i := 0; i < cyclesToExecute; i++ {
		opcode := c.fetch()
		instruction := c.decode(opcode)
		c.execute(instruction, opcode)
	}
}

func (c *Chip8) handleInput() {
	c.keyPressed = [16]bool{}
	c.keyJustReleased = [16]bool{}

	keyMap := map[pixel.Button]byte{
		pixel.Key1: 0x1, pixel.Key2: 0x2, pixel.Key3: 0x3, pixel.Key4: 0xC,
		pixel.KeyQ: 0x4, pixel.KeyW: 0x5, pixel.KeyE: 0x6, pixel.KeyR: 0xD,
		pixel.KeyA: 0x7, pixel.KeyS: 0x8, pixel.KeyD: 0x9, pixel.KeyF: 0xE,
		pixel.KeyZ: 0xA, pixel.KeyX: 0x0, pixel.KeyC: 0xB, pixel.KeyV: 0xF,

		pixel.KeyUp:    0x2,
		pixel.KeyLeft:  0x4,
		pixel.KeyRight: 0x6,
		pixel.KeyDown:  0x8,
	}

	if c.screen.Pressed(pixel.KeyEscape) {
		os.Exit(0)
	}

	for key, chip8Key := range keyMap {
		if c.screen.Pressed(key) {
			c.keyPressed[chip8Key] = true
		}

		if c.screen.JustReleased(key) {
			c.keyJustReleased[chip8Key] = true
		}
	}
}

func (c *Chip8) LoadRomFile(romFile string) {
	f, err := os.ReadFile(romFile)
	if err != nil {
		panic(err)
	}

	// dump rom into memory at game start position
	copy(c.MainMemory[RamGameStart:RamGameStart+uint16(len(f))], f)

	c.PositionProgramCounter(RamGameStart)
}

func (c *Chip8) PositionProgramCounter(pos uint16) {
	c.PC = Reg16Bit(pos)
}

func (c *Chip8) fetch() uint16 {
	defer func() {
		c.PC += 2
	}()

	return uint16(c.MainMemory[c.PC])<<8 | uint16(c.MainMemory[c.PC+1])
}

func (c *Chip8) decode(opcode uint16) Opcode {
	switch opcode & 0xF000 { // Mask the first 4 bits
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0:
			return opcode00E0
		case 0x00EE:
			return opcode00EE
		}
	case 0x1000:
		return opcode1NNN
	case 0x2000:
		return opcode2NNN
	case 0x3000:
		return opcode3XNN
	case 0x4000:
		return opcode4XNN
	case 0x5000:
		return opcode5XY0
	case 0x6000:
		return opcode6XNN
	case 0x7000:
		return opcode7XNN
	case 0x8000:
		switch opcode & 0x000F {
		case 0x0000:
			return opcode8XY0
		case 0x0001:
			return opcode8XY1
		case 0x0002:
			return opcode8XY2
		case 0x0003:
			return opcode8XY3
		case 0x0004:
			return opcode8XY4
		case 0x0005:
			return opcode8XY5
		case 0x0006:
			return opcode8XY6
		case 0x0007:
			return opcode8XY7
		case 0x000E:
			return opcode8XYE
		}
	case 0x9000:
		return opcode9XY0
	case 0xA000:
		return opcodeANNN
	case 0xB000:
		return opcodeBNNN
	case 0xC000:
		return opcodeCXNN
	case 0xD000:
		return opcodeDXYN
	case 0xE000:
		switch opcode & 0x000F {
		case 0x000E:
			return opcodeEX9E
		case 0x0001:
			return opcodeEXA1
		}
	case 0xF000:
		switch opcode & 0x00FF {
		case 0x0007:
			return opcodeFX07
		case 0x000A:
			return opcodeFX0A
		case 0x0015:
			return opcodeFX15
		case 0x0018:
			return opcodeFX18
		case 0x001E:
			return opcodeFX1E
		case 0x0029:
			return opcodeFX29
		case 0x0033:
			return opcodeFX33
		case 0x0055:
			return opcodeFX55
		case 0x0065:
			return opcodeFX65
		}
	default:
	}

	return opcode00E0 // TODO REMOVE
}

func (c *Chip8) execute(opcode Opcode, opcodeRaw uint16) {
	switch opcode {
	case opcode00E0:
		c.clearScreen()
	case opcode00EE:
		c.exitSubroutine()
	case opcode1NNN:
		c.JumpToAddr(opcodeRaw)
	case opcode2NNN:
		c.callSubroutine(opcodeRaw)
	case opcode3XNN:
		c.checkVxEqlNN(opcodeRaw)
	case opcode4XNN:
		c.checkVxNotEqlNN(opcodeRaw)
	case opcode5XY0:
		c.checkVxEqlVy(opcodeRaw)
	case opcode6XNN:
		c.setVxToNN(opcodeRaw)
	case opcode7XNN:
		c.addAssignToVx(opcodeRaw)
	case opcode8XY0:
		c.setVxToVy(opcodeRaw)
	case opcode8XY1:
		c.bitwiseORAssignVxToVy(opcodeRaw)
	case opcode8XY2:
		c.bitwiseANDAssignVxToVy(opcodeRaw)
	case opcode8XY3:
		c.bitwiseXORAssignVxToVy(opcodeRaw)
	case opcode8XY4:
		c.addAssignVyToVx(opcodeRaw)
	case opcode8XY5:
		c.subAssignVyToVx(opcodeRaw)
	case opcode8XY6:
		c.rightShiftVxBy1(opcodeRaw)
	case opcode8XY7:
		c.setVxToVySubVx(opcodeRaw)
	case opcode8XYE:
		c.leftShiftVxBy1(opcodeRaw)
	case opcode9XY0:
		c.checkVxNotEqlVy(opcodeRaw)
	case opcodeANNN:
		c.setIReg(opcodeRaw)
	case opcodeBNNN:
		c.pcJump(opcodeRaw)
	case opcodeCXNN:
		c.setVxToRand(opcodeRaw)
	case opcodeDXYN:
		c.drawSprite(opcodeRaw)
	case opcodeEX9E:
		c.keyOpEqlCheck(opcodeRaw)
	case opcodeEXA1:
		c.keyOpNotEqlCheck(opcodeRaw)
	case opcodeFX07:
		c.setVxToDelayTimer(opcodeRaw)
	case opcodeFX0A:
		c.setVxToKeyPress(opcodeRaw)
	case opcodeFX15:
		c.setDelayTimerToVx(opcodeRaw)
	case opcodeFX18:
		c.setSoundTimerToVx(opcodeRaw)
	case opcodeFX1E:
		c.addAssignVxToI(opcodeRaw)
	case opcodeFX29:
		c.setIToSpriteAddrVx(opcodeRaw)
	case opcodeFX33:
		c.storeBCDToI(opcodeRaw)
	case opcodeFX55:
		c.regDump(opcodeRaw)
	case opcodeFX65:
		c.regLoad(opcodeRaw)
	}
}

func (c *Chip8) clearScreen() {
	c.screen.Clear(colorOff)
	for i := range c.screenState {
		c.screenState[i] = [64]uint8{}
	}
}

func (c *Chip8) exitSubroutine() {
	if c.SP <= 0 {
		return
	}
	c.PC = c.Stack[c.SP-1]
	c.SP--
}

func (c *Chip8) JumpToAddr(opcode uint16) {
	c.PC = Reg16Bit(opcode & 0x0FFF)
}

// callSubroutine increments the stack pointer, sets current PC to top of stack, sets PC to NNN
func (c *Chip8) callSubroutine(opcode uint16) {
	if c.SP >= 15 {
		return
	}

	c.SP++
	c.Stack[c.SP-1] = c.PC // TODO: MIGHT HAVE TO DO c.SP-1 for index access
	c.PC = Reg16Bit(opcode & 0x0FFF)
}

// checkVxEqlNN skips the next instruction if Vx equals NN
func (c *Chip8) checkVxEqlNN(opcode uint16) {
	if c.Vx[(opcode&0x0F00)>>8] == Reg8Bit(opcode&0x00FF) {
		c.PC += 2 // skip next instruction
	}
}

// checkVxNotEqlNN skips the next instruction if Vx does not equal NN
func (c *Chip8) checkVxNotEqlNN(opcode uint16) {
	if c.Vx[(opcode&0x0F00)>>8] != Reg8Bit(opcode&0x00FF) {
		c.PC += 2 // skip next instruction
	}
}

// checkVxEqualVy skips the next instruction if Vx equals Vy
func (c *Chip8) checkVxEqlVy(opcode uint16) {
	if c.Vx[(opcode&0x0F00)>>8] == c.Vx[(opcode&0x00F0)>>4] {
		c.PC += 2 // skip next instruction
	}
}

func (c *Chip8) setVxToNN(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = Reg8Bit(opcode & 0x00FF)
}

func (c *Chip8) addAssignToVx(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] + Reg8Bit(opcode&0x00FF)
}

func (c *Chip8) setVxToVy(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) bitwiseORAssignVxToVy(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] | c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) bitwiseANDAssignVxToVy(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] & c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) bitwiseXORAssignVxToVy(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] ^ c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) addAssignVyToVx(opcode uint16) {
	if c.Vx[(opcode&0x00F0)>>4] > 0xFF-c.Vx[(opcode&0x0F00)>>8] {
		c.Vx[0xF] = 1
	} else {
		c.Vx[0xF] = 0
	}
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] + c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) subAssignVyToVx(opcode uint16) {
	if c.Vx[(opcode&0x00F0)>>4] > c.Vx[(opcode&0x0F00)>>8] {
		c.Vx[0xF] = 0
	} else {
		c.Vx[0xF] = 1
	}
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] - c.Vx[(opcode&0x00F0)>>4]
}

func (c *Chip8) rightShiftVxBy1(opcode uint16) {
	c.Vx[0xF] = c.Vx[(opcode&0x0F00)>>8] & 0x1
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] >> 1
}

func (c *Chip8) setVxToVySubVx(opcode uint16) {
	if c.Vx[(opcode&0x0F00)>>8] > c.Vx[(opcode&0x00F0)>>4] {
		c.Vx[0xF] = 0
	} else {
		c.Vx[0xF] = 1
	}
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x00F0)>>4] - c.Vx[(opcode&0x0F00)>>8]
}

func (c *Chip8) leftShiftVxBy1(opcode uint16) {
	c.Vx[0xF] = c.Vx[(opcode&0x0F00)>>8] >> 7
	c.Vx[(opcode&0x0F00)>>8] = c.Vx[(opcode&0x0F00)>>8] << 1
}

func (c *Chip8) checkVxNotEqlVy(opcode uint16) {
	if c.Vx[(opcode&0x0F00)>>8] != c.Vx[(opcode&0x00F0)>>4] {
		c.PC = c.PC + 2
	}
}

func (c *Chip8) setIReg(opcode uint16) {
	c.I = Reg16Bit(opcode & 0x0FFF)
}

func (c *Chip8) pcJump(opcode uint16) {
	c.PC = Reg16Bit(c.Vx[0]) + Reg16Bit(opcode&0x0FFF)
}

func (c *Chip8) setVxToRand(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = Reg8Bit(rand.Intn(256)) & Reg8Bit(opcode&0x00FF)
}

func (c *Chip8) drawSprite(opcode uint16) {
	x := c.Vx[(opcode&0x0F00)>>8] % 64
	y := c.Vx[(opcode&0x00F0)>>4] % 32
	h := opcode & 0x000F
	c.Vx[0xF] = 0
	var j uint16 = 0
	var i uint16 = 0
	colorOn := color.RGBA{0x74, 0x8c, 0xab, 255}
	img := image.NewRGBA(image.Rect(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT))

	for j = 0; j < h; j++ {
		pixel := c.MainMemory[uint16(c.I)+j]

		if uint8(y)+uint8(j) >= SCREEN_HEIGHT {
			continue
		}

		for i = 0; i < 8; i++ {
			if uint8(x)+uint8(i) >= SCREEN_WIDTH {
				continue
			}

			if (pixel & (0x80 >> i)) != 0 {
				if c.screenState[(uint8(y) + uint8(j))][uint8(x)+uint8(i)] == 1 {
					c.Vx[0xF] = 1
				}
				c.screenState[(uint8(y) + uint8(j))][uint8(x)+uint8(i)] ^= 1
			}
		}
	}

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if c.screenState[y][x] == 1 {
				img.Set(x, y, colorOn)
			} else {
				img.Set(x, y, colorOff)
			}
		}
	}

	pic := pixel.PictureDataFromImage(img)
	sprite := pixel.NewSprite(pic, pic.Bounds())

	mat := pixel.IM.
		Scaled(pixel.ZV, SCALING_FACTOR).
		Moved(c.screen.Bounds().Center())

	sprite.Draw(c.screen, mat)
}

func (c *Chip8) keyOpEqlCheck(opcode uint16) {
	if c.keyPressed[c.Vx[(opcode&0x0F00)>>8]] {
		c.PC += 2
	}
}

func (c *Chip8) keyOpNotEqlCheck(opcode uint16) {
	if !c.keyPressed[c.Vx[(opcode&0x0F00)>>8]] {
		c.PC += 2
	}
}

func (c *Chip8) setVxToDelayTimer(opcode uint16) {
	c.Vx[(opcode&0x0F00)>>8] = c.DT
}

var emptyBoolSlice [16]bool

func (c *Chip8) setVxToKeyPress(opcode uint16) {
	if c.keyJustReleased == emptyBoolSlice {
		c.PC -= 2
		return
	}
	for i := 0; i < len(c.keyPressed); i++ {
		if c.keyJustReleased[i] {
			c.Vx[(opcode&0x0F00)>>8] = Reg8Bit(i)
		}
	}
}

func (c *Chip8) setDelayTimerToVx(opcode uint16) {
	c.DT = c.Vx[(opcode&0x0F00)>>8]
}

func (c *Chip8) setSoundTimerToVx(opcode uint16) {
	c.ST = c.Vx[(opcode&0x0F00)>>8]
}

func (c *Chip8) addAssignVxToI(opcode uint16) {
	if c.I+Reg16Bit(c.Vx[(opcode&0x0F00)>>8]) > 0xFFF {
		c.Vx[0xF] = 1
	} else {
		c.Vx[0xF] = 0
	}
	c.I = c.I + Reg16Bit(c.Vx[(opcode&0x0F00)>>8])
}

func (c *Chip8) setIToSpriteAddrVx(opcode uint16) {
	switch (opcode >> 8) & 0x0F {
	case 0x00:
		c.I = defaultSprite0Loc
	case 0x01:
		c.I = defaultSprite1Loc
	case 0x02:
		c.I = defaultSprite2Loc
	case 0x03:
		c.I = defaultSprite3Loc
	case 0x04:
		c.I = defaultSprite4Loc
	case 0x05:
		c.I = defaultSprite5Loc
	case 0x06:
		c.I = defaultSprite6Loc
	case 0x07:
		c.I = defaultSprite7Loc
	case 0x08:
		c.I = defaultSprite8Loc
	case 0x09:
		c.I = defaultSprite9Loc
	case 0x0a:
		c.I = defaultSpriteALoc
	case 0x0b:
		c.I = defaultSpriteBLoc
	case 0x0c:
		c.I = defaultSpriteCLoc
	case 0x0d:
		c.I = defaultSpriteDLoc
	case 0x0e:
		c.I = defaultSpriteELoc
	case 0x0f:
		c.I = defaultSpriteFLoc
	}
}

func (c *Chip8) storeBCDToI(opcode uint16) {
	vxIdx := uint8((opcode & 0x0F00) >> 8)
	val := uint8(c.Vx[vxIdx])

	c.MainMemory[c.I] = byte(val / 100)
	c.MainMemory[c.I+1] = byte((val / 10) % 10)
	c.MainMemory[c.I+2] = byte(val % 10)
}

func (c *Chip8) regDump(opcode uint16) {
	var i uint8 = 0
	lastVxReg := uint8((opcode & 0x0F00) >> 8)

	regICopy := c.I

	for i <= lastVxReg {
		c.MainMemory[regICopy] = byte(c.Vx[i])
		regICopy++
		i++
	}
}

func (c *Chip8) regLoad(opcode uint16) {
	var i uint8 = 0
	lastVxReg := uint8((opcode & 0x0F00) >> 8)

	regICopy := c.I

	for i <= lastVxReg {
		c.Vx[i] = Reg8Bit(c.MainMemory[regICopy])
		regICopy++
		i++
	}
}
