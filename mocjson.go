package mocjson

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/bits"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	EOF            = '\x00'
	BeginArray     = '['
	BeginObject    = '{'
	EndArray       = ']'
	EndObject      = '}'
	NameSeparator  = ':'
	ValueSeparator = ','
	Space          = ' '
	HorizontalTab  = '\t'
	LineFeed       = '\n'
	CarriageReturn = '\r'
	QuotationMark  = '"'
	ReverseSolidus = '\\'
	Solidus        = '/'
	Backspace      = '\b'
	FormFeed       = '\f'
)

const (
	is64Bit      = int(^uint(0) >> 63)
	intDigitLen  = 10 + 9*is64Bit
	uintDigitLen = 10 + 10*is64Bit
)

type ByteMask [4]uint64

func matchByteMask(mask ByteMask, b byte) bool {
	return mask[b>>6]&(1<<(b&0x3f)) != 0
}

var whitespaceASCIIMask = ByteMask{
	1<<Space | 1<<HorizontalTab | 1<<LineFeed | 1<<CarriageReturn,
	0,
	0,
	0,
}

func isWhitespace(b byte) bool {
	return matchByteMask(whitespaceASCIIMask, b)
}

var digitByteMask = ByteMask{
	1<<'0' | 1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	0,
	0,
	0,
}

func isDigit(b byte) bool {
	return matchByteMask(digitByteMask, b)
}

var nonZeroDigitByteMask = ByteMask{
	1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	0,
}

func isNonZeroDigit(b byte) bool {
	return matchByteMask(nonZeroDigitByteMask, b)
}

var hexDigitByteMask = ByteMask{
	1<<'0' | 1<<'1' | 1<<'2' | 1<<'3' | 1<<'4' | 1<<'5' | 1<<'6' | 1<<'7' | 1<<'8' | 1<<'9',
	1<<('A'-64) | 1<<('B'-64) | 1<<('C'-64) | 1<<('D'-64) | 1<<('E'-64) | 1<<('F'-64) | 1<<('a'-64) | 1<<('b'-64) | 1<<('c'-64) | 1<<('d'-64) | 1<<('e'-64) | 1<<('f'-64),
	0,
	0,
}

func isHexDigit(b byte) bool {
	return matchByteMask(hexDigitByteMask, b)
}

var hexDigitValueTable = [256]int{
	'0': 0,
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
	'7': 7,
	'8': 8,
	'9': 9,
	'a': 10,
	'b': 11,
	'c': 12,
	'd': 13,
	'e': 14,
	'f': 15,
	'A': 10,
	'B': 11,
	'C': 12,
	'D': 13,
	'E': 14,
	'F': 15,
}

func hexDigitToValue[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](b byte) T {
	return T(hexDigitValueTable[b])
}

func digitToValue[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](b byte) T {
	return T(b - '0')
}

var signByteMask = ByteMask{
	1<<'-' | 1<<'+',
	0,
	0,
	0,
}

var expByteMask = ByteMask{
	0,
	1<<(('e')-64) | 1<<(('E')-64),
	0,
	0,
}

var endOfValueByteMask = ByteMask{
	1<<EOF | 1<<ValueSeparator,
	1<<(EndArray-64) | 1<<(EndObject-64),
}

var endOfStringValueByteMask = ByteMask{
	1<<EOF | 1<<ValueSeparator | 1<<NameSeparator,
	1<<(EndArray-64) | 1<<(EndObject-64),
}

type Chunk uint64

func (c Chunk) String() string {
	return fmt.Sprintf("%q%q%q%q%q%q%q%q", byte(c>>56), byte(c>>48), byte(c>>40), byte(c>>32), byte(c>>24), byte(c>>16), byte(c>>8), byte(c))
}

const (
	ChunkSize           = 8
	OnesChunk           = 0xFFFFFFFFFFFFFFFF
	WhitespaceChunk     = 0x2020202020202020
	TabChunk            = 0x0909090909090909
	CarriageReturnChunk = 0x0d0d0d0d0d0d0d0d
	LineFeedChunk       = 0x0a0a0a0a0a0a0a0a
	NULLChunk           = 'n'<<56 | 'u'<<48 | 'l'<<40 | 'l'<<32
	TrueChunk           = 't'<<56 | 'r'<<48 | 'u'<<40 | 'e'<<32
	FalseChunk          = 'f'<<56 | 'a'<<48 | 'l'<<40 | 's'<<32 | 'e'<<24
)

func NewChunk(b []byte) Chunk {
	return Chunk(binary.BigEndian.Uint64(b))
}

func (c Chunk) MatchBytes(other Chunk) int {
	return bits.LeadingZeros64(uint64(c^other)) >> 3
}

func (c Chunk) WhitespaceCount() int {
	a := (c ^ WhitespaceChunk) & (c ^ TabChunk) & (c ^ CarriageReturnChunk) & (c ^ LineFeedChunk)
	a |= ^(OnesChunk << (uint64(bits.TrailingZeros64(uint64(c))) & OnesChunk))
	return bits.LeadingZeros64(uint64(a)) >> 3
}

func (c Chunk) DigitMask() uint8 {
	// 0-9 ascii
	// 0: 0b00110000
	// 1: 0b00110001
	// 2: 0b00110010
	// 3: 0b00110011
	// 4: 0b00110100
	// 5: 0b00110101
	// 6: 0b00110110
	// 7: 0b00110111
	// 8: 0b00111000
	// 9: 0b00111001

	const (
		mask0 = 0x3030303030303030
		mask1 = 0xF8F8F8F8F8F8F8F8
		mask2 = 0x3838383838383838
		mask3 = 0xFEFEFEFEFEFEFEFE
	)

	is1to7 := ^((c ^ mask0) & mask1)
	is1to7 = is1to7 & (is1to7 >> 1)
	is1to7 = is1to7 & (is1to7 >> 2)
	is1to7 = is1to7 & (is1to7 >> 4)

	is8to9 := ^((c ^ mask2) & mask3)
	is8to9 = is8to9 & (is8to9 >> 1)
	is8to9 = is8to9 & (is8to9 >> 2)
	is8to9 = is8to9 & (is8to9 >> 4)

	// added
	is1to7 |= is8to9
	is1to7 = is1to7 & 0x0101010101010101

	is1to7 = is1to7>>49 |
		is1to7>>42 |
		is1to7>>35 |
		is1to7>>28 |
		is1to7>>21 |
		is1to7>>14 |
		is1to7>>7 |
		is1to7

	return uint8(is1to7)
}

