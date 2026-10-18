package imageproc

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"os/exec"
	"testing"
)

func TestOxipngStripsMetadata(t *testing.T) {
	if _, err := exec.LookPath("oxipng"); err != nil {
		t.Skip("oxipng is required for PNG optimization")
	}
	var input bytes.Buffer
	if err := png.Encode(&input, image.NewNRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	metadata := []byte("RegressionMetadata\x00remove this text")
	chunk := make([]byte, 12+len(metadata))
	binary.BigEndian.PutUint32(chunk, uint32(len(metadata)))
	copy(chunk[4:8], "tEXt")
	copy(chunk[8:], metadata)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	original := input.Bytes()
	data := append(append(append([]byte{}, original[:len(original)-12]...), chunk...), original[len(original)-12:]...)
	optimized, err := compressWithOxipng(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(optimized, []byte("RegressionMetadata")) {
		t.Fatal("optimizer did not strip metadata; check for a silent command fallback")
	}
	if _, err := png.Decode(bytes.NewReader(optimized)); err != nil {
		t.Fatal(err)
	}
}

func TestImagePipeline(t *testing.T) {
	for _, format := range []string{"jpeg", "opaque-png", "transparent-png", "oversized-png"} {
		t.Run(format, func(t *testing.T) {
			if format == "transparent-png" {
				if _, err := exec.LookPath("oxipng"); err != nil {
					t.Skip("oxipng is required for PNG optimization")
				}
			}
			width, height := 800, 800
			if format == "oversized-png" {
				width, height = 4000, 100
			}
			img := image.NewNRGBA(image.Rect(0, 0, width, height))
			rng := rand.New(rand.NewPCG(1, 2))
			for i := 0; i < len(img.Pix); i += 4 {
				img.Pix[i], img.Pix[i+1], img.Pix[i+2] = byte(rng.Uint32()), byte(rng.Uint32()), byte(rng.Uint32())
				img.Pix[i+3] = 255
				if format == "transparent-png" {
					img.Pix[i+3] = 128
				}
			}
			var input bytes.Buffer
			contentType, wantType := "image/png", "image/jpeg"
			var err error
			if format == "jpeg" {
				contentType = "image/jpeg"
				err = jpeg.Encode(&input, img, &jpeg.Options{Quality: 100})
			} else {
				err = png.Encode(&input, img)
			}
			if err != nil {
				t.Fatal(err)
			}
			if input.Len() <= oneMB {
				t.Fatalf("fixture must exercise compression: got %d bytes", input.Len())
			}
			if format == "transparent-png" {
				wantType = "image/png"
			}
			result, err := NewProcessor(84, true, true).Process(input.Bytes(), contentType)
			if err != nil {
				t.Fatal(err)
			}
			if result.ContentType != wantType {
				t.Fatalf("content type = %s, want %s", result.ContentType, wantType)
			}
			decoded, actualFormat, err := image.Decode(bytes.NewReader(result.Data))
			if err != nil {
				t.Fatal(err)
			}
			if "image/"+actualFormat != wantType {
				t.Fatalf("encoded format = %s, want %s", actualFormat, wantType)
			}
			if format == "oversized-png" {
				width, height = 3840, 96
			}
			if decoded.Bounds().Dx() != width || decoded.Bounds().Dy() != height {
				t.Fatalf("dimensions = %v, want %dx%d", decoded.Bounds(), width, height)
			}
			if wantType == "image/png" {
				_, _, _, alpha := decoded.At(0, 0).RGBA()
				if alpha == 65535 {
					t.Fatal("PNG transparency was lost")
				}
			}
		})
	}
}
