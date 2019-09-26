package roaring

import (
	"bytes"
	"log"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/willf/bitset"
)

func TestReverseIteratorCount(t *testing.T) {
	array := []int{2, 63, 64, 65, 4095, 4096, 4097, 4159, 4160, 4161, 5000, 20000, 66666}
	for _, testSize := range array {
		b := New()
		for i := uint32(0); i < uint32(testSize); i++ {
			b.Add(i)
		}
		it := b.ReverseIterator()
		count := 0
		for it.HasNext() {
			it.Next()
			count++
		}

		assert.Equal(t, testSize, count)
	}
}

func TestRoaringIntervalCheck(t *testing.T) {
	r := BitmapOf(1, 2, 3, 1000)
	rangeb := New()
	rangeb.AddRange(10, 1000+1)

	assert.True(t, r.Intersects(rangeb))

	rangeb2 := New()
	rangeb2.AddRange(10, 1000)

	assert.False(t, r.Intersects(rangeb2))
}

func TestRoaringRangeEnd(t *testing.T) {
	r := New()
	r.Add(MaxUint32)
	assert.EqualValues(t, 1, r.GetCardinality())

	r.RemoveRange(0, MaxUint32)
	assert.EqualValues(t, 1, r.GetCardinality())

	r.RemoveRange(0, math.MaxUint64)
	assert.EqualValues(t, 0, r.GetCardinality())

	r.Add(MaxUint32)
	assert.EqualValues(t, 1, r.GetCardinality())

	r.RemoveRange(0, 0x100000001)
	assert.EqualValues(t, 0, r.GetCardinality())

	r.Add(MaxUint32)
	assert.EqualValues(t, 1, r.GetCardinality())

	r.RemoveRange(0, 0x100000000)
	assert.EqualValues(t, 0, r.GetCardinality())
}

func TestFirstLast(t *testing.T) {
	bm := New()
	bm.AddInt(2)
	bm.AddInt(4)
	bm.AddInt(8)

	assert.EqualValues(t, 2, bm.Minimum())
	assert.EqualValues(t, 8, bm.Maximum())

	i := 1 << 5

	for ; i < (1 << 17); i++ {
		bm.AddInt(i)

		assert.EqualValues(t, 2, bm.Minimum())
		assert.EqualValues(t, i, bm.Maximum())
	}

	bm.RunOptimize()

	assert.EqualValues(t, 2, bm.Minimum())
	assert.EqualValues(t, i-1, bm.Maximum())
}

func TestRoaringBitmapBitmapOf(t *testing.T) {
	array := []uint32{5580, 33722, 44031, 57276, 83097}
	bmp := BitmapOf(array...)

	assert.EqualValues(t, len(array), bmp.GetCardinality())

	by, _ := bmp.ToBytes()

	assert.EqualValues(t, len(by), bmp.GetSerializedSizeInBytes())
}

func TestRoaringBitmapAdd(t *testing.T) {
	array := []uint32{5580, 33722, 44031, 57276, 83097}
	bmp := New()
	for _, v := range array {
		bmp.Add(v)
	}

	assert.EqualValues(t, len(array), bmp.GetCardinality())
}

func TestRoaringBitmapAddMany(t *testing.T) {
	array := []uint32{5580, 33722, 44031, 57276, 83097}
	bmp := NewBitmap()
	bmp.AddMany(array)

	assert.EqualValues(t, len(array), bmp.GetCardinality())
}

func TestRoaringBitmapAddOffset(t *testing.T) {
	cases := []struct {
		arr      []uint32
		offset   int64
		expected []uint32
	}{
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   25000,
			expected: []uint32{30580, 58722, 69031, 82276, 108097},
		},
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   -25000,
			expected: []uint32{8722, 19031, 32276, 58097},
		},
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   -83097,
			expected: []uint32{0},
		},
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   MaxUint32,
			expected: []uint32{},
		},
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   -MaxUint32,
			expected: []uint32{},
		},
		{
			arr:      []uint32{5580, 33722, 44031, 57276, 83097},
			offset:   0,
			expected: []uint32{5580, 33722, 44031, 57276, 83097},
		},
	}

	for _, c := range cases {
		bmp := NewBitmap()
		bmp.AddMany(c.arr)

		cop := AddOffset(bmp, c.offset)

		assert.EqualValues(t, len(c.expected), cop.GetCardinality())
		assert.EqualValues(t, c.expected, cop.ToArray())
	}
}

func TestRoaringInPlaceAndNotBitmapContainer(t *testing.T) {
	bm := NewBitmap()
	for i := 0; i < 8192; i++ {
		bm.Add(uint32(i))
	}
	toRemove := NewBitmap()
	for i := 128; i < 8192; i++ {
		toRemove.Add(uint32(i))
	}
	bm.AndNot(toRemove)

	var b bytes.Buffer
	_, err := bm.WriteTo(&b)

	assert.NoError(t, err)

	bm2 := NewBitmap()
	bm2.ReadFrom(bytes.NewBuffer(b.Bytes()))

	assert.True(t, bm2.Equals(bm))
}

// https://github.com/RoaringBitmap/roaring/issues/64
func TestFlip64(t *testing.T) {
	bm := New()
	bm.AddInt(0)
	bm.Flip(1, 2)
	i := bm.Iterator()

	assert.False(t, i.Next() != 0 || i.Next() != 1 || i.HasNext())
}

// https://github.com/RoaringBitmap/roaring/issues/64
func TestFlip64Off(t *testing.T) {
	bm := New()
	bm.AddInt(10)
	bm.Flip(11, 12)
	i := bm.Iterator()

	assert.False(t, i.Next() != 10 || i.Next() != 11 || i.HasNext())
}

func TestStringer(t *testing.T) {
	v := NewBitmap()
	for i := uint32(0); i < 10; i++ {
		v.Add(i)
	}

	assert.Equal(t, "{0,1,2,3,4,5,6,7,8,9}", v.String())

	v.RunOptimize()

	assert.Equal(t, "{0,1,2,3,4,5,6,7,8,9}", v.String())
}

func TestFastCard(t *testing.T) {
	bm := NewBitmap()
	bm.Add(1)
	bm.AddRange(21, 260000)
	bm2 := NewBitmap()
	bm2.Add(25)

	assert.EqualValues(t, 1, bm2.AndCardinality(bm))
	assert.Equal(t, bm.GetCardinality(), bm2.OrCardinality(bm))
	assert.EqualValues(t, 1, bm.AndCardinality(bm2))
	assert.Equal(t, bm.GetCardinality(), bm.OrCardinality(bm2))
	assert.EqualValues(t, 1, bm2.AndCardinality(bm))
	assert.Equal(t, bm.GetCardinality(), bm2.OrCardinality(bm))

	bm.RunOptimize()

	assert.EqualValues(t, 1, bm2.AndCardinality(bm))
	assert.Equal(t, bm.GetCardinality(), bm2.OrCardinality(bm))
	assert.EqualValues(t, 1, bm.AndCardinality(bm2))
	assert.Equal(t, bm.GetCardinality(), bm.OrCardinality(bm2))
	assert.EqualValues(t, 1, bm2.AndCardinality(bm))
	assert.Equal(t, bm.GetCardinality(), bm2.OrCardinality(bm))
}

func TestIntersects1(t *testing.T) {
	bm := NewBitmap()
	bm.Add(1)
	bm.AddRange(21, 26)
	bm2 := NewBitmap()
	bm2.Add(25)

	assert.True(t, bm2.Intersects(bm))

	bm.Remove(25)
	assert.Equal(t, false, bm2.Intersects(bm))

	bm.AddRange(1, 100000)
	assert.True(t, bm2.Intersects(bm))
}

func TestRangePanic(t *testing.T) {
	bm := NewBitmap()
	bm.Add(1)
	bm.AddRange(21, 26)
	bm.AddRange(9, 14)
	bm.AddRange(11, 16)
}

