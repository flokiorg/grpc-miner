// Copyright 2009 The Go Authors. All rights reserved.
// Copyright (c) 2024 The Flokicoin developers
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sha256 implements the SHA224 and SHA256 hash algorithms as defined
// in FIPS 180-4.
package sha256

import (
	"io"
)

type Hash interface {
	io.Writer
	Sum(b []byte) []byte
	Reset()
	Size() int
	BlockSize() int

	State() ([8]uint32, uint64)
	SetState([8]uint32, uint64)

	SumDouble(b []byte) []byte
}

func init() {
	// crypto.RegisterHash(crypto.SHA224, New224)
	// crypto.RegisterHash(crypto.SHA256, New)
}

// The size of a SHA256 checksum in bytes.
const Size = 32

// The size of a SHA224 checksum in bytes.
const Size224 = 28

// The blocksize of SHA256 and SHA224 in bytes.
const BlockSize = 64

const (
	chunk     = 64
	init0     = 0x6A09E667
	init1     = 0xBB67AE85
	init2     = 0x3C6EF372
	init3     = 0xA54FF53A
	init4     = 0x510E527F
	init5     = 0x9B05688C
	init6     = 0x1F83D9AB
	init7     = 0x5BE0CD19
	init0_224 = 0xC1059ED8
	init1_224 = 0x367CD507
	init2_224 = 0x3070DD17
	init3_224 = 0xF70E5939
	init4_224 = 0xFFC00B31
	init5_224 = 0x68581511
	init6_224 = 0x64F98FA7
	init7_224 = 0xBEFA4FA4
)

// digest represents the partial evaluation of a checksum.
type digest struct {
	h     [8]uint32
	x     [chunk]byte
	nx    int
	len   uint64
	is224 bool // mark if this digest is SHA-224
}

func (d *digest) Reset() {
	if !d.is224 {
		d.h[0] = init0
		d.h[1] = init1
		d.h[2] = init2
		d.h[3] = init3
		d.h[4] = init4
		d.h[5] = init5
		d.h[6] = init6
		d.h[7] = init7
	} else {
		d.h[0] = init0_224
		d.h[1] = init1_224
		d.h[2] = init2_224
		d.h[3] = init3_224
		d.h[4] = init4_224
		d.h[5] = init5_224
		d.h[6] = init6_224
		d.h[7] = init7_224
	}
	d.nx = 0
	d.len = 0
}

func (d *digest) State() ([8]uint32, uint64) {
	return d.h, d.len
}

func (d *digest) SetState(h [8]uint32, l uint64) {
	d.h = h
	d.len = l
}

// New returns a new Hash computing the SHA256 checksum.
func New() Hash {
	d := new(digest)
	d.Reset()
	return d
}

// New224 returns a new Hash computing the SHA224 checksum.
func New224() Hash {
	d := new(digest)
	d.is224 = true
	d.Reset()
	return d
}

func (d *digest) Size() int {
	if !d.is224 {
		return Size
	}
	return Size224
}

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == chunk {
			block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= chunk {
		n := len(p) &^ (chunk - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

func (d0 *digest) Sum(in []byte) []byte {
	// Make a copy of d0 so that caller can keep writing and summing.
	d := *d0
	d.Write(in)
	hash := d.checkSum()
	if d.is224 {
		return hash[:Size224]
	}
	return hash[:]
}

// SumDouble
func (d0 *digest) SumDouble(in []byte) []byte {
	d := *d0
	d.Write(in)
	hash := d.checkSum()
	if d.is224 {
		return hash[:Size224]
	}

	res := Sum256(hash[:])
	return res[:]
}

func (d *digest) checkSum() [Size]byte {
	len := d.len
	// Padding. Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		d.Write(tmp[0 : 56-len%64])
	} else {
		d.Write(tmp[0 : 64+56-len%64])
	}

	// Length in bits.
	len <<= 3
	for i := uint(0); i < 8; i++ {
		tmp[i] = byte(len >> (56 - 8*i))
	}
	d.Write(tmp[0:8])

	if d.nx != 0 {
		panic("d.nx != 0")
	}

	h := d.h[:]
	if d.is224 {
		h = d.h[:7]
	}

	var digest [Size]byte
	for i, s := range h {
		digest[i*4] = byte(s >> 24)
		digest[i*4+1] = byte(s >> 16)
		digest[i*4+2] = byte(s >> 8)
		digest[i*4+3] = byte(s)
	}

	return digest
}

// Sum256 returns the SHA256 checksum of the data.
func Sum256(data []byte) [Size]byte {
	var d digest
	d.Reset()
	d.Write(data)
	return d.checkSum()
}

// Sum224 returns the SHA224 checksum of the data.
func Sum224(data []byte) (sum224 [Size224]byte) {
	var d digest
	d.is224 = true
	d.Reset()
	d.Write(data)
	sum := d.checkSum()
	copy(sum224[:], sum[:Size224])
	return
}

// ZSL BEGIN

// HashCompress is an interface implemented by hash functions supporting Compress
type HashCompress interface {
	Hash
	Compress() []byte
}

// NewCompress returns a new sha256.HashCompress (which embeds Hash) computing the SHA256 checksum without padding.
func NewCompress() HashCompress {
	d := new(digest)
	d.Reset()
	return d
}

// Apply SHA-256 to one input block, excluding the padding step specified in [NIST2015, Section 5.1]
func (d *digest) Compress() []byte {

	if d.len != BlockSize {
		panic("Compress can only be invoked on 64 bytes of input data")
	}

	if d.is224 {
		panic("Compress is not available for sha224")
	}

	var digest [Size]byte
	for i, s := range d.h {
		digest[i*4] = byte(s >> 24)
		digest[i*4+1] = byte(s >> 16)
		digest[i*4+2] = byte(s >> 8)
		digest[i*4+3] = byte(s)
	}

	return digest[:]
}

// ZSL END

func DoubleSum256(data []byte) []byte {
	hash := Sum256(data)
	hash = Sum256(hash[:])
	return hash[:]
}

// Sum256 returns the SHA256 checksum of the data.
func OneSum256(data []byte) []byte {
	var d digest
	d.Reset()
	d.Write(data)
	ret := d.checkSum()
	return ret[:]
}