func (c Chunk) HexMask() uint8 {
	// 0-9 A-f a-f ascii
	// 0: 0b00110000
	// 1: 0b00110001
	// 2: 0b00110010
	// 3: 0b00110011
	// 4: 0b00110100
	// 5: 0b00110101
	// 6: 0b00110110
	// 7: 0b00110111
	// 8: 0b00111000
	// 9: 0b00111001
	// A: 0b01000001
	// B: 0b01000010
	// C: 0b01000011
	// D: 0b01000100
	// E: 0b01000101
	// F: 0b01000110
	// a: 0b01100001
	// b: 0b01100010
	// c: 0b01100011
	// d: 0b01100100
	// e: 0b01100101
	// f: 0b01100110

	const (
		mask0 = 0x3030303030303030
		mask1 = 0xF8F8F8F8F8F8F8F8
		mask2 = 0x3838383838383838
		mask3 = 0xFEFEFEFEFEFEFEFE
		mask4 = 0x4040404040404040
		mask5 = 0xF8F8F8F8F8F8F8F8
		mask6 = 0x4747474747474747
	)

	is1to7 := ^((c ^ mask0) & mask1)
	is1to7 = is1to7 & (is1to7 >> 1)
	is1to7 = is1to7 & (is1to7 >> 2)
	is1to7 = is1to7 & (is1to7 >> 4)

	is8to9 := ^((c ^ mask2) & mask3)
	is8to9 = is8to9 & (is8to9 >> 1)
	is8to9 = is8to9 & (is8to9 >> 2)
	is8to9 = is8to9 & (is8to9 >> 4)

	upper := c & 0xDFDFDFDFDFDFDFDF

	isBacktickToG := ^((upper ^ mask4) & mask5)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 1)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 2)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 4)

	isBacktick := ^(upper ^ mask4)
	isBacktick = isBacktick & (isBacktick >> 1)
	isBacktick = isBacktick & (isBacktick >> 2)
	isBacktick = isBacktick & (isBacktick >> 4)

	isG := ^(upper ^ mask6)
	isG = isG & (isG >> 1)
	isG = isG & (isG >> 2)
	isG = isG & (isG >> 4)

	res := (is1to7 | is8to9) | (isBacktickToG ^ isBacktick ^ isG)
	res &= 0x0101010101010101
	res = res>>49 |
		res>>42 |
		res>>35 |
		res>>28 |
		res>>21 |
		res>>14 |
		res>>7 |
		res

	return uint8(res)
}

func (c Chunk) aToFMask() uint8 {
	const (
		mask0 = 0x4040404040404040
		mask1 = 0xF8F8F8F8F8F8F8F8
		maskG = 0x4747474747474747
	)

	// to upper case
	c &= 0xDFDFDFDFDFDFDFDF

	isBacktickToG := ^((c ^ mask0) & mask1)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 1)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 2)
	isBacktickToG = isBacktickToG & (isBacktickToG >> 4)

	isBacktick := ^(c ^ mask0)
	isBacktick = isBacktick & (isBacktick >> 1)
	isBacktick = isBacktick & (isBacktick >> 2)
	isBacktick = isBacktick & (isBacktick >> 4)

	isG := ^(c ^ maskG)
	isG = isG & (isG >> 1)
	isG = isG & (isG >> 2)
	isG = isG & (isG >> 4)

	isAtoF := isBacktickToG ^ isBacktick ^ isG

	isAtoF = isAtoF & 0x0101010101010101

	isAtoF = isAtoF>>49 |
		isAtoF>>42 |
		isAtoF>>35 |
		isAtoF>>28 |
		isAtoF>>21 |
		isAtoF>>14 |
		isAtoF>>7 |
		isAtoF

	return uint8(isAtoF)
}

func (c Chunk) ASCIIMask() uint8 {
	c &= 0x8080808080808080
	c >>= 7
	c = c>>49 |
		c>>42 |
		c>>35 |
		c>>28 |
		c>>21 |
		c>>14 |
		c>>7 |
		c

	return ^uint8(c)
}