func TestRangeRemoval(t *testing.T) {
	bm := NewBitmap()
	bm.Add(1)
	bm.AddRange(21, 26)
	bm.AddRange(9, 14)
	bm.RemoveRange(11, 16)
	bm.RemoveRange(1, 26)
	c := bm.GetCardinality()

	assert.EqualValues(t, 0, c)

	bm.AddRange(1, 10000)
	c = bm.GetCardinality()

	assert.EqualValues(t, 10000-1, c)

	bm.RemoveRange(1, 10000)
	c = bm.GetCardinality()

	assert.EqualValues(t, 0, c)
}

func TestRangeRemovalFromContent(t *testing.T) {
	bm := NewBitmap()
	for i := 100; i < 10000; i++ {
		bm.AddInt(i * 3)
	}
	bm.AddRange(21, 26)
	bm.AddRange(9, 14)
	bm.RemoveRange(11, 16)
	bm.RemoveRange(0, 30000)
	c := bm.GetCardinality()

	assert.EqualValues(t, 00, c)
}

func TestFlipOnEmpty(t *testing.T) {
	t.Run("TestFlipOnEmpty in-place", func(t *testing.T) {
		bm := NewBitmap()
		bm.Flip(0, 10)
		c := bm.GetCardinality()

		assert.EqualValues(t, 10, c)
	})

	t.Run("TestFlipOnEmpty, generating new result", func(t *testing.T) {
		bm := NewBitmap()
		bm = Flip(bm, 0, 10)
		c := bm.GetCardinality()

		assert.EqualValues(t, 10, c)
	})
}

func TestBitmapRank2(t *testing.T) {
	r := NewBitmap()
	for i := uint32(1); i < 8194; i += 2 {
		r.Add(i)
	}

	rank := r.Rank(63)
	assert.EqualValues(t, 32, rank)
}

func TestBitmapRank(t *testing.T) {
	for N := uint32(1); N <= 1048576; N *= 2 {
		t.Run("rank tests"+strconv.Itoa(int(N)), func(t *testing.T) {
			for gap := uint32(1); gap <= 65536; gap *= 2 {
				rb1 := NewBitmap()
				for x := uint32(0); x <= N; x += gap {
					rb1.Add(x)
				}
				for y := uint32(0); y <= N; y++ {
					if rb1.Rank(y) != uint64((y+1+gap-1)/gap) {
						assert.Equal(t, (y+1+gap-1)/gap, rb1.Rank(y))
					}
				}
			}
		})
	}
}

func TestBitmapSelect(t *testing.T) {
	for N := uint32(1); N <= 1048576; N *= 2 {
		t.Run("rank tests"+strconv.Itoa(int(N)), func(t *testing.T) {
			for gap := uint32(1); gap <= 65536; gap *= 2 {
				rb1 := NewBitmap()
				for x := uint32(0); x <= N; x += gap {
					rb1.Add(x)
				}
				for y := uint32(0); y <= N/gap; y++ {
					expectedInt := y * gap
					i, err := rb1.Select(y)
					if err != nil {
						t.Fatal(err)
					}

					if i != expectedInt {
						assert.Equal(t, expectedInt, i)
					}
				}
			}
		})
	}
}

// some extra tests
func TestBitmapExtra(t *testing.T) {
	for N := uint32(1); N <= 65536; N *= 2 {
		t.Run("extra tests"+strconv.Itoa(int(N)), func(t *testing.T) {
			for gap := uint32(1); gap <= 65536; gap *= 2 {
				bs1 := bitset.New(0)
				rb1 := NewBitmap()
				for x := uint32(0); x <= N; x += gap {
					bs1.Set(uint(x))
					rb1.Add(x)
				}

				assert.EqualValues(t, rb1.GetCardinality(), bs1.Count())
				assert.True(t, equalsBitSet(bs1, rb1))

				for offset := uint32(1); offset <= gap; offset *= 2 {
					bs2 := bitset.New(0)
					rb2 := NewBitmap()
					for x := uint32(0); x <= N; x += gap {
						bs2.Set(uint(x + offset))
						rb2.Add(x + offset)
					}

					assert.EqualValues(t, rb2.GetCardinality(), bs2.Count())
					assert.True(t, equalsBitSet(bs2, rb2))

					clonebs1 := bs1.Clone()
					clonebs1.InPlaceIntersection(bs2)

					if !equalsBitSet(clonebs1, And(rb1, rb2)) {
						v := rb1.Clone()
						v.And(rb2)

						assert.True(t, equalsBitSet(clonebs1, v))
					}

					// testing OR
					clonebs1 = bs1.Clone()
					clonebs1.InPlaceUnion(bs2)

					assert.True(t, equalsBitSet(clonebs1, Or(rb1, rb2)))
					// testing XOR
					clonebs1 = bs1.Clone()
					clonebs1.InPlaceSymmetricDifference(bs2)
					assert.True(t, equalsBitSet(clonebs1, Xor(rb1, rb2)))

					//testing NOTAND
					clonebs1 = bs1.Clone()
					clonebs1.InPlaceDifference(bs2)
					assert.True(t, equalsBitSet(clonebs1, AndNot(rb1, rb2)))
				}
			}
		})
	}
}

func FlipRange(start, end int, bs *bitset.BitSet) {
	for i := start; i < end; i++ {
		bs.Flip(uint(i))
	}
}

