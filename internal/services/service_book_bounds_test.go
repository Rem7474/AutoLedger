package services

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"runtime"
	"testing"
)

// craftedPNG builds an RGBA PNG whose header declares width x height and whose single IDAT inflates to inflated zero bytes.
func craftedPNG(t *testing.T, width, height uint32, inflated int) []byte {
	t.Helper()
	chunk := func(kind string, data []byte) []byte {
		var b bytes.Buffer
		_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
		b.WriteString(kind)
		b.Write(data)
		_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(kind), data...)))
		return b.Bytes()
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], width)
	binary.BigEndian.PutUint32(ihdr[4:], height)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	var idat bytes.Buffer
	zw := zlib.NewWriter(&idat)
	zeros := make([]byte, 1<<20)
	for left := inflated; left > 0; left -= len(zeros) {
		n := len(zeros)
		if left < n {
			n = left
		}
		if _, err := zw.Write(zeros[:n]); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	out.WriteString("\x89PNG\r\n\x1a\n")
	out.Write(chunk("IHDR", ihdr))
	out.Write(chunk("IDAT", idat.Bytes()))
	out.Write(chunk("IEND", nil))
	return out.Bytes()
}

func renderWithProof(t *testing.T, data []byte) []byte {
	t.Helper()
	d := bookData(t)
	d.Attachments["doc-1"] = ServiceBookAttachment{Filename: "proof.png", MimeType: "image/png", Data: data}
	out, err := RenderServiceBook(d)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestServiceBookEmbedsAValidPNGProof(t *testing.T) {
	if !bytes.Contains(renderWithProof(t, testPNG(t)), []byte("/Subtype /Image")) {
		t.Fatal("a valid proof must be embedded")
	}
}

func TestServiceBookSkipsAPNGDeclaringAHugePicture(t *testing.T) {
	out := renderWithProof(t, craftedPNG(t, 40000, 40000, 1<<10))
	if bytes.Contains(out, []byte("/Subtype /Image")) {
		t.Fatal("an oversized picture must be replaced by the note")
	}
}

func TestServiceBookDoesNotInflateAPNGBeyondItsHeader(t *testing.T) {
	bomb := craftedPNG(t, 100, 100, 64<<20) // 40 kB of real pixel data, 64 MiB once inflated
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	out := renderWithProof(t, bomb)
	runtime.ReadMemStats(&after)
	if bytes.Contains(out, []byte("/Subtype /Image")) {
		t.Fatal("a stream carrying more data than its header declares must not be embedded")
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 24<<20 {
		t.Fatalf("rendering allocated %d MiB for a 100x100 proof", grown>>20)
	}
}