func (c Chunk) UTF8Mask() uint8 {
	const (
		mask01    = 0x8080808080808080
		mask02    = 0xC0C0C0C0C0C0C0C0
		mask1     = 0x8080808080808080
		mask21    = 0xC0C0C0C0C0C0C0C0
		mask22    = 0xE0E0E0E0E0E0E0E0
		mask2ng1  = 0xC0C0C0C0C0C0C0C0
		mask2ng2  = 0xFEFEFEFEFEFEFEFE
		mask31    = 0xE0E0E0E0E0E0E0E0
		mask32    = 0xF0F0F0F0F0F0F0F0
		mask3ng11 = 0xE080E080E080E080
		mask3ng12 = 0xFFFEFFFEFFFEFFFE
		mask3ng21 = 0x80E080E080E080E0
		mask3ng22 = 0xFEFFFEFFFEFFFEFF
		mask41    = 0xF0F0F0F0F0F0F0F0
		mask42    = 0xF8F8F8F8F8F8F8F8
		mask4ng11 = 0xF080F080F080F080
		mask4ng12 = 0xFFF0FFF0FFF0FFF0
		mask4ng21 = 0x80F080F080F080F0
		mask4ng22 = 0xF0FFF0FFF0FFF0FF
	)

	m0 := ^((c ^ mask01) & mask02)
	m0 = m0 & (m0 >> 1)
	m0 = m0 & (m0 >> 2)
	m0 = m0 & (m0 >> 4)
	m0 &= 0x0101010101010101
	m0 = m0>>49 |
		m0>>42 |
		m0>>35 |
		m0>>28 |
		m0>>21 |
		m0>>14 |
		m0>>7 |
		m0
	r0 := uint8(m0)

	m1 := (c & mask1) >> 7
	m1 = m1>>49 |
		m1>>42 |
		m1>>35 |
		m1>>28 |
		m1>>21 |
		m1>>14 |
		m1>>7 |
		m1
	r1 := ^uint8(m1)

	m2 := ^((c ^ mask21) & mask22)
	m2 = m2 & (m2 >> 1)
	m2 = m2 & (m2 >> 2)
	m2 = m2 & (m2 >> 4)
	m2 &= 0x0101010101010101
	m2 = m2>>49 |
		m2>>42 |
		m2>>35 |
		m2>>28 |
		m2>>21 |
		m2>>14 |
		m2>>7 |
		m2
	r2 := uint8(m2)

	m2ng := ^((c ^ mask2ng1) & mask2ng2)
	m2ng = m2ng & (m2ng >> 1)
	m2ng = m2ng & (m2ng >> 2)
	m2ng = m2ng & (m2ng >> 4)
	m2ng &= 0x0101010101010101
	m2ng = m2ng>>49 |
		m2ng>>42 |
		m2ng>>35 |
		m2ng>>28 |
		m2ng>>21 |
		m2ng>>14 |
		m2ng>>7 |
		m2ng
	r2ng := uint8(m2ng)
	r2 &= ^r2ng
	r20 := r0 & ^r2ng

	m3 := ^((c ^ mask31) & mask32)
	m3 = m3 & (m3 >> 1)
	m3 = m3 & (m3 >> 2)
	m3 = m3 & (m3 >> 4)
	m3 &= 0x0101010101010101
	m3 = m3>>49 |
		m3>>42 |
		m3>>35 |
		m3>>28 |
		m3>>21 |
		m3>>14 |
		m3>>7 |
		m3
	r3 := uint8(m3)

	m3ng1 := ^((c ^ mask3ng11) & mask3ng12)
	m3ng1 = m3ng1 & (m3ng1 >> 1)
	m3ng1 = m3ng1 & (m3ng1 >> 2)
	m3ng1 = m3ng1 & (m3ng1 >> 4)
	m3ng1 = m3ng1 & (m3ng1 >> 8)
	m3ng1 &= 0x0101010101010101
	m3ng1 = m3ng1>>49 |
		m3ng1>>42 |
		m3ng1>>35 |
		m3ng1>>28 |
		m3ng1>>21 |
		m3ng1>>14 |
		m3ng1>>7 |
		m3ng1
	r3ng1 := uint8(m3ng1)

	m3ng2 := ^((c ^ mask3ng21) & mask3ng22)
	m3ng2 = m3ng2 & (m3ng2 >> 1)
	m3ng2 = m3ng2 & (m3ng2 >> 2)
	m3ng2 = m3ng2 & (m3ng2 >> 4)
	m3ng2 = m3ng2 & (m3ng2 >> 8)
	m3ng2 &= 0x0101010101010101
	m3ng2 = m3ng2>>49 |
		m3ng2>>42 |
		m3ng2>>35 |
		m3ng2>>28 |
		m3ng2>>21 |
		m3ng2>>14 |
		m3ng2>>7 |
		m3ng2
	r3ng2 := uint8(m3ng2)
	r3ng := r3ng1 | r3ng2
	r3 &= ^r3ng
	r30 := r0 & ^r3ng

	m4 := ^((c ^ mask41) & mask42)
	m4 = m4 & (m4 >> 1)
	m4 = m4 & (m4 >> 2)
	m4 = m4 & (m4 >> 4)
	m4 &= 0x0101010101010101
	m4 = m4>>49 |
		m4>>42 |
		m4>>35 |
		m4>>28 |
		m4>>21 |
		m4>>14 |
		m4>>7 |
		m4
	r4 := uint8(m4)

	m4ng1 := ^((c ^ mask4ng11) & mask4ng12)
	m4ng1 = m4ng1 & (m4ng1 >> 1)
	m4ng1 = m4ng1 & (m4ng1 >> 2)
	m4ng1 = m4ng1 & (m4ng1 >> 4)
	m4ng1 = m4ng1 & (m4ng1 >> 8)
	m4ng1 &= 0x0101010101010101
	m4ng1 = m4ng1>>49 |
		m4ng1>>42 |
		m4ng1>>35 |
		m4ng1>>28 |
		m4ng1>>21 |
		m4ng1>>14 |
		m4ng1>>7 |
		m4ng1

	m4ng2 := ^((c ^ mask4ng21) & mask4ng22)
	m4ng2 = m4ng2 & (m4ng2 >> 1)
	m4ng2 = m4ng2 & (m4ng2 >> 2)
	m4ng2 = m4ng2 & (m4ng2 >> 4)
	m4ng2 = m4ng2 & (m4ng2 >> 8)
	m4ng2 &= 0x0101010101010101
	m4ng2 = m4ng2>>49 |
		m4ng2>>42 |
		m4ng2>>35 |
		m4ng2>>28 |
		m4ng2>>21 |
		m4ng2>>14 |
		m4ng2>>7 |
		m4ng2
	r4ng := uint8(m4ng1 | m4ng2)
	r4 &= ^r4ng
	r40 := r0 & ^r4ng

	return r1 |
		(r2&(r20<<1) | (r2>>1)&r20) |
		(r3&(r30<<1)&(r30<<2) | (r3>>1)&r30&(r30<<1) | (r3>>2)&(r30>>1)&r30) |
		(r4&(r40<<1)&(r40<<2)&(r40<<3) | (r4>>1)&r40&(r40<<1)&(r40<<2) | (r4>>2)&(r40>>1)&r40&(r40<<1) | (r4>>3)&(r40>>2)&(r40>>1)&r40)
}