func TestBitmap(t *testing.T) {
	t.Run("Test Contains", func(t *testing.T) {
		rbm1 := NewBitmap()
		for k := 0; k < 1000; k++ {
			rbm1.AddInt(17 * k)
		}

		for k := 0; k < 17*1000; k++ {
			assert.Equal(t, (k/17*17 == k), rbm1.ContainsInt(k))
		}
	})

	t.Run("Test Clone", func(t *testing.T) {
		rb1 := NewBitmap()
		rb1.Add(10)

		rb2 := rb1.Clone()
		rb2.Remove(10)

		assert.True(t, rb1.Contains(10))
	})

	t.Run("Test run array not equal", func(t *testing.T) {
		rb := NewBitmap()
		rb2 := NewBitmap()
		rb.AddRange(0, 1<<16)
		for i := 0; i < 10; i++ {
			rb2.AddInt(i)
		}

		assert.EqualValues(t, 1<<16, rb.GetCardinality())
		assert.EqualValues(t, 10, rb2.GetCardinality())
		assert.False(t, rb.Equals(rb2))

		rb.RunOptimize()
		rb2.RunOptimize()

		assert.EqualValues(t, 1<<16, rb.GetCardinality())
		assert.EqualValues(t, 10, rb2.GetCardinality())
		assert.False(t, rb.Equals(rb2))
	})

	t.Run("Test ANDNOT4", func(t *testing.T) {
		rb := NewBitmap()
		rb2 := NewBitmap()

		for i := 0; i < 200000; i += 4 {
			rb2.AddInt(i)
		}
		for i := 200000; i < 400000; i += 14 {
			rb2.AddInt(i)
		}

		off := AndNot(rb2, rb)
		andNotresult := AndNot(rb, rb2)

		assert.True(t, rb.Equals(andNotresult))
		assert.True(t, rb2.Equals(off))

		rb2.AndNot(rb)
		assert.True(t, rb2.Equals(off))
	})

	t.Run("Test AND", func(t *testing.T) {
		rr := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
		}
		rr.Add(100000)
		rr.Add(110000)
		rr2 := NewBitmap()
		rr2.Add(13)
		rrand := And(rr, rr2)
		array := rrand.ToArray()

		assert.Equal(t, 1, len(array))
		assert.EqualValues(t, 13, array[0])

		rr.And(rr2)
		array = rr.ToArray()

		assert.Equal(t, 1, len(array))
		assert.EqualValues(t, 13, array[0])
	})

	t.Run("Test AND 2", func(t *testing.T) {
		rr := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr.AddInt(k)
		}
		for k := 3 * 65536; k < 3*65536+9000; k++ {
			rr.AddInt(k)
		}
		for k := 4 * 65535; k < 4*65535+7000; k++ {
			rr.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+10000; k++ {
			rr.AddInt(k)
		}
		for k := 8 * 65535; k < 8*65535+1000; k++ {
			rr.AddInt(k)
		}
		for k := 9 * 65535; k < 9*65535+30000; k++ {
			rr.AddInt(k)
		}

		rr2 := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr2.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr2.AddInt(k)
		}
		for k := 3*65536 + 2000; k < 3*65536+6000; k++ {
			rr2.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 7 * 65535; k < 7*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 10 * 65535; k < 10*65535+5000; k++ {
			rr2.AddInt(k)
		}
		correct := And(rr, rr2)
		rr.And(rr2)

		assert.True(t, correct.Equals(rr))
	})

	t.Run("Test AND 2", func(t *testing.T) {
		rr := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
		}
		rr.AddInt(100000)
		rr.AddInt(110000)
		rr2 := NewBitmap()
		rr2.AddInt(13)

		rrand := And(rr, rr2)
		array := rrand.ToArray()

		assert.Equal(t, 1, len(array))
		assert.EqualValues(t, 13, array[0])
	})

	t.Run("Test AND 3a", func(t *testing.T) {
		rr := NewBitmap()
		rr2 := NewBitmap()
		for k := 6 * 65536; k < 6*65536+10000; k++ {
			rr.AddInt(k)
		}
		for k := 6 * 65536; k < 6*65536+1000; k++ {
			rr2.AddInt(k)
		}
		result := And(rr, rr2)

		assert.EqualValues(t, 1000, result.GetCardinality())
	})

	t.Run("Test AND 3", func(t *testing.T) {
		var arrayand [11256]uint32
		//393,216
		pos := 0
		rr := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr.AddInt(k)
		}
		for k := 3 * 65536; k < 3*65536+1000; k++ {
			rr.AddInt(k)
		}
		for k := 3*65536 + 1000; k < 3*65536+7000; k++ {
			rr.AddInt(k)
		}
		for k := 3*65536 + 7000; k < 3*65536+9000; k++ {
			rr.AddInt(k)
		}
		for k := 4 * 65536; k < 4*65536+7000; k++ {
			rr.AddInt(k)
		}
		for k := 8 * 65536; k < 8*65536+1000; k++ {
			rr.AddInt(k)
		}
		for k := 9 * 65536; k < 9*65536+30000; k++ {
			rr.AddInt(k)
		}

		rr2 := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr2.AddInt(k)
			arrayand[pos] = uint32(k)
			pos++
		}
		for k := 65536; k < 65536+4000; k++ {
			rr2.AddInt(k)
			arrayand[pos] = uint32(k)
			pos++
		}
		for k := 3*65536 + 1000; k < 3*65536+7000; k++ {
			rr2.AddInt(k)
			arrayand[pos] = uint32(k)
			pos++
		}
		for k := 6 * 65536; k < 6*65536+10000; k++ {
			rr.AddInt(k)
		}
		for k := 6 * 65536; k < 6*65536+1000; k++ {
			rr2.AddInt(k)
			arrayand[pos] = uint32(k)
			pos++
		}

		for k := 7 * 65536; k < 7*65536+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 10 * 65536; k < 10*65536+5000; k++ {
			rr2.AddInt(k)
		}
		rrand := And(rr, rr2)

		arrayres := rrand.ToArray()
		ok := true
		for i := range arrayres {
			if i < len(arrayand) {
				if arrayres[i] != arrayand[i] {
					log.Println(i, arrayres[i], arrayand[i])
					ok = false
				}
			} else {
				log.Println('x', arrayres[i])
				ok = false
			}
		}

		assert.Equal(t, len(arrayres), len(arrayand))
		assert.True(t, ok)

	})

	t.Run("Test AND 4", func(t *testing.T) {
		rb := NewBitmap()
		rb2 := NewBitmap()

		for i := 0; i < 200000; i += 4 {
			rb2.AddInt(i)
		}
		for i := 200000; i < 400000; i += 14 {
			rb2.AddInt(i)
		}
		//TODO: Bitmap.And(bm,bm2)
		andresult := And(rb, rb2)
		off := And(rb2, rb)

		assert.True(t, andresult.Equals(off))
		assert.EqualValues(t, 0, andresult.GetCardinality())

		for i := 500000; i < 600000; i += 14 {
			rb.AddInt(i)
		}
		for i := 200000; i < 400000; i += 3 {
			rb2.AddInt(i)
		}
		andresult2 := And(rb, rb2)

		assert.EqualValues(t, 0, andresult.GetCardinality())
		assert.EqualValues(t, 0, andresult2.GetCardinality())

		for i := 0; i < 200000; i += 4 {
			rb.AddInt(i)
		}
		for i := 200000; i < 400000; i += 14 {
			rb.AddInt(i)
		}

		assert.EqualValues(t, 0, andresult.GetCardinality())

		rc := And(rb, rb2)
		rb.And(rb2)

		assert.Equal(t, rb.GetCardinality(), rc.GetCardinality())
	})

	t.Run("ArrayContainerCardinalityTest", func(t *testing.T) {
		ac := newArrayContainer()
		for k := uint16(0); k < 100; k++ {
			ac.iadd(k)
			assert.EqualValues(t, k+1, ac.getCardinality())
		}
		for k := uint16(0); k < 100; k++ {
			ac.iadd(k)
			assert.EqualValues(t, 100, ac.getCardinality())
		}
	})

	t.Run("or test", func(t *testing.T) {
		rr := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
		}
		rr2 := NewBitmap()
		for k := 4000; k < 8000; k++ {
			rr2.AddInt(k)
		}
		result := Or(rr, rr2)

		assert.Equal(t, rr.GetCardinality()+rr2.GetCardinality(), result.GetCardinality())
	})

	t.Run("basic test", func(t *testing.T) {
		rr := NewBitmap()
		var a [4002]uint32
		pos := 0
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
			a[pos] = uint32(k)
			pos++
		}
		rr.AddInt(100000)
		a[pos] = 100000
		pos++
		rr.AddInt(110000)
		a[pos] = 110000
		pos++
		array := rr.ToArray()
		ok := true
		for i := range a {
			if array[i] != a[i] {
				log.Println("rr : ", array[i], " a : ", a[i])
				ok = false
			}
		}

		assert.Equal(t, len(a), len(array))
		assert.True(t, ok)
	})

	t.Run("BitmapContainerCardinalityTest", func(t *testing.T) {
		ac := newBitmapContainer()
		for k := uint16(0); k < 100; k++ {
			ac.iadd(k)
			assert.EqualValues(t, k+1, ac.getCardinality())
		}
		for k := uint16(0); k < 100; k++ {
			ac.iadd(k)
			assert.EqualValues(t, 100, ac.getCardinality())
		}
	})

	t.Run("BitmapContainerTest", func(t *testing.T) {
		rr := newBitmapContainer()
		rr.iadd(uint16(110))
		rr.iadd(uint16(114))
		rr.iadd(uint16(115))
		var array [3]uint16
		pos := 0
		for itr := rr.getShortIterator(); itr.hasNext(); {
			array[pos] = itr.next()
			pos++
		}

		assert.EqualValues(t, 110, array[0])
		assert.EqualValues(t, 114, array[1])
		assert.EqualValues(t, 115, array[2])
	})

	t.Run("cardinality test", func(t *testing.T) {
		N := 1024
		for gap := 7; gap < 100000; gap *= 10 {
			for offset := 2; offset <= 1024; offset *= 2 {
				rb := NewBitmap()
				for k := 0; k < N; k++ {
					rb.AddInt(k * gap)
					assert.EqualValues(t, k+1, rb.GetCardinality())
				}

				assert.EqualValues(t, N, rb.GetCardinality())

				// check the add of existing values
				for k := 0; k < N; k++ {
					rb.AddInt(k * gap)
					assert.EqualValues(t, N, rb.GetCardinality())
				}

				rb2 := NewBitmap()

				for k := 0; k < N; k++ {
					rb2.AddInt(k * gap * offset)
					assert.EqualValues(t, k+1, rb2.GetCardinality())
				}

				assert.EqualValues(t, N, rb2.GetCardinality())

				for k := 0; k < N; k++ {
					rb2.AddInt(k * gap * offset)
					assert.EqualValues(t, N, rb2.GetCardinality())
				}

				assert.EqualValues(t, N/offset, And(rb, rb2).GetCardinality())
				assert.EqualValues(t, 2*N-2*N/offset, Xor(rb, rb2).GetCardinality())
				assert.EqualValues(t, 2*N-N/offset, Or(rb, rb2).GetCardinality())
			}
		}
	})

	t.Run("clear test", func(t *testing.T) {
		rb := NewBitmap()
		for i := 0; i < 200000; i += 7 {
			// dense
			rb.AddInt(i)
		}
		for i := 200000; i < 400000; i += 177 {
			// sparse
			rb.AddInt(i)
		}

		rb2 := NewBitmap()
		rb3 := NewBitmap()
		for i := 0; i < 200000; i += 4 {
			rb2.AddInt(i)
		}
		for i := 200000; i < 400000; i += 14 {
			rb2.AddInt(i)
		}

		rb.Clear()

		assert.EqualValues(t, 0, rb.GetCardinality())
		assert.NotEqual(t, 0, rb2.GetCardinality())

		rb.AddInt(4)
		rb3.AddInt(4)
		andresult := And(rb, rb2)
		orresult := Or(rb, rb2)

		assert.EqualValues(t, 1, andresult.GetCardinality())
		assert.Equal(t, rb2.GetCardinality(), orresult.GetCardinality())

		for i := 0; i < 200000; i += 4 {
			rb.AddInt(i)
			rb3.AddInt(i)
		}
		for i := 200000; i < 400000; i += 114 {
			rb.AddInt(i)
			rb3.AddInt(i)
		}

		arrayrr := rb.ToArray()
		arrayrr3 := rb3.ToArray()
		ok := true
		for i := range arrayrr {
			if arrayrr[i] != arrayrr3[i] {
				ok = false
			}
		}

		assert.Equal(t, len(arrayrr3), len(arrayrr))
		assert.True(t, ok)
	})

	t.Run("constainer factory ", func(t *testing.T) {
		bc1 := newBitmapContainer()
		bc2 := newBitmapContainer()
		bc3 := newBitmapContainer()
		ac1 := newArrayContainer()
		ac2 := newArrayContainer()
		ac3 := newArrayContainer()

		for i := 0; i < 5000; i++ {
			bc1.iadd(uint16(i * 70))
		}
		for i := 0; i < 5000; i++ {
			bc2.iadd(uint16(i * 70))
		}
		for i := 0; i < 5000; i++ {
			bc3.iadd(uint16(i * 70))
		}
		for i := 0; i < 4000; i++ {
			ac1.iadd(uint16(i * 50))
		}
		for i := 0; i < 4000; i++ {
			ac2.iadd(uint16(i * 50))
		}
		for i := 0; i < 4000; i++ {
			ac3.iadd(uint16(i * 50))
		}

		rbc := ac1.clone().(*arrayContainer).toBitmapContainer()
		assert.True(t, validate(rbc, ac1))

		rbc = ac2.clone().(*arrayContainer).toBitmapContainer()
		assert.True(t, validate(rbc, ac2))

		rbc = ac3.clone().(*arrayContainer).toBitmapContainer()
		assert.True(t, validate(rbc, ac3))
	})

	t.Run("flipTest1 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.Flip(100000, 200000) // in-place on empty bitmap
		rbcard := rb.GetCardinality()

		assert.EqualValues(t, 100000, rbcard)

		bs := bitset.New(20000 - 10000)
		for i := uint(100000); i < 200000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest1A", func(t *testing.T) {
		rb := NewBitmap()
		rb1 := Flip(rb, 100000, 200000)
		rbcard := rb1.GetCardinality()

		assert.EqualValues(t, 100000, rbcard)
		assert.EqualValues(t, 0, rb.GetCardinality())

		bs := bitset.New(0)
		assert.True(t, equalsBitSet(bs, rb))

		for i := uint(100000); i < 200000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb1))
	})

	t.Run("flipTest2", func(t *testing.T) {
		rb := NewBitmap()
		rb.Flip(100000, 100000)
		rbcard := rb.GetCardinality()

		assert.EqualValues(t, 0, rbcard)

		bs := bitset.New(0)
		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest2A", func(t *testing.T) {
		rb := NewBitmap()
		rb1 := Flip(rb, 100000, 100000)

		rb.AddInt(1)
		rbcard := rb1.GetCardinality()

		assert.EqualValues(t, 0, rbcard)
		assert.EqualValues(t, 1, rb.GetCardinality())

		bs := bitset.New(0)
		assert.True(t, equalsBitSet(bs, rb1))

		bs.Set(1)
		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest3A", func(t *testing.T) {
		rb := NewBitmap()
		rb.Flip(100000, 200000) // got 100k-199999
		rb.Flip(100000, 199991) // give back 100k-199990
		rbcard := rb.GetCardinality()

		assert.EqualValues(t, 9, rbcard)

		bs := bitset.New(0)
		for i := uint(199991); i < 200000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest4A", func(t *testing.T) {
		// fits evenly on both ends
		rb := NewBitmap()
		rb.Flip(100000, 200000) // got 100k-199999
		rb.Flip(65536, 4*65536)
		rbcard := rb.GetCardinality()

		// 65536 to 99999 are 1s
		// 200000 to 262143 are 1s: total card

		assert.EqualValues(t, 96608, rbcard)

		bs := bitset.New(0)
		for i := uint(65536); i < 100000; i++ {
			bs.Set(i)
		}
		for i := uint(200000); i < 262144; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest5", func(t *testing.T) {
		// fits evenly on small end, multiple
		// containers
		rb := NewBitmap()
		rb.Flip(100000, 132000)
		rb.Flip(65536, 120000)
		rbcard := rb.GetCardinality()

		// 65536 to 99999 are 1s
		// 120000 to 131999

		assert.EqualValues(t, 46464, rbcard)

		bs := bitset.New(0)
		for i := uint(65536); i < 100000; i++ {
			bs.Set(i)
		}
		for i := uint(120000); i < 132000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTest6", func(t *testing.T) {
		rb := NewBitmap()
		rb1 := Flip(rb, 100000, 132000)
		rb2 := Flip(rb1, 65536, 120000)
		//rbcard := rb2.GetCardinality()

		bs := bitset.New(0)
		for i := uint(65536); i < 100000; i++ {
			bs.Set(i)
		}
		for i := uint(120000); i < 132000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb2))
	})

	t.Run("flipTest6A", func(t *testing.T) {
		rb := NewBitmap()
		rb1 := Flip(rb, 100000, 132000)
		rb2 := Flip(rb1, 99000, 2*65536)
		rbcard := rb2.GetCardinality()

		assert.EqualValues(t, rbcard, 1928)

		bs := bitset.New(0)
		for i := uint(99000); i < 100000; i++ {
			bs.Set(i)
		}
		for i := uint(2 * 65536); i < 132000; i++ {
			bs.Set(i)
		}
		assert.True(t, equalsBitSet(bs, rb2))
	})

	t.Run("flipTest7", func(t *testing.T) {
		// within 1 word, first container
		rb := NewBitmap()
		rb.Flip(650, 132000)
		rb.Flip(648, 651)
		rbcard := rb.GetCardinality()

		// 648, 649, 651-131999

		assert.EqualValues(t, rbcard, 132000-651+2)

		bs := bitset.New(0)
		bs.Set(648)
		bs.Set(649)
		for i := uint(651); i < 132000; i++ {
			bs.Set(i)
		}

		assert.True(t, equalsBitSet(bs, rb))
	})

	t.Run("flipTestBig", func(t *testing.T) {
		numCases := 1000
		rb := NewBitmap()
		bs := bitset.New(0)
		//Random r = new Random(3333);
		checkTime := 2.0

		for i := 0; i < numCases; i++ {
			start := rand.Intn(65536 * 20)
			end := rand.Intn(65536 * 20)
			if rand.Float64() < float64(0.1) {
				end = start + rand.Intn(100)
			}
			rb.Flip(uint64(start), uint64(end))
			if start < end {
				FlipRange(start, end, bs) // throws exception
			}
			// otherwise
			// insert some more ANDs to keep things sparser
			if rand.Float64() < 0.2 {
				mask := NewBitmap()
				mask1 := bitset.New(0)
				startM := rand.Intn(65536 * 20)
				endM := startM + 100000
				mask.Flip(uint64(startM), uint64(endM))
				FlipRange(startM, endM, mask1)
				mask.Flip(0, 65536*20+100000)
				FlipRange(0, 65536*20+100000, mask1)
				rb.And(mask)
				bs.InPlaceIntersection(mask1)
			}
			// see if we can detect incorrectly shared containers
			if rand.Float64() < 0.1 {
				irrelevant := Flip(rb, 10, 100000)
				irrelevant.Flip(5, 200000)
				irrelevant.Flip(190000, 260000)
			}
			if float64(i) > checkTime {
				assert.True(t, equalsBitSet(bs, rb))
				checkTime *= 1.5
			}
		}
	})

	t.Run("ortest", func(t *testing.T) {
		rr := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
		}
		rr.AddInt(100000)
		rr.AddInt(110000)
		rr2 := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr2.AddInt(k)
		}

		rror := Or(rr, rr2)

		array := rror.ToArray()

		rr.Or(rr2)
		arrayirr := rr.ToArray()

		assert.True(t, IntsEquals(array, arrayirr))
	})

	t.Run("ORtest", func(t *testing.T) {
		rr := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr.AddInt(k)
		}
		for k := 3 * 65536; k < 3*65536+9000; k++ {
			rr.AddInt(k)
		}
		for k := 4 * 65535; k < 4*65535+7000; k++ {
			rr.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+10000; k++ {
			rr.AddInt(k)
		}
		for k := 8 * 65535; k < 8*65535+1000; k++ {
			rr.AddInt(k)
		}
		for k := 9 * 65535; k < 9*65535+30000; k++ {
			rr.AddInt(k)
		}

		rr2 := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr2.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr2.AddInt(k)
		}
		for k := 3*65536 + 2000; k < 3*65536+6000; k++ {
			rr2.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 7 * 65535; k < 7*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 10 * 65535; k < 10*65535+5000; k++ {
			rr2.AddInt(k)
		}
		correct := Or(rr, rr2)
		rr.Or(rr2)

		assert.True(t, correct.Equals(rr))
	})

	t.Run("ortest2", func(t *testing.T) {
		arrayrr := make([]uint32, 4000+4000+2)
		pos := 0
		rr := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr.AddInt(k)
			arrayrr[pos] = uint32(k)
			pos++
		}
		rr.AddInt(100000)
		rr.AddInt(110000)
		rr2 := NewBitmap()
		for k := 4000; k < 8000; k++ {
			rr2.AddInt(k)
			arrayrr[pos] = uint32(k)
			pos++
		}

		arrayrr[pos] = 100000
		pos++
		arrayrr[pos] = 110000
		pos++

		rror := Or(rr, rr2)

		arrayor := rror.ToArray()

		assert.True(t, IntsEquals(arrayor, arrayrr))
	})

	t.Run("ortest3", func(t *testing.T) {
		V1 := make(map[int]bool)
		V2 := make(map[int]bool)

		rr := NewBitmap()
		rr2 := NewBitmap()
		for k := 0; k < 4000; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}
		for k := 3500; k < 4500; k++ {
			rr.AddInt(k)
			V1[k] = true
		}
		for k := 4000; k < 65000; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}

		// In the second node of each roaring bitmap, we have two bitmap
		// containers.
		// So, we will check the union between two BitmapContainers
		for k := 65536; k < 65536+10000; k++ {
			rr.AddInt(k)
			V1[k] = true
		}

		for k := 65536; k < 65536+14000; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}

		// In the 3rd node of each Roaring Bitmap, we have an
		// ArrayContainer, so, we will try the union between two
		// ArrayContainers.
		for k := 4 * 65535; k < 4*65535+1000; k++ {
			rr.AddInt(k)
			V1[k] = true
		}

		for k := 4 * 65535; k < 4*65535+800; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}

		// For the rest, we will check if the union will take them in
		// the result
		for k := 6 * 65535; k < 6*65535+1000; k++ {
			rr.AddInt(k)
			V1[k] = true
		}

		for k := 7 * 65535; k < 7*65535+2000; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}

		rror := Or(rr, rr2)
		valide := true

		for _, k := range rror.ToArray() {
			_, found := V1[int(k)]
			if !found {
				valide = false
			}
			V2[int(k)] = true
		}

		for k := range V1 {
			_, found := V2[k]
			if !found {
				valide = false
			}
		}

		assert.True(t, valide)
	})

	t.Run("ortest4", func(t *testing.T) {
		rb := NewBitmap()
		rb2 := NewBitmap()

		for i := 0; i < 200000; i += 4 {
			rb2.AddInt(i)
		}
		for i := 200000; i < 400000; i += 14 {
			rb2.AddInt(i)
		}
		rb2card := rb2.GetCardinality()

		// check or against an empty bitmap
		orresult := Or(rb, rb2)
		off := Or(rb2, rb)

		assert.True(t, orresult.Equals(off))
		assert.Equal(t, orresult.GetCardinality(), rb2card)

		for i := 500000; i < 600000; i += 14 {
			rb.AddInt(i)
		}
		for i := 200000; i < 400000; i += 3 {
			rb2.AddInt(i)
		}
		// check or against an empty bitmap
		orresult2 := Or(rb, rb2)

		assert.Equal(t, orresult.GetCardinality(), rb2card)
		assert.Equal(t, rb2.GetCardinality()+rb.GetCardinality(), orresult2.GetCardinality())

		rb.Or(rb2)
		assert.True(t, rb.Equals(orresult2))
	})

	t.Run("randomTest", func(t *testing.T) {
		rTest(t, 15)
		rTest(t, 1024)
		rTest(t, 4096)
		rTest(t, 65536)
		rTest(t, 65536*16)
	})

	t.Run("SimpleCardinality", func(t *testing.T) {
		N := 512
		gap := 70

		rb := NewBitmap()
		for k := 0; k < N; k++ {
			rb.AddInt(k * gap)
			assert.EqualValues(t, k+1, rb.GetCardinality())
		}

		assert.EqualValues(t, N, rb.GetCardinality())

		for k := 0; k < N; k++ {
			rb.AddInt(k * gap)
			assert.EqualValues(t, N, rb.GetCardinality())
		}
	})

	t.Run("XORtest", func(t *testing.T) {
		rr := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr.AddInt(k)
		}
		for k := 3 * 65536; k < 3*65536+9000; k++ {
			rr.AddInt(k)
		}
		for k := 4 * 65535; k < 4*65535+7000; k++ {
			rr.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+10000; k++ {
			rr.AddInt(k)
		}
		for k := 8 * 65535; k < 8*65535+1000; k++ {
			rr.AddInt(k)
		}
		for k := 9 * 65535; k < 9*65535+30000; k++ {
			rr.AddInt(k)
		}

		rr2 := NewBitmap()
		for k := 4000; k < 4256; k++ {
			rr2.AddInt(k)
		}
		for k := 65536; k < 65536+4000; k++ {
			rr2.AddInt(k)
		}
		for k := 3*65536 + 2000; k < 3*65536+6000; k++ {
			rr2.AddInt(k)
		}
		for k := 6 * 65535; k < 6*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 7 * 65535; k < 7*65535+1000; k++ {
			rr2.AddInt(k)
		}
		for k := 10 * 65535; k < 10*65535+5000; k++ {
			rr2.AddInt(k)
		}

		correct := Xor(rr, rr2)
		rr.Xor(rr2)

		assert.True(t, correct.Equals(rr))
	})

	t.Run("xortest1", func(t *testing.T) {
		V1 := make(map[int]bool)
		V2 := make(map[int]bool)

		rr := NewBitmap()
		rr2 := NewBitmap()
		// For the first 65536: rr2 has a bitmap container, and rr has
		// an array container.
		// We will check the union between a BitmapCintainer and an
		// arrayContainer
		for k := 0; k < 4000; k++ {
			rr2.AddInt(k)
			if k < 3500 {
				V1[k] = true
			}
		}
		for k := 3500; k < 4500; k++ {
			rr.AddInt(k)
		}
		for k := 4000; k < 65000; k++ {
			rr2.AddInt(k)
			if k >= 4500 {
				V1[k] = true
			}
		}

		for k := 65536; k < 65536+30000; k++ {
			rr.AddInt(k)
		}

		for k := 65536; k < 65536+50000; k++ {
			rr2.AddInt(k)
			if k >= 65536+30000 {
				V1[k] = true
			}
		}

		// In the 3rd node of each Roaring Bitmap, we have an
		// ArrayContainer. So, we will try the union between two
		// ArrayContainers.
		for k := 4 * 65535; k < 4*65535+1000; k++ {
			rr.AddInt(k)
			if k >= (4*65535 + 800) {
				V1[k] = true
			}
		}

		for k := 4 * 65535; k < 4*65535+800; k++ {
			rr2.AddInt(k)
		}

		for k := 6 * 65535; k < 6*65535+1000; k++ {
			rr.AddInt(k)
			V1[k] = true
		}

		for k := 7 * 65535; k < 7*65535+2000; k++ {
			rr2.AddInt(k)
			V1[k] = true
		}

		rrxor := Xor(rr, rr2)
		valide := true

		for _, i := range rrxor.ToArray() {
			_, found := V1[int(i)]
			if !found {
				valide = false
			}
			V2[int(i)] = true
		}
		for k := range V1 {
			_, found := V2[k]
			if !found {
				valide = false
			}
		}

		assert.True(t, valide)
	})
}
func TestXORtest4(t *testing.T) {
	t.Run("XORtest 4", func(t *testing.T) {
		rb := NewBitmap()
		rb2 := NewBitmap()
		counter := 0

		for i := 0; i < 200000; i += 4 {
			rb2.AddInt(i)
			counter++
		}

		assert.EqualValues(t, counter, rb2.GetCardinality())

		for i := 200000; i < 400000; i += 14 {
			rb2.AddInt(i)
			counter++
		}

		assert.EqualValues(t, counter, rb2.GetCardinality())

		rb2card := rb2.GetCardinality()
		assert.EqualValues(t, counter, rb2card)

		// check or against an empty bitmap
		xorresult := Xor(rb, rb2)
		assert.EqualValues(t, counter, xorresult.GetCardinality())
		off := Or(rb2, rb)

		assert.EqualValues(t, counter, off.GetCardinality())
		assert.True(t, xorresult.Equals(off))

		assert.Equal(t, xorresult.GetCardinality(), rb2card)
		for i := 500000; i < 600000; i += 14 {
			rb.AddInt(i)
		}
		for i := 200000; i < 400000; i += 3 {
			rb2.AddInt(i)
		}
		// check or against an empty bitmap
		xorresult2 := Xor(rb, rb2)

		assert.EqualValues(t, xorresult.GetCardinality(), rb2card)
		assert.Equal(t, xorresult2.GetCardinality(), rb2.GetCardinality()+rb.GetCardinality())

		rb.Xor(rb2)
		assert.True(t, xorresult2.Equals(rb))
	})
	//need to add the massives
}

