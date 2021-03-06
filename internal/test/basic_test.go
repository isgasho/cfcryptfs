package test

import (
	"crypto/rand"
	"io"
	mrand "math/rand"
	"os"
	"sync"
	"testing"

	"bytes"

	"github.com/declan94/cfcryptfs/corecrypter"
)

func TestSeqWrite(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}
	fd, err := os.OpenFile(getPath("TestWrite"), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		t.Errorf("Open file (write only) failed: %v", err)
	}
	fd2, err := os.OpenFile(getPath("TestWrite"), os.O_RDONLY, 0600)
	defer fd2.Close()
	if err != nil {
		t.Errorf("Open file (read only) failed: %v", err)
	}
	text, _ := corecrypter.RandomBytes(10240 + 520)
	_, err = fd.Write(text)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	fd.Close()
	text2 := make([]byte, len(text))
	fd2.Read(text2)
	if !bytes.Equal(text, text2) {
		t.Error("Context not matched")
	}
}

func TestRandomWrite(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}
	bs := mrand.Int()%defaultConfig().PlainBS + 1
	cnt := 12
	length := bs * cnt
	text, _ := corecrypter.RandomBytes(length)
	index := mrand.Perm(cnt)
	fd, err := os.OpenFile(getPath("TestRandomWrite"), os.O_WRONLY|os.O_CREATE, 0600)
	for i := 0; i < cnt; i++ {
		if err != nil {
			t.Errorf("Open file (write only) failed: %v", err)
		}
		j := index[i]
		begin := bs * j
		end := bs * (j + 1)
		var part []byte
		if j == cnt-1 {
			part = text[begin:]
		} else {
			part = text[begin:end]
		}
		_, err := fd.WriteAt(part, int64(begin))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
	}
	defer fd.Close()
	fd2, err := os.OpenFile(getPath("TestRandomWrite"), os.O_RDONLY, 0600)
	if err != nil {
		t.Fatalf("Open file (read only) failed: %v", err)
	}
	defer fd2.Close()
	text2 := make([]byte, len(text))
	fd2.ReadAt(text2, 0)
	if !bytes.Equal(text, text2) {
		t.Error("Context not matched")
		// t.Errorf("truth: %v", text)
		// t.Errorf("resul: %v", text2)
	}
}

func TestRewrite(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}
	fd, err := os.OpenFile(getPath("TestRewrite"), os.O_WRONLY|os.O_CREATE, 0600)
	defer fd.Close()
	if err != nil {
		t.Errorf("Open file (write only) failed: %v", err)
	}
	fd2, err := os.OpenFile(getPath("TestRewrite"), os.O_RDONLY, 0600)
	defer fd2.Close()
	if err != nil {
		t.Errorf("Open file (read only) failed: %v", err)
	}
	text, _ := corecrypter.RandomBytes(10240 + 520)
	_, err = fd.Write(text)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	part := text[500:600]
	io.ReadFull(rand.Reader, part)
	_, err = fd.WriteAt(part, 500)
	if err != nil {
		t.Errorf("Write part failed: %v", err)
	}
	text2 := make([]byte, len(text))
	fd2.Read(text2)
	if !bytes.Equal(text, text2) {
		t.Error("Context not matched")
	}
}

func TestEOF(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}
	length := 2048 + 128
	text, _ := corecrypter.RandomBytes(length)
	fd, err := os.OpenFile(getPath("TestRandomWrite"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatalf("Open file (write) failed: %v", err)
	}
	_, err = fd.Write(text)
	if err != nil {
		t.Fatalf("Write file failed: %v", err)
	}
	fd.Close()
	fd2, err := os.OpenFile(getPath("TestRandomWrite"), os.O_RDONLY, 0600)
	if err != nil {
		t.Fatalf("Open file (read) failed: %v", err)
	}
	text2 := make([]byte, length)
	defer fd2.Close()
	n, err := fd2.ReadAt(text2, 1)
	if err != io.EOF {
		t.Fatalf("Not eof error: %v", err)
	}
	if n != length-1 {
		t.Fatalf("Read n: %d, length: %d", n, length)
	}
	if !bytes.Equal(text[1:], text2[:length-1]) {
		t.Error("Context not matched")
		t.Errorf("truth: %v", text[1:])
		t.Errorf("resul: %v", text2[:length-1])
	}
}

func TestParallelWrite(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}

	bs := 545
	cnt := 20
	text, _ := corecrypter.RandomBytes(bs*cnt + bs/2)

	var wg sync.WaitGroup
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fd, err := os.OpenFile(getPath("TestParallelWrite"), os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				t.Errorf("Open file (write only) failed: %v", err)
			}
			defer fd.Close()
			begin := bs * i
			end := bs * (i + 1)
			var part []byte
			if i == cnt-1 {
				part = text[begin:]
			} else {
				part = text[begin:end]
			}
			_, err = fd.WriteAt(part, int64(begin))
			if err != nil {
				t.Errorf("Write failed: %v", err)
			}
		}(i)
	}
	wg.Wait()
	fd2, err := os.OpenFile(getPath("TestParallelWrite"), os.O_RDONLY, 0600)
	if err != nil {
		t.Fatalf("Open file (read only) failed: %v", err)
	}
	defer fd2.Close()
	text2 := make([]byte, len(text))
	fd2.ReadAt(text2, 0)
	if !bytes.Equal(text, text2) {
		t.Error("Context not matched")
		// t.Errorf("truth: %v", text)
		// t.Errorf("resul: %v", text2)
	}
}

func TestMonkeys(t *testing.T) {
	if !fsMounted {
		initMountFs()
		defer umountFs()
	}
	bs := defaultConfig().PlainBS
	paracnt := 10
	maxlen := bs * 4
	maxoffset := bs * 1024

	var wg sync.WaitGroup
	for i := 0; i < paracnt; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fd, err := os.OpenFile(getPath("TestMonkeys"), os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				t.Errorf("Open file (write only) failed: %v", err)
			}
			defer fd.Close()
			fdc, err := os.OpenFile(getCompPath("TestMonkeys"), os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				t.Errorf("Open compare file (write only) failed: %v", err)
			}
			defer fdc.Close()
			for j := 0; j <= i; j++ {
				len := mrand.Int() % maxlen
				offset := mrand.Int() % maxoffset
				cont, _ := corecrypter.RandomBytes(len)
				_, err = fd.WriteAt(cont, int64(offset))
				if err != nil {
					t.Errorf("Write failed: %v", err)
				}
				_, err = fdc.WriteAt(cont, int64(offset))
				if err != nil {
					t.Errorf("Write compare failed: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()
	if diff("TestMonkeys") {
		t.Error("Context not matched")
	}
}