func (c Chunk) UTF8TwoBytesMask() uint8 {
	const (
		mask00   = 0xC0C0C0C0C0C0C0C0
		mask01   = 0xE0E0E0E0E0E0E0E0
		mask0ng0 = 0xC0C0C0C0C0C0C0C0
		mask0ng1 = 0xFEFEFEFEFEFEFEFE
		mask10   = 0x8080808080808080
		mask11   = 0xC0C0C0C0C0C0C0C0
	)

	m0 := ^((c ^ mask00) & mask01)
	m0 = m0 & (m0 >> 1)
	m0 = m0 & (m0 >> 2)
	m0 = m0 & (m0 >> 4)

	m1 := ^((c ^ mask10) & mask11)
	m1 = m1 & (m1 >> 1)
	m1 = m1 & (m1 >> 2)
	m1 = m1 & (m1 >> 4)

	m0ng := ^((c ^ mask0ng0) & mask0ng1)
	m0ng = m0ng & (m0ng >> 1)
	m0ng = m0ng & (m0ng >> 2)
	m0ng = m0ng & (m0ng >> 4)

	res0 := (m0 & ^m0ng) & 0x0101010101010101
	res0 = res0>>49 |
		res0>>42 |
		res0>>35 |
		res0>>28 |
		res0>>21 |
		res0>>14 |
		res0>>7 |
		res0
	r0 := uint8(res0)

	res1 := m1 & 0x0101010101010101
	res1 = res1>>49 |
		res1>>42 |
		res1>>35 |
		res1>>28 |
		res1>>21 |
		res1>>14 |
		res1>>7 |
		res1
	r1 := uint8(res1)

	return r0&(r1<<1) | (r0>>1)&r1
}

func (c Chunk) UTF8ThreeBytesMask() uint8 {
	const (
		mask00   = 0xE0E0E0E0E0E0E0E0
		mask01   = 0xF0F0F0F0F0F0F0F0
		mask10   = 0x8080808080808080
		mask11   = 0xC0C0C0C0C0C0C0C0
		mask0ng0 = 0xE080E080E080E080
		mask0ng1 = 0xFFFEFFFEFFFEFFFE
		mask1ng0 = 0x80E080E080E080E0
		mask1ng1 = 0xFEFFFEFFFEFFFEFF
	)

	m0 := ^((c ^ mask00) & mask01)
	m0 = m0 & (m0 >> 1)
	m0 = m0 & (m0 >> 2)
	m0 = m0 & (m0 >> 4)

	m1 := ^((c ^ mask10) & mask11)
	m1 = m1 & (m1 >> 1)
	m1 = m1 & (m1 >> 2)
	m1 = m1 & (m1 >> 4)

	m0ng := ^((c ^ mask0ng0) & mask0ng1)
	m0ng = m0ng & (m0ng >> 1)
	m0ng = m0ng & (m0ng >> 2)
	m0ng = m0ng & (m0ng >> 4)
	m0ng = m0ng & (m0ng >> 8)

	m1ng := ^((c ^ mask1ng0) & mask1ng1)
	m1ng = m1ng & (m1ng >> 1)
	m1ng = m1ng & (m1ng >> 2)
	m1ng = m1ng & (m1ng >> 4)
	m1ng = m1ng & (m1ng >> 8)

	mng := m0ng | m1ng

	res0 := (m0 & ^mng) & 0x0101010101010101
	res0 = res0>>49 |
		res0>>42 |
		res0>>35 |
		res0>>28 |
		res0>>21 |
		res0>>14 |
		res0>>7 |
		res0
	r0 := uint8(res0)

	res1 := (m1 & ^mng) & 0x0101010101010101
	res1 = res1>>49 |
		res1>>42 |
		res1>>35 |
		res1>>28 |
		res1>>21 |
		res1>>14 |
		res1>>7 |
		res1
	r1 := uint8(res1)

	return r0&(r1<<1)&(r1<<2) | (r0>>1)&r1&(r1<<1) | (r0>>2)&(r1>>1)&r1
}

func (c Chunk) UTF8FourBytesMask() uint8 {
	const (
		mask00   = 0xF0F0F0F0F0F0F0F0
		mask01   = 0xF8F8F8F8F8F8F8F8
		mask10   = 0x8080808080808080
		mask11   = 0xC0C0C0C0C0C0C0C0
		mask0ng0 = 0xF080F080F080F080
		mask0ng1 = 0xFFF0FFF0FFF0FFF0
		mask1ng0 = 0x80F080F080F080F0
		mask1ng1 = 0xF0FFF0FFF0FFF0FF
	)

	m0 := ^((c ^ mask00) & mask01)
	m0 = m0 & (m0 >> 1)
	m0 = m0 & (m0 >> 2)
	m0 = m0 & (m0 >> 4)

	m1 := ^((c ^ mask10) & mask11)
	m1 = m1 & (m1 >> 1)
	m1 = m1 & (m1 >> 2)
	m1 = m1 & (m1 >> 4)

	m0ng := ^((c ^ mask0ng0) & mask0ng1)
	m0ng = m0ng & (m0ng >> 1)
	m0ng = m0ng & (m0ng >> 2)
	m0ng = m0ng & (m0ng >> 4)
	m0ng = m0ng & (m0ng >> 8)

	m1ng := ^((c ^ mask1ng0) & mask1ng1)
	m1ng = m1ng & (m1ng >> 1)
	m1ng = m1ng & (m1ng >> 2)
	m1ng = m1ng & (m1ng >> 4)
	m1ng = m1ng & (m1ng >> 8)

	mng := m0ng | m1ng

	res0 := (m0 & ^mng) & 0x0101010101010101
	res0 = res0>>49 |
		res0>>42 |
		res0>>35 |
		res0>>28 |
		res0>>21 |
		res0>>14 |
		res0>>7 |
		res0
	r0 := uint8(res0)

	res1 := (m1 & ^mng) & 0x0101010101010101
	res1 = res1>>49 |
		res1>>42 |
		res1>>35 |
		res1>>28 |
		res1>>21 |
		res1>>14 |
		res1>>7 |
		res1
	r1 := uint8(res1)

	return r0&(r1<<1)&(r1<<2)&(r1<<3) |
		(r0>>1)&r1&(r1<<1)&(r1<<2) |
		(r0>>2)&(r1>>1)&r1&(r1<<1) |
		(r0>>3)&(r1>>2)&(r1>>1)&r1
}