func TestNextMany(t *testing.T) {
	count := 70000

	for _, gap := range []uint32{1, 8, 32, 128} {
		expected := make([]uint32, count)
		{
			v := uint32(0)
			for i := range expected {
				expected[i] = v
				v += gap
			}
		}
		bm := BitmapOf(expected...)
		for _, bufSize := range []int{1, 64, 4096, count} {
			buf := make([]uint32, bufSize)
			it := bm.ManyIterator()
			cur := 0
			for n := it.NextMany(buf); n != 0; n = it.NextMany(buf) {
				// much faster tests... (10s -> 5ms)
				if cur+n > count {
					assert.LessOrEqual(t, count, cur+n)
				}

				for i, v := range buf[:n] {
					// much faster tests...
					if v != expected[cur+i] {
						assert.Equal(t, expected[cur+i], v)
					}
				}

				cur += n
			}

			assert.Equal(t, count, cur)
		}
	}
}

func TestBigRandom(t *testing.T) {
	rTest(t, 15)
	rTest(t, 100)
	rTest(t, 512)
	rTest(t, 1023)
	rTest(t, 1025)
	rTest(t, 4095)
	rTest(t, 4096)
	rTest(t, 4097)
	rTest(t, 65536)
	rTest(t, 65536*16)
}

func rTest(t *testing.T, N int) {
	log.Println("rtest N=", N)
	for gap := 1; gap <= 65536; gap *= 2 {
		bs1 := bitset.New(0)
		rb1 := NewBitmap()
		for x := 0; x <= N; x += gap {
			bs1.Set(uint(x))
			rb1.AddInt(x)
		}

		assert.EqualValues(t, rb1.GetCardinality(), bs1.Count())
		assert.True(t, equalsBitSet(bs1, rb1))

		for offset := 1; offset <= gap; offset *= 2 {
			bs2 := bitset.New(0)
			rb2 := NewBitmap()
			for x := 0; x <= N; x += gap {
				bs2.Set(uint(x + offset))
				rb2.AddInt(x + offset)
			}

			assert.EqualValues(t, rb2.GetCardinality(), bs2.Count())
			assert.True(t, equalsBitSet(bs2, rb2))

			clonebs1 := bs1.Clone()
			clonebs1.InPlaceIntersection(bs2)

			if !equalsBitSet(clonebs1, And(rb1, rb2)) {
				v := rb1.Clone()
				v.And(rb2)
				assert.True(t, equalsBitSet(clonebs1, v))
			}

			// testing OR
			clonebs1 = bs1.Clone()
			clonebs1.InPlaceUnion(bs2)

			assert.True(t, equalsBitSet(clonebs1, Or(rb1, rb2)))

			// testing XOR
			clonebs1 = bs1.Clone()
			clonebs1.InPlaceSymmetricDifference(bs2)

			assert.True(t, equalsBitSet(clonebs1, Xor(rb1, rb2)))

			//testing NOTAND
			clonebs1 = bs1.Clone()
			clonebs1.InPlaceDifference(bs2)

			assert.True(t, equalsBitSet(clonebs1, AndNot(rb1, rb2)))
		}
	}
}

