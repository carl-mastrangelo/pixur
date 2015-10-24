package image

import (
	"encoding/binary"
	"image"
	"math"
	"sort"

	"github.com/nfnt/resize"
)

const (
	dctSize = 32
)

func CountBits(num uint64) int {
	num = num - ((num >> 1) & 0x5555555555555555)
	num = (num & 0x3333333333333333) + ((num >> 2) & 0x3333333333333333)
	num = (((num + (num >> 4)) & 0x0f0f0f0f0f0f0f0f) * 0x0101010101010101) >> 56
	return int(num)
}

func PerceptualHash0(im image.Image) ([]byte, []float32) {
	gray := dctResize(im)
	arr := image2Array(gray)
	dcts := dct2d(arr)
	hash, inputs := phash0(dcts)
	outputs := make([]float32, len(inputs))
	for i, input := range inputs {
		outputs[i] = float32(input)
	}
	hashBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(hashBytes, hash)

	return hashBytes, outputs
}

// This approach is riddled with inefficiencies.
func phash0(vals [][]float64) (uint64, []float64) {
	inputs := make([]float64, 0, 64)
	flatvals := make([]float64, 0, len(inputs))
	for y := 1; y < 9; y++ {
		for x := 1; x < 9; x++ {
			inputs = append(inputs, vals[y][x])
			flatvals = append(flatvals, vals[y][x])
		}
	}

	sort.Float64s(flatvals)
	mid := len(flatvals) / 2
	median := (flatvals[mid-1] + flatvals[mid]) / 2
	var hash uint64

	for i, val := range inputs {
		if val > median {
			hash |= 1 << uint(i)
		}
	}

	return hash, inputs
}

func dctResize(im image.Image) *image.Gray {
	small := resize.Resize(dctSize, dctSize, im, resize.Lanczos2)

	// TODO: do colorspace conversion in Lab colorspace
	gray := image.NewGray(small.Bounds())
	for y := gray.Bounds().Min.Y; y < gray.Bounds().Max.Y; y++ {
		for x := gray.Bounds().Min.X; x < gray.Bounds().Max.X; x++ {
			gray.Set(x, y, small.At(x, y))
		}
	}
	return gray
}

func image2Array(im *image.Gray) [][]float64 {
	arr := make([][]float64, im.Bounds().Dy())
	for y := 0; y < len(arr); y++ {
		arr[y] = make([]float64, im.Bounds().Dx())
	}
	for y := 0; y < len(arr); y++ {
		for x := 0; x < len(arr[y]); x++ {
			arr[y][x] = float64(im.GrayAt(x, y).Y) - 128
		}
	}
	return arr
}

// this whole function could be faster.
func dct2d(s [][]float64) [][]float64 {
	n := len(s) // row count
	S := make([][]float64, n)
	for v, d := range s {
		if len(d) != n {
			panic("Non square matrix")
		}
		S[v] = make([]float64, n)
		copy(S[v], d)
	}

	// rows
	for y := 0; y < n; y++ {
		S[y] = dct(S[y])
	}

	transpose(S)
	// columns
	for x := 0; x < n; x++ {
		S[x] = dct(S[x])
	}

	transpose(S)
	return S
}

func transpose(s [][]float64) {
	for i := 0; i < len(s); i++ {
		for k := i + 1; k < len(s); k++ {
			s[i][k], s[k][i] = s[k][i], s[i][k]
		}
	}
}

func dct(s []float64) []float64 {
	n := len(s)
	N := float64(n)
	S := make([]float64, n)

	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			S[k] += s[i] * Cos(i, k, n)
		}
		S[k] *= math.Sqrt(2/N) * C(k)
	}

	return S
}

func C(x int) float64 {
	if x == 0 {
		return 1 / math.Sqrt2
	}
	return 1
}

func Cos(y, v, n int) float64 {
	Y, V, N := float64(y), float64(v), float64(n)
	PI := math.Pi

	return math.Cos((2*Y + 1) * V * PI / (2 * N))
}

func idct2d(S [][]float64) [][]float64 {
	n := len(S) // row count
	N := float64(n)
	s := make([][]float64, n)
	for v, d := range S {
		if len(d) != n {
			panic("Non square matrix")
		}
		s[v] = make([]float64, n)
	}

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var sum float64

			for v := 0; v < n; v++ {
				for u := 0; u < n; u++ {
					sum += S[v][u] * C(v) * C(u) * Cos2(y, x, v, u, n)
				}
			}

			s[y][x] = sum * (2 / N)
		}
	}

	return s
}

func Cos2(y, x, v, u, n int) float64 {
	Y, X, V, U, N := float64(y), float64(x), float64(v), float64(u), float64(n)
	PI := math.Pi

	return math.Cos((2*Y+1)*V*PI/(2*N)) * math.Cos((2*X+1)*U*PI/(2*N))
}