func (c Chunk) ReverseSolidusMask() uint8 {
	// 0x5c: '\'
	const mask = 0x5c5c5c5c5c5c5c5c
	m := ^(c ^ mask)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) QuotationMarkMask() uint8 {
	// 0x22: '"'
	const mask = 0x2222222222222222
	m := ^(c ^ mask)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedQuotationMarkMask() uint8 {
	// 0x5c: '\'
	// 0x22: '"'
	const (
		mask0 = 0x5c225c225c225c22
		mask1 = 0x225c225c225c225c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedReverseSolidusMask() uint8 {
	// 0x5c: '\'
	const mask = 0x5c5c5c5c5c5c5c5c
	m := ^(c ^ mask)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedSolidusMask() uint8 {
	// 0x5c: '\'
	// 0x2f: '/'
	const (
		mask0 = 0x5c2f5c2f5c2f5c2f
		mask1 = 0x2f5c2f5c2f5c2f5c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedBackspaceMask() uint8 {
	// 0x5c: '\'
	// 0x62: 'b'
	const (
		mask0 = 0x5c625c625c625c62
		mask1 = 0x625c625c625c625c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedFormFeedMask() uint8 {
	// 0x5c: '\'
	// 0x66: 'f'
	const (
		mask0 = 0x5c665c665c665c66
		mask1 = 0x665c665c665c665c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedLineFeedMask() uint8 {
	const (
		mask0 = 0x5c6e5c6e5c6e5c6e
		mask1 = 0x6e5c6e5c6e5c6e5c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedCarriageReturnMask() uint8 {
	const (
		mask0 = 0x5c725c725c725c72
		mask1 = 0x725c725c725c725c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedHorizontalTabMask() uint8 {
	// 0x5c: '\'
	// 0x74: 't'
	const (
		mask0 = 0x5c747c5c747c5c74
		mask1 = 0x747c5c747c5c747c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m
	return uint8(m)
}

func (c Chunk) EscapedUTF16Mask() uint8 {
	// 0x5c: '\'
	// 0x75: 'u'
	const (
		mask0 = 0x5c757c5c757c5c75
		mask1 = 0x757c5c757c5c757c
	)

	m := ^(c ^ mask0) | ^(c ^ mask1)
	m = m & (m >> 1)
	m = m & (m >> 2)
	m = m & (m >> 4)
	m = m & (m >> 8)
	m &= 0x0101010101010101
	m = m>>49 |
		m>>42 |
		m>>35 |
		m>>28 |
		m>>21 |
		m>>14 |
		m>>7 |
		m

	return uint8(m)
}

func (c Chunk) FirstByte() byte {
	return byte(c >> 56)
}

type ChunkScanner struct {
	r io.Reader
	c Chunk
	b [8]byte
}

func NewChunkScanner(r io.Reader) ChunkScanner {
	return ChunkScanner{r: r}
}

func (r *ChunkScanner) Chunk() Chunk {
	return r.c
}

func (r *ChunkScanner) ShiftN(n int) (int, error) {
	nn, err := r.r.Read(r.b[:n])

	c := binary.BigEndian.Uint64(r.b[:])
	c &= ^(OnesChunk >> (nn << 3))
	r.c = r.c<<(n<<3) | Chunk(bits.RotateLeft64(c, n<<3))

	return nn, err
}

type PeekReader struct {
	r   io.Reader
	buf [1]byte
}

func NewPeekReader(r io.Reader) PeekReader {
	return PeekReader{r: r}
}

func (r *PeekReader) readIntoBuf() error {
	if _, err := r.r.Read(r.buf[:]); err != nil {
		return err
	}
	return nil
}

func (r *PeekReader) peeked() bool {
	return r.buf[0] != 0
}

func (r *PeekReader) Peek() (byte, error) {
	if r.peeked() {
		return r.buf[0], nil
	}

	if err := r.readIntoBuf(); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

func (r *PeekReader) Read(b []byte) (int, error) {
	if r.peeked() {
		if len(b) == 0 {
			return 0, nil
		}

		b[0] = r.buf[0]
		r.buf[0] = 0
		if len(b) == 1 {
			return 1, nil
		}

		n, err := r.r.Read(b[1:])
		return n + 1, err
	}

	return r.r.Read(b)
}

func readExpectedByte(r *PeekReader, buf []byte, expected byte) error {
	b, err := r.Peek()
	if err != nil {
		return err
	}
	if b != expected {
		return fmt.Errorf("unexpected byte: %c", b)
	}
	_, _ = r.Read(buf[:1])
	return nil
}

func readExpectedByteMask(r *PeekReader, buf []byte, expected ByteMask) (byte, error) {
	b, err := r.Peek()
	if err != nil {
		return 0, err
	}
	if !matchByteMask(expected, b) {
		return 0, fmt.Errorf("unexpected byte: %c", b)
	}
	_, _ = r.Read(buf[:1])
	return b, nil
}

func peekExpectedByte(r *PeekReader, expected byte) (bool, error) {
	b, err := r.Peek()
	if err != nil {
		if err == io.EOF {
			return expected == 0, nil
		}
		return false, err
	}
	return b == expected, nil
}

func peekExpectedByteMask(r *PeekReader, expected ByteMask) (byte, bool, error) {
	b, err := r.Peek()
	if err != nil {
		if err == io.EOF {
			return 0, matchByteMask(expected, 0), nil
		}
		return 0, false, err
	}
	return b, matchByteMask(expected, b), nil
}

func consumeWhitespace(r *PeekReader) error {
	for {
		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !isWhitespace(b) {
			return nil
		}
		_, _ = r.Read(r.buf[:])
	}
}

func consumeWhitespaceAndPeekExpectedByte(r *PeekReader, expected byte) (bool, error) {
	if err := consumeWhitespace(r); err != nil {
		return false, err
	}
	return peekExpectedByte(r, expected)
}

func consumeWhitespaceAndPeekExpectedByteMask(r *PeekReader, expected ByteMask) (byte, bool, error) {
	if err := consumeWhitespace(r); err != nil {
		return 0, false, err
	}
	return peekExpectedByteMask(r, expected)
}

func readRuneBytes(r *PeekReader, buf []byte) (int, error) {
	_, err := r.Read(buf[:1])
	if err != nil {
		return 0, err
	}

	if !utf8.RuneStart(buf[0]) {
		return 1, fmt.Errorf("invalid utf-8 sequence")
	}

	if buf[0] < utf8.RuneSelf {
		return 1, nil
	}

	idx := 1
	for ; ; idx++ {
		_, err = r.Read(buf[idx : idx+1])
		if err != nil {
			return idx, err
		}

		b, err := r.Peek()
		if err != nil {
			if err == io.EOF {
				break
			}
			return idx + 1, err
		}

		if utf8.RuneStart(b) {
			break
		}
		if idx == utf8.UTFMax-1 {
			return idx + 1, fmt.Errorf("invalid utf-8 sequence")
		}
	}

	if !utf8.Valid(buf[:idx+1]) {
		return idx + 1, fmt.Errorf("invalid utf-8 sequence")
	}

	return idx + 1, nil
}

type Decoder struct {
	buf     []byte
	bufinit [2 << 10]byte
}

func NewDecoder() Decoder {
	ret := Decoder{}
	ret.buf = ret.bufinit[:]
	return ret
}

func ExpectNull(d *Decoder, r *PeekReader) error {
	if _, err := r.Read(d.buf[:4]); err != nil {
		return fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != 'n' || d.buf[1] != 'u' || d.buf[2] != 'l' || d.buf[3] != 'l' {
		return fmt.Errorf("invalid null value")
	}

	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return fmt.Errorf("invalid null value")
	}
	return nil
}

func ExpectNull2(sc *ChunkScanner) error {
	if sc.Chunk().MatchBytes(NULLChunk) < 4 {
		return fmt.Errorf("invalid null value")
	}

	_, err := sc.ShiftN(4)
	if err != nil {
		if err == io.EOF {
			goto CheckSuffix
		}
		return fmt.Errorf("read error: %v", err)
	}

	for {
		c := sc.Chunk().WhitespaceCount()
		if c == 0 {
			goto CheckSuffix
		}

		_, err := sc.ShiftN(c)
		if err != nil {
			if err == io.EOF {
				goto CheckSuffix
			}
			return fmt.Errorf("read error: %v", err)
		}
	}

CheckSuffix:
	if !matchByteMask(endOfValueByteMask, sc.Chunk().FirstByte()) {
		return fmt.Errorf("invalid null value")
	}

	return nil
}

func ExpectBool[T ~bool](d *Decoder, r *PeekReader) (T, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return false, fmt.Errorf("read error: %v", err)
	}
	switch d.buf[0] {
	case 't':
		if _, err := r.Read(d.buf[:3]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'r' || d.buf[1] != 'u' || d.buf[2] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return false, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return false, fmt.Errorf("invalid bool value")
		}
		return true, nil

	case 'f':
		if _, err := r.Read(d.buf[:4]); err != nil {
			return false, fmt.Errorf("read error: %v", err)
		}
		if d.buf[0] != 'a' || d.buf[1] != 'l' || d.buf[2] != 's' || d.buf[3] != 'e' {
			return false, fmt.Errorf("invalid bool value")
		}

		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return false, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return false, fmt.Errorf("invalid bool value")
		}
		return false, nil
	}

	return false, fmt.Errorf("invalid bool value")
}

func ExpectBool2[T ~bool](sc *ChunkScanner) (T, error) {
	var ret bool

	switch sc.Chunk().FirstByte() {
	case 't':
		if sc.Chunk().MatchBytes(TrueChunk) < 4 {
			return false, fmt.Errorf("invalid bool value")
		}
		ret = true

		_, err := sc.ShiftN(4)
		if err != nil {
			if err == io.EOF {
				goto CheckSuffix
			}
			return false, fmt.Errorf("read error: %v", err)
		}

	case 'f':
		if sc.Chunk().MatchBytes(FalseChunk) < 5 {
			return false, fmt.Errorf("invalid bool value")
		}

		_, err := sc.ShiftN(5)
		if err != nil {
			if err == io.EOF {
				goto CheckSuffix
			}
			return false, fmt.Errorf("read error: %v", err)
		}

	default:
		return false, fmt.Errorf("invalid bool value")
	}

	for {
		c := sc.Chunk().WhitespaceCount()
		if c == 0 {
			goto CheckSuffix
		}

		_, err := sc.ShiftN(c)
		if err != nil {
			if err == io.EOF {
				goto CheckSuffix
			}
			return false, fmt.Errorf("read error: %v", err)
		}
	}

CheckSuffix:
	if !matchByteMask(endOfValueByteMask, sc.Chunk().FirstByte()) {
		return false, fmt.Errorf("invalid bool value")
	}

	return T(ret), nil
}

func ExpectString[T ~string](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadStringValueIntoBuf(d, r)
	if err != nil {
		return "", fmt.Errorf("load string value into buf error: %v", err)
	}

	b, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfStringValueByteMask)
	if err != nil {
		return "", fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return "", fmt.Errorf("invalid string value: %c", b)
	}

	return T(d.buf[:idx]), nil
}

func loadStringValueIntoBuf(d *Decoder, r *PeekReader) (int, error) {
	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] != QuotationMark {
		return 0, fmt.Errorf("invalid string value")
	}

	idx := 0

ReadLoop:
	for {
		n, err := readRuneBytes(r, d.buf[idx:])
		if err != nil {
			return 0, fmt.Errorf("read rune error: %v", err)
		}

		if n != 1 {
			idx += n
			continue ReadLoop
		}

		// n == 1
		switch d.buf[idx] {
		case QuotationMark:
			break ReadLoop

		case ReverseSolidus:
			b, err := r.Peek()
			if err != nil {
				return 0, fmt.Errorf("read error: %v", err)
			}

			switch b {
			case QuotationMark, ReverseSolidus, Solidus:
				// can be read as is
				_, _ = r.Read(d.buf[idx : idx+1])
				idx++

			case 'b':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = Backspace
				idx++

			case 'f':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = FormFeed
				idx++

			case 'n':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = LineFeed
				idx++

			case 'r':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = CarriageReturn
				idx++

			case 't':
				_, _ = r.Read(d.buf[idx : idx+1])
				d.buf[idx] = HorizontalTab
				idx++

			case 'u':
				_, _ = r.Read(d.buf[idx : idx+5])
				if !isHexDigit(d.buf[idx+1]) || !isHexDigit(d.buf[idx+2]) || !isHexDigit(d.buf[idx+3]) || !isHexDigit(d.buf[idx+4]) {
					return 0, fmt.Errorf("invalid escape sequence")
				}
				ru := hexDigitToValue[rune](d.buf[idx+1])<<12 | hexDigitToValue[rune](d.buf[idx+2])<<8 | hexDigitToValue[rune](d.buf[idx+3])<<4 | hexDigitToValue[rune](d.buf[idx+4])
				if utf16.IsSurrogate(ru) {
					_, err := r.Read(d.buf[idx : idx+6])
					if err != nil {
						return 0, fmt.Errorf("read error: %v", err)
					}
					if d.buf[idx] != ReverseSolidus || d.buf[idx+1] != 'u' || !isHexDigit(d.buf[idx+2]) || !isHexDigit(d.buf[idx+3]) || !isHexDigit(d.buf[idx+4]) || !isHexDigit(d.buf[idx+5]) {
						return 0, fmt.Errorf("invalid escape sequence")
					}
					ru2 := hexDigitToValue[rune](d.buf[idx+2])<<12 | hexDigitToValue[rune](d.buf[idx+3])<<8 | hexDigitToValue[rune](d.buf[idx+4])<<4 | hexDigitToValue[rune](d.buf[idx+5])
					ru = utf16.DecodeRune(ru, ru2)
					if ru == utf8.RuneError {
						return 0, fmt.Errorf("invalid escape sequence")
					}
				}
				if !utf8.ValidRune(ru) {
					return 0, fmt.Errorf("invalid escape sequence")
				}
				idx += utf8.EncodeRune(d.buf[idx:], ru)

			default:
				return 0, fmt.Errorf("invalid escape sequence")
			}

		default:
			idx += n
		}
	}

	return idx, nil
}

func ExpectInt[T ~int](d *Decoder, r *PeekReader) (T, error) {
	var ret T
	sign := T(1)

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '-' {
		sign = -1
		if _, err := r.Read(d.buf[:1]); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

	if d.buf[0] == '0' {
		// must be exactly int(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid int value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid int value")
	}

	idx := 1
	ret = sign * digitToValue[T](d.buf[0])
	for ; idx < intDigitLen-1; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + sign*digitToValue[T](b)
	}
	if idx == intDigitLen-1 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if sign == 1 {
			if ret > math.MaxInt/10 {
				return 0, fmt.Errorf("int overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret > math.MaxInt-v {
				return 0, fmt.Errorf("int overflow")
			}
			ret += v
		} else {
			if ret < math.MinInt/10 {
				return 0, fmt.Errorf("int overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret < math.MinInt+v {
				return 0, fmt.Errorf("int overflow")
			}
			ret -= v
		}
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid int value")
	}

	return ret, nil
}

func ExpectInt32[T ~int32](d *Decoder, r *PeekReader) (T, error) {
	var ret T
	sign := T(1)

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '-' {
		sign = -1
		if _, err := r.Read(d.buf[:1]); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

	if d.buf[0] == '0' {
		// must be exactly int32(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid int32 value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid int32 value")
	}

	idx := 1
	ret = sign * digitToValue[T](d.buf[0])
	for ; idx < 9; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + sign*digitToValue[T](b)
	}
	if idx == 9 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if sign == 1 {
			if ret > math.MaxInt32/10 {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret > math.MaxInt32-v {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret += v
		} else {
			if ret < math.MinInt32/10 {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret *= 10
			_, _ = r.Read(d.buf[:1])
			v := digitToValue[T](b)
			if ret < math.MinInt32+v {
				return 0, fmt.Errorf("int32 overflow")
			}
			ret -= v
		}
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid int32 value")
	}

	return ret, nil
}

func ExpectUint[T ~uint](d *Decoder, r *PeekReader) (T, error) {
	var ret T

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '0' {
		// must be exactly uint(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid uint value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid uint value")
	}

	idx := 1
	ret = digitToValue[T](d.buf[0])
	for ; idx < uintDigitLen-1; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + digitToValue[T](b)
	}
	if idx == uintDigitLen-1 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if ret > math.MaxUint/10 {
			return 0, fmt.Errorf("uint overflow")
		}
		ret *= 10
		_, _ = r.Read(d.buf[:1])
		v := digitToValue[T](b)
		if ret > math.MaxUint-v {
			return 0, fmt.Errorf("uint overflow")
		}
		ret += v
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid uint value")
	}

	return ret, nil
}

func ExpectUint32[T ~uint32](d *Decoder, r *PeekReader) (T, error) {
	var ret T

	if _, err := r.Read(d.buf[:1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[0] == '0' {
		// must be exactly uint32(0)
		_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
		if err != nil {
			return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
		}
		if !ok {
			return 0, fmt.Errorf("invalid uint32 value")
		}
		return 0, nil
	}
	if !isDigit(d.buf[0]) {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	idx := 1
	ret = digitToValue[T](d.buf[0])
	for ; idx < 9; idx++ {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		_, _ = r.Read(d.buf[:1])
		ret = ret*10 + digitToValue[T](b)
	}
	if idx == 9 {
		b, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			goto ConsumedWhitespace
		}

		if ret > math.MaxUint32/10 {
			return 0, fmt.Errorf("uint32 overflow")
		}
		ret *= 10
		_, _ = r.Read(d.buf[:1])
		v := digitToValue[T](b)
		if ret > math.MaxUint32-v {
			return 0, fmt.Errorf("uint32 overflow")
		}
		ret += v
	}

ConsumedWhitespace:
	_, ok, err := consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	return ret, nil
}

func ExpectUint32_2[T ~uint32](sc *ChunkScanner) (T, error) {
	var ret T

	c := sc.Chunk()
	mask := c.DigitMask()

	if mask == 0 {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	// leading zero is not allowed
	if (mask&0xC0) == 0xC0 && c>>56 == '0' {
		return 0, fmt.Errorf("leading zero is not allowed")
	}

	n := 0
	for ; mask&0x80 != 0; mask <<= 1 {
		c = Chunk(bits.RotateLeft64(uint64(c), 8))
		ret = ret*10 + T(c&0x0F)
		n++
	}
	if _, err := sc.ShiftN(n); err != nil {
		if err == io.EOF {
			goto CheckSuffix
		}
		return 0, fmt.Errorf("read error: %v", err)
	}

	if n == 8 {
		c = sc.Chunk()
		mask := c.DigitMask()

		if mask^0xE0 == 0 {
			return 0, fmt.Errorf("uint32 overflow")
		}
		if mask&0x80 == 0 {
			goto CheckSuffix
		}

		c = Chunk(bits.RotateLeft64(uint64(c), 8))
		ret = ret*10 + T(c&0x0F)

		if mask&0x40 == 0 {
			if _, err := sc.ShiftN(1); err != nil {
				if err == io.EOF {
					goto CheckSuffix
				}
				return 0, fmt.Errorf("read error: %v", err)
			}
		}

		c = Chunk(bits.RotateLeft64(uint64(c), 8))
		hi, lo := bits.Mul32(uint32(ret), 10)
		if hi != 0 {
			return 0, fmt.Errorf("uint32 overflow")
		}
		sum, carry := bits.Add32(lo, uint32(c&0x0F), 0)
		if carry != 0 {
			return 0, fmt.Errorf("uint32 overflow")
		}
		ret = T(sum)

		if _, err := sc.ShiftN(2); err != nil {
			if err == io.EOF {
				goto CheckSuffix
			}
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

CheckSuffix:
	for {
		cnt := sc.Chunk().WhitespaceCount()
		if cnt == 0 {
			break
		}

		_, err := sc.ShiftN(cnt)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("read error: %v", err)
		}
	}

	if !matchByteMask(endOfValueByteMask, sc.Chunk().FirstByte()) {
		return 0, fmt.Errorf("invalid uint32 value")
	}

	return ret, nil
}

func ExpectFloat64[T ~float64](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadNumberValueIntoBuf(d, r)
	if err != nil {
		return 0, fmt.Errorf("load number value into buf error: %v", err)
	}

	// it's too difficult to parse float64 by hand
	ret, err := strconv.ParseFloat(string(d.buf[:idx]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse float64 error: %v", err)
	}
	return T(ret), nil
}

func ExpectFloat32[T ~float32](d *Decoder, r *PeekReader) (T, error) {
	idx, err := loadNumberValueIntoBuf(d, r)
	if err != nil {
		return 0, fmt.Errorf("load number value into buf error: %v", err)
	}

	// it's too difficult to parse float32 by hand
	ret, err := strconv.ParseFloat(string(d.buf[:idx]), 32)
	if err != nil {
		return 0, fmt.Errorf("parse float32 error: %v", err)
	}
	return T(ret), nil
}

func loadNumberValueIntoBuf(d *Decoder, r *PeekReader) (int, error) {
	idx := 0

	// minus if negative
	ok, err := peekExpectedByte(r, '-')
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		_, _ = r.Read(d.buf[:1])
		idx++
	}

	// integer part
	if _, err := r.Read(d.buf[idx : idx+1]); err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}
	if d.buf[idx] == '0' {
		// leading zero is not allowed
		_, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if ok {
			return 0, fmt.Errorf("invalid number value")
		}
	} else if !isDigit(d.buf[idx]) {
		return 0, fmt.Errorf("invalid number value")
	}
	idx++

	// integer part (remaining)
	for {
		_, ok, err := peekExpectedByteMask(r, digitByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if !ok {
			break
		}

		_, _ = r.Read(d.buf[idx : idx+1])
		idx++
	}

	// fraction part
	ok, err = peekExpectedByte(r, '.')
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		// .
		_, _ = r.Read(d.buf[idx : idx+1])
		idx++

		// fist digit
		if _, err := readExpectedByteMask(r, d.buf[idx:idx+1], digitByteMask); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
		idx++

		// remaining digits
		for {
			_, ok, err := peekExpectedByteMask(r, digitByteMask)
			if err != nil {
				return 0, fmt.Errorf("peek error: %v", err)
			}
			if !ok {
				break
			}

			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}
	}

	// exponent part
	_, ok, err = peekExpectedByteMask(r, expByteMask)
	if err != nil {
		return 0, fmt.Errorf("peek error: %v", err)
	}
	if ok {
		// e
		_, _ = r.Read(d.buf[idx : idx+1])
		idx++

		// sign
		_, ok, err := peekExpectedByteMask(r, signByteMask)
		if err != nil {
			return 0, fmt.Errorf("peek error: %v", err)
		}
		if ok {
			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}

		// first digit (required)
		if _, err := readExpectedByteMask(r, d.buf[idx:idx+1], digitByteMask); err != nil {
			return 0, fmt.Errorf("read error: %v", err)
		}
		idx++

		// remaining digits
		for {
			_, ok, err := peekExpectedByteMask(r, digitByteMask)
			if err != nil {
				return 0, fmt.Errorf("peek error: %v", err)
			}
			if !ok {
				break
			}

			_, _ = r.Read(d.buf[idx : idx+1])
			idx++
		}
	}

	_, ok, err = consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return 0, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return 0, fmt.Errorf("invalid number value")
	}

	return idx, nil
}

func ExpectArrayInt[T ~int](d *Decoder, r *PeekReader) ([]T, error) {
	if err := readExpectedByte(r, d.buf[:1], BeginArray); err != nil {
		return nil, fmt.Errorf("read expected byte error: %v", err)
	}

	var ret []T

	ok, err := consumeWhitespaceAndPeekExpectedByte(r, EndArray)
	if err != nil {
		return nil, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if ok {
		_, _ = r.Read(d.buf[:1])
		goto CheckEndOfValue
	}

Loop:
	for {
		v, err := ExpectInt[T](d, r)
		if err != nil {
			return nil, fmt.Errorf("expect int error: %v", err)
		}
		ret = append(ret, v)

		if err := consumeWhitespace(r); err != nil {
			return nil, fmt.Errorf("consume whitespace error: %v", err)
		}

		b, err := r.Peek()
		if err != nil {
			return nil, fmt.Errorf("peek error: %v", err)
		}
		switch b {
		case EndArray:
			break Loop

		case ValueSeparator:
			_, _ = r.Read(d.buf[:1])
			if err := consumeWhitespace(r); err != nil {
				return nil, fmt.Errorf("consume whitespace error: %v", err)
			}

		default:
			return nil, fmt.Errorf("invalid array value")
		}
	}

	if err := readExpectedByte(r, d.buf[:1], EndArray); err != nil {
		return nil, fmt.Errorf("read expected byte error: %v", err)
	}

CheckEndOfValue:
	_, ok, err = consumeWhitespaceAndPeekExpectedByteMask(r, endOfValueByteMask)
	if err != nil {
		return nil, fmt.Errorf("consume whitespace and peek expected byte error: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("invalid array value")
	}

	if ret == nil {
		ret = []T{}
	}

	return ret, nil
}