func equalsBitSet(a *bitset.BitSet, b *Bitmap) bool {
	for i, e := a.NextSet(0); e; i, e = a.NextSet(i + 1) {
		if !b.ContainsInt(int(i)) {
			return false
		}
	}
	i := b.Iterator()
	for i.HasNext() {
		if !a.Test(uint(i.Next())) {
			return false
		}
	}
	return true
}

func equalsArray(a []int, b *Bitmap) bool {
	if uint64(len(a)) != b.GetCardinality() {
		return false
	}
	for _, x := range a {
		if !b.ContainsInt(x) {
			return false
		}
	}
	return true
}

func IntsEquals(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func validate(bc *bitmapContainer, ac *arrayContainer) bool {
	// Checking the cardinalities of each container

	if bc.getCardinality() != ac.getCardinality() {
		log.Println("cardinality differs")
		return false
	}
	// Checking that the two containers contain the same values
	counter := 0

	for i := bc.NextSetBit(0); i >= 0; i = bc.NextSetBit(i + 1) {
		counter++
		if !ac.contains(uint16(i)) {
			log.Println("content differs")
			log.Println(bc)
			log.Println(ac)
			return false
		}

	}

	// checking the cardinality of the BitmapContainer
	return counter == bc.getCardinality()
}

func TestRoaringArray(t *testing.T) {
	a := newRoaringArray()

	t.Run("Test Init", func(t *testing.T) {
		assert.Equal(t, 0, a.size())
	})

	t.Run("Test Insert", func(t *testing.T) {
		a.appendContainer(0, newArrayContainer(), false)
		assert.Equal(t, 1, a.size())
	})

	t.Run("Test Remove", func(t *testing.T) {
		a.remove(0)
		assert.Equal(t, 0, a.size())
	})

	t.Run("Test popcount Full", func(t *testing.T) {
		res := popcount(uint64(0xffffffffffffffff))
		assert.EqualValues(t, 64, res)
	})

	t.Run("Test popcount Empty", func(t *testing.T) {
		res := popcount(0)
		assert.EqualValues(t, 0, res)
	})

	t.Run("Test popcount 16", func(t *testing.T) {
		res := popcount(0xff00ff)
		assert.EqualValues(t, 16, res)
	})

	t.Run("Test ArrayContainer Add", func(t *testing.T) {
		ar := newArrayContainer()
		ar.iadd(1)

		assert.EqualValues(t, 1, ar.getCardinality())
	})

	t.Run("Test ArrayContainer Add wacky", func(t *testing.T) {
		ar := newArrayContainer()
		ar.iadd(0)
		ar.iadd(5000)

		assert.EqualValues(t, 2, ar.getCardinality())
	})

	t.Run("Test ArrayContainer Add Reverse", func(t *testing.T) {
		ar := newArrayContainer()
		ar.iadd(5000)
		ar.iadd(2048)
		ar.iadd(0)

		assert.EqualValues(t, 3, ar.getCardinality())
	})

	t.Run("Test BitmapContainer Add ", func(t *testing.T) {
		bm := newBitmapContainer()
		bm.iadd(0)

		assert.EqualValues(t, 1, bm.getCardinality())
	})
}

func TestFlipBigA(t *testing.T) {
	numCases := 1000
	bs := bitset.New(0)
	checkTime := 2.0
	rb1 := NewBitmap()
	rb2 := NewBitmap()

	for i := 0; i < numCases; i++ {
		start := rand.Intn(65536 * 20)
		end := rand.Intn(65536 * 20)
		if rand.Float64() < 0.1 {
			end = start + rand.Intn(100)
		}

		if (i & 1) == 0 {
			rb2 = FlipInt(rb1, start, end)
			// tweak the other, catch bad sharing
			rb1.FlipInt(rand.Intn(65536*20), rand.Intn(65536*20))
		} else {
			rb1 = FlipInt(rb2, start, end)
			rb2.FlipInt(rand.Intn(65536*20), rand.Intn(65536*20))
		}

		if start < end {
			FlipRange(start, end, bs) // throws exception
		}
		// otherwise
		// insert some more ANDs to keep things sparser
		if (rand.Float64() < 0.2) && (i&1) == 0 {
			mask := NewBitmap()
			mask1 := bitset.New(0)
			startM := rand.Intn(65536 * 20)
			endM := startM + 100000
			mask.FlipInt(startM, endM)
			FlipRange(startM, endM, mask1)
			mask.FlipInt(0, 65536*20+100000)
			FlipRange(0, 65536*20+100000, mask1)
			rb2.And(mask)
			bs.InPlaceIntersection(mask1)
		}

		if float64(i) > checkTime {
			var rb *Bitmap

			if (i & 1) == 0 {
				rb = rb2
			} else {
				rb = rb1
			}

			assert.True(t, equalsBitSet(bs, rb))
			checkTime *= 1.5
		}
	}
}

func TestNextManyOfAddRangeAcrossContainers(t *testing.T) {
	rb := NewBitmap()
	rb.AddRange(65530, 65540)
	expectedCard := 10
	expected := []uint32{65530, 65531, 65532, 65533, 65534, 65535, 65536, 65537, 65538, 65539, 0}

	// test where all values can be returned in a single buffer
	it := rb.ManyIterator()
	buf := make([]uint32, 11)
	n := it.NextMany(buf)

	assert.Equal(t, expectedCard, n)

	for i, e := range expected {
		assert.Equal(t, e, buf[i])
	}

	// test where buf is size 1, so many iterations
	it = rb.ManyIterator()
	n = 0
	buf = make([]uint32, 1)

	for i := 0; i < expectedCard; i++ {
		n = it.NextMany(buf)

		assert.Equal(t, 1, n)
		assert.Equal(t, expected[i], buf[0])
	}

	n = it.NextMany(buf)
	assert.Equal(t, 0, n)
}

func TestDoubleAdd(t *testing.T) {
	t.Run("doubleadd ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536)
		rb.AddRange(65530, 65536)
		rb2 := NewBitmap()
		rb2.AddRange(65530, 65536)

		assert.True(t, rb.Equals(rb2))

		rb2.RemoveRange(65530, 65536)

		assert.EqualValues(t, 0, rb2.GetCardinality())
	})

	t.Run("doubleadd2 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20)
		rb.AddRange(65530, 65536*20)
		rb2 := NewBitmap()
		rb2.AddRange(65530, 65536*20)

		assert.True(t, rb.Equals(rb2))

		rb2.RemoveRange(65530, 65536*20)

		assert.EqualValues(t, 0, rb2.GetCardinality())
	})

	t.Run("doubleadd3 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20+10)
		rb.AddRange(65530, 65536*20+10)
		rb2 := NewBitmap()
		rb2.AddRange(65530, 65536*20+10)

		assert.True(t, rb.Equals(rb2))

		rb2.RemoveRange(65530, 65536*20+1)

		assert.EqualValues(t, 9, rb2.GetCardinality())
	})

	t.Run("doubleadd4 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20)
		rb.RemoveRange(65533+5, 65536*20)

		assert.EqualValues(t, 5, rb.GetCardinality())
	})

	t.Run("doubleadd5 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20)
		rb.RemoveRange(65533+5, 65536*20-5)

		assert.EqualValues(t, 10, rb.GetCardinality())
	})

	t.Run("doubleadd6 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20-5)
		rb.RemoveRange(65533+5, 65536*20-10)

		assert.EqualValues(t, 10, rb.GetCardinality())
	})

	t.Run("doubleadd7 ", func(t *testing.T) {
		rb := NewBitmap()
		rb.AddRange(65533, 65536*20+1)
		rb.RemoveRange(65533+1, 65536*20)

		assert.EqualValues(t, 2, rb.GetCardinality())
	})

	t.Run("AndNotBug01 ", func(t *testing.T) {
		rb1 := NewBitmap()
		rb1.AddRange(0, 60000)
		rb2 := NewBitmap()
		rb2.AddRange(60000-10, 60000+10)
		rb2.AndNot(rb1)
		rb3 := NewBitmap()
		rb3.AddRange(60000, 60000+10)

		assert.True(t, rb2.Equals(rb3))
	})
}

