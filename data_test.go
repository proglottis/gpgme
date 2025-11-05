package gpgme

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestData_memory_empty(t *testing.T) {
	dh, err := NewData()
	checkError(t, err)

	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testCipherText))
		checkError(t, err)
	}

	_, err = dh.Seek(0, SeekSet)
	checkError(t, err)

	testReader(t, dh, bytes.Repeat([]byte(testCipherText), 5))

	checkError(t, dh.Close())
}

func TestData_memory(t *testing.T) {
	// Test ordinary data, and empty slices
	for _, content := range [][]byte{[]byte(testCipherText), []byte{}} {
		dh, err := NewDataBytes(content)
		checkError(t, err)

		testReader(t, dh, content)

		checkError(t, dh.Close())
	}
}

func TestData_file(t *testing.T) {
	f, err := ioutil.TempFile("", "gpgme")
	checkError(t, err)
	defer func() {
		checkError(t, f.Close())
		checkError(t, os.Remove(f.Name()))
	}()

	dh, err := NewDataFile(f)
	checkError(t, err)

	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testCipherText))
		checkError(t, err)
	}

	_, err = dh.Seek(0, SeekSet)
	checkError(t, err)

	testReader(t, dh, bytes.Repeat([]byte(testCipherText), 5))

	checkError(t, dh.Close())
}

func TestData_callback_reading(t *testing.T) {
	r := bytes.NewReader([]byte(testCipherText))
	dh, err := NewDataReader(r)
	checkError(t, err)

	testReader(t, dh, []byte(testCipherText))

	checkError(t, dh.Close())
}

func TestData_callback_reading_error(t *testing.T) {
	expectedErr := errors.New("a special error")
	r := errReadSeeker{err: expectedErr}
	dh, err := NewDataReader(r)
	checkError(t, err)

	_, err = dh.Read(make([]byte, 10))
	if err != expectedErr {
		t.Errorf("err = %v, want %v", err, expectedErr)
	}

	checkError(t, dh.Close())
}

func TestData_callback_seeking_error(t *testing.T) {
	expectedErr := errors.New("a special error")
	r := errReadSeeker{err: expectedErr}
	dh, err := NewDataReader(r)
	checkError(t, err)

	_, err = dh.Seek(0, 0)
	if err != expectedErr {
		t.Errorf("err = %v, want %v", err, expectedErr)
	}

	checkError(t, dh.Close())
}

func TestData_callback_writing(t *testing.T) {
	var buf bytes.Buffer
	dh, err := NewDataWriter(&buf)
	checkError(t, err)

	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testCipherText))
		checkError(t, err)
	}

	expected := bytes.Repeat([]byte(testCipherText), 5)
	diff(t, buf.Bytes(), expected)

	checkError(t, dh.Close())
}

func TestData_callback_writing_error(t *testing.T) {
	expectedErr := errors.New("a special error")
	dh, err := NewDataWriter(errWriter{err: expectedErr})
	checkError(t, err)

	_, err = dh.Write([]byte(testData))
	if err != expectedErr {
		t.Errorf("err = %v, want %v", err, expectedErr)
	}

	checkError(t, dh.Close())
}

func TestData_callback_writing_short(t *testing.T) {
	shortWriter := &invalidShortWriter{maxWrite: 3}
	dh, err := NewDataWriter(shortWriter)
	checkError(t, err)
	defer dh.Close()

	// Write should loop and write all data despite short writes
	_, err = dh.Write([]byte("test data"))
	checkError(t, err)

	expected := []byte("test data")
	diff(t, shortWriter.written, expected)
}

// invalidShortWriter is an INVALID io.Writer implementation which succeeds with a short write.
// We use it to simulate gpgme_data_write returning a short write, which is legitimate.
type invalidShortWriter struct {
	maxWrite int
	written  []byte
}

func (w *invalidShortWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n > w.maxWrite {
		n = w.maxWrite
	}
	w.written = append(w.written, p[:n]...)
	return n, nil
}

func testReader(t testing.TB, r io.Reader, content []byte) {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, r)
	checkError(t, err)

	if int(n) != len(content) {
		t.Errorf("n = %d, want %d", n, len(content))
	}

	diff(t, buf.Bytes(), content)
}

type errWriter struct{ err error }

func (w errWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

type errReadSeeker struct{ err error }

func (rs errReadSeeker) Read([]byte) (int, error) {
	return 0, rs.err
}

func (rs errReadSeeker) Seek(int64, int) (int64, error) {
	return 0, rs.err
}