func TestAndNot(t *testing.T) {
	rr := NewBitmap()

	for k := 4000; k < 4256; k++ {
		rr.AddInt(k)
	}
	for k := 65536; k < 65536+4000; k++ {
		rr.AddInt(k)
	}
	for k := 3 * 65536; k < 3*65536+9000; k++ {
		rr.AddInt(k)
	}
	for k := 4 * 65535; k < 4*65535+7000; k++ {
		rr.AddInt(k)
	}
	for k := 6 * 65535; k < 6*65535+10000; k++ {
		rr.AddInt(k)
	}
	for k := 8 * 65535; k < 8*65535+1000; k++ {
		rr.AddInt(k)
	}
	for k := 9 * 65535; k < 9*65535+30000; k++ {
		rr.AddInt(k)
	}

	rr2 := NewBitmap()

	for k := 4000; k < 4256; k++ {
		rr2.AddInt(k)
	}
	for k := 65536; k < 65536+4000; k++ {
		rr2.AddInt(k)
	}
	for k := 3*65536 + 2000; k < 3*65536+6000; k++ {
		rr2.AddInt(k)
	}
	for k := 6 * 65535; k < 6*65535+1000; k++ {
		rr2.AddInt(k)
	}
	for k := 7 * 65535; k < 7*65535+1000; k++ {
		rr2.AddInt(k)
	}
	for k := 10 * 65535; k < 10*65535+5000; k++ {
		rr2.AddInt(k)
	}

	correct := AndNot(rr, rr2)
	rr.AndNot(rr2)

	assert.True(t, correct.Equals(rr))
}

func TestStats(t *testing.T) {
	t.Run("Test Stats with empty bitmap", func(t *testing.T) {
		expectedStats := Statistics{}
		rr := NewBitmap()

		assert.EqualValues(t, expectedStats, rr.Stats())
	})

	t.Run("Test Stats with bitmap Container", func(t *testing.T) {
		// Given a bitmap that should have a single bitmap container
		expectedStats := Statistics{
			Cardinality: 60000,
			Containers:  1,

			BitmapContainers:      1,
			BitmapContainerValues: 60000,
			BitmapContainerBytes:  8192,

			RunContainers:      0,
			RunContainerBytes:  0,
			RunContainerValues: 0,
		}

		rr := NewBitmap()

		for i := uint32(0); i < 60000; i++ {
			rr.Add(i)
		}

		assert.EqualValues(t, expectedStats, rr.Stats())
	})

	t.Run("Test Stats with run Container", func(t *testing.T) {
		// Given that we should have a single run container
		intSize := int(unsafe.Sizeof(int(0)))
		var runContainerBytes uint64
		if intSize == 4 {
			runContainerBytes = 40
		} else {
			runContainerBytes = 52
		}

		expectedStats := Statistics{
			Cardinality: 60000,
			Containers:  1,

			BitmapContainers:      0,
			BitmapContainerValues: 0,
			BitmapContainerBytes:  0,

			RunContainers:      1,
			RunContainerBytes:  runContainerBytes,
			RunContainerValues: 60000,
		}

		rr := NewBitmap()
		rr.AddRange(0, 60000)

		assert.EqualValues(t, expectedStats, rr.Stats())
	})

	t.Run("Test Stats with Array Container", func(t *testing.T) {
		// Given a bitmap that should have a single array container
		expectedStats := Statistics{
			Cardinality: 2,
			Containers:  1,

			ArrayContainers:      1,
			ArrayContainerValues: 2,
			ArrayContainerBytes:  4,
		}
		rr := NewBitmap()
		rr.Add(2)
		rr.Add(4)

		assert.EqualValues(t, expectedStats, rr.Stats())
	})
}

func TestFlipVerySmall(t *testing.T) {
	rb := NewBitmap()
	rb.Flip(0, 10) // got [0,9], card is 10
	rb.Flip(0, 1)  // give back the number 0, card goes to 9
	rbcard := rb.GetCardinality()

	assert.EqualValues(t, 9, rbcard)
}

func TestReverseIterator(t *testing.T) {
	t.Run("#1", func(t *testing.T) {
		values := []uint32{0, 2, 15, 16, 31, 32, 33, 9999, MaxUint16, MaxUint32}
		bm := New()
		for n := 0; n < len(values); n++ {
			bm.Add(values[n])
		}
		i := bm.ReverseIterator()
		n := len(values) - 1

		for i.HasNext() {
			assert.EqualValues(t, i.Next(), values[n])
			n--
		}

		// HasNext() was terminating early - add test
		i = bm.ReverseIterator()
		n = len(values) - 1
		for ; n >= 0; n-- {
			assert.EqualValues(t, i.Next(), values[n])
			assert.False(t, n > 0 && !i.HasNext())
		}
	})

	t.Run("#2", func(t *testing.T) {
		bm := New()
		i := bm.ReverseIterator()

		assert.False(t, i.HasNext())
	})

	t.Run("#3", func(t *testing.T) {
		bm := New()
		bm.AddInt(0)
		i := bm.ReverseIterator()

		assert.True(t, i.HasNext())
		assert.EqualValues(t, 0, i.Next())
		assert.False(t, i.HasNext())
	})

	t.Run("#4", func(t *testing.T) {
		bm := New()
		bm.AddInt(9999)
		i := bm.ReverseIterator()

		assert.True(t, i.HasNext())
		assert.EqualValues(t, 9999, i.Next())
		assert.False(t, i.HasNext())
	})

	t.Run("#5", func(t *testing.T) {
		bm := New()
		bm.AddInt(MaxUint16)
		i := bm.ReverseIterator()

		assert.True(t, i.HasNext())
		assert.EqualValues(t, MaxUint16, i.Next())
		assert.False(t, i.HasNext())
	})

	t.Run("#6", func(t *testing.T) {
		bm := New()
		bm.Add(MaxUint32)
		i := bm.ReverseIterator()

		assert.True(t, i.HasNext())
		assert.EqualValues(t, uint32(MaxUint32), i.Next())
		assert.False(t, i.HasNext())
	})
}

func TestIteratorPeekNext(t *testing.T) {
	values := []uint32{0, 2, 15, 16, 31, 32, 33, 9999, MaxUint16, MaxUint32}
	bm := New()

	for n := 0; n < len(values); n++ {
		bm.Add(values[n])
	}

	i := bm.Iterator()
	assert.True(t, i.HasNext())

	for i.HasNext() {
		assert.Equal(t, i.PeekNext(), i.Next())
	}
}

func TestIteratorAdvance(t *testing.T) {
	values := []uint32{1, 2, 15, 16, 31, 32, 33, 9999, MaxUint16}
	bm := New()

	for n := 0; n < len(values); n++ {
		bm.Add(values[n])
	}

	cases := []struct {
		minval   uint32
		expected uint32
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 15},
		{30, 31},
		{33, 33},
		{9998, 9999},
		{MaxUint16, MaxUint16},
	}

	t.Run("advance by using a new int iterator", func(t *testing.T) {
		for _, c := range cases {
			i := bm.Iterator()
			i.AdvanceIfNeeded(c.minval)

			assert.True(t, i.HasNext())
			assert.Equal(t, c.expected, i.PeekNext())
		}
	})

	t.Run("advance by using the same int iterator", func(t *testing.T) {
		i := bm.Iterator()

		for _, c := range cases {
			i.AdvanceIfNeeded(c.minval)

			assert.True(t, i.HasNext())
			assert.Equal(t, c.expected, i.PeekNext())
		}
	})

	t.Run("advance out of a container value", func(t *testing.T) {
		i := bm.Iterator()

		i.AdvanceIfNeeded(MaxUint32)
		assert.False(t, i.HasNext())

		i.AdvanceIfNeeded(MaxUint32)
		assert.False(t, i.HasNext())
	})

	t.Run("advance on a value that is less than the pointed value", func(t *testing.T) {
		i := bm.Iterator()
		i.AdvanceIfNeeded(29)

		assert.True(t, i.HasNext())
		assert.EqualValues(t, 31, i.PeekNext())

		i.AdvanceIfNeeded(13)

		assert.True(t, i.HasNext())
		assert.EqualValues(t, 31, i.PeekNext())
	})
}

func TestPackageFlipMaxRangeEnd(t *testing.T) {
	var empty Bitmap
	flipped := Flip(&empty, 0, MaxRange)

	assert.EqualValues(t, MaxRange, flipped.GetCardinality())
}

func TestBitmapFlipMaxRangeEnd(t *testing.T) {
	var bm Bitmap
	bm.Flip(0, MaxRange)

	assert.EqualValues(t, MaxRange, bm.GetCardinality())
}
