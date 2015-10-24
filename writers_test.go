package updater

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBuffer(t *testing.T) {
	// Pre-defined file
	{
		// Get filename
		f, err := ioutil.TempFile("", "testing-")
		require.Nil(t, err)
		path := f.Name()
		f.Close()

		// Write file
		b := &FileBuffer{
			Path: path,
		}
		defer func() {
			err := b.Close()
			assert.Nil(t, err, "Close error: %v", err)
		}()

		_, err = b.Write([]byte("hello world"))
		assert.Nil(t, err, "Write error: %v", err)

		// Check contents
		f, err = os.Open(path)
		require.Nil(t, err, "Could not open file: %v", err)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		assert.Nil(t, err, "Could not read file: %v", err)
		assert.Equal(t, "hello world", string(data))

		// Clean up
		err = os.Remove(path)
		assert.Nil(t, err, "Could not clean up: %v", err)
	}

	// Temporary file
	{
		// Write file
		b := &FileBuffer{}
		defer func() {
			err := b.Close()
			assert.Nil(t, err, "Close error: %v", err)
		}()

		_, err := b.Write([]byte("hello world"))
		assert.Nil(t, err, "Write error: %v", err)
		require.NotEqual(t, "", b.Path)

		// Check contents
		f, err := os.Open(b.Path)
		require.Nil(t, err, "Could not open file: %v", err)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		assert.Equal(t, "hello world", string(data))

		// Clean up
		err = os.Remove(b.Path)
		assert.Nil(t, err, "Could not clean up: %v", err)
	}

	// Error file
	{
		b := &FileBuffer{
			Path: "/n/o/n/e/x/i/s/t/i/n/g/file",
		}
		defer func() {
			err := b.Close()
			assert.Nil(t, err, "Close error: %v", err)
		}()

		_, err := b.Write([]byte("hello world"))
		assert.Error(t, err)
	}

	// Close unopend file
	{
		b := &FileBuffer{}
		err := b.Close()
		assert.Nil(t, err)
	}
}

func TestDelayedFile(t *testing.T) {
	// Valid test
	{
		// Get filename
		f, err := ioutil.TempFile("", "testing-")
		require.Nil(t, err)
		path := f.Name()
		f.Close()

		// Write data
		df := NewDelayedFile(path)
		_, err = df.Write([]byte("hello world"))
		err = df.Close()
		assert.Nil(t, err, "Could not close file: %v", err)

		// Check contents
		f, err = os.Open(f.Name())
		require.Nil(t, err, "Could not open file: %v", err)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		assert.Nil(t, err, "Could not read file: %v", err)
		assert.Equal(t, "hello world", string(data))

		// Make sure temp file does not exists
		_, err = os.Stat(df.buffer.Path)
		assert.True(t, os.IsNotExist(err), "Temporary file was not removed.")

		// Clean up
		err = os.Remove(path)
		assert.Nil(t, err, "Could not clean up: %v", err)
	}

	// Invalid destination file
	{
		// Write
		df := NewDelayedFile("/n/o/n/e/x/i/s/t/i/n/g/file")
		_, err := df.Write([]byte("hello world"))
		assert.Nil(t, err, "Could not write to file: %v", err)

		// Close
		err = df.Close()
		assert.True(t, os.IsNotExist(err))

		// Make sure temp file does not exists
		_, err = os.Stat(df.buffer.Path)
		assert.True(t, os.IsNotExist(err), "Temporary file was not removed.")
	}

	// Invalid source file
	{
		// Get filename
		f, err := ioutil.TempFile("", "testing-")
		require.Nil(t, err)
		path := f.Name()
		f.Close()

		// Write
		df := NewDelayedFile(path)
		_, err = df.Write([]byte("hello world"))
		assert.Nil(t, err, "Could not write to file: %v", err)

		// Close
		err = os.Remove(df.buffer.Path)
		assert.Nil(t, err, "Could not remove temporary file: %v", err)
		err = df.Close()
		assert.True(t, os.IsNotExist(err))

		// Make sure temp file does not exists
		_, err = os.Stat(df.buffer.Path)
		assert.True(t, os.IsNotExist(err), "Temporary file was not removed.")
	}

	// Aborted
	{
		// Get filename
		f, err := ioutil.TempFile("", "testing-")
		require.Nil(t, err)
		path := f.Name()
		f.Close()
		os.Remove(path)

		// Write
		df := NewDelayedFile(path)
		_, err = df.Write([]byte("hello world"))
		assert.Nil(t, err, "Could not write to file: %v", err)

		// Abort
		df.Abort()

		// Close
		err = df.Close()
		assert.Nil(t, err, "Could not close file: %v", err)

		// Make sure temporary file is removed
		_, err = os.Stat(df.buffer.Path)
		assert.True(t, os.IsNotExist(err))

		// Make sure the contents were not copied
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	}

	// Faulty copier
	{
		// Get filename
		f, err := ioutil.TempFile("", "testing-")
		require.Nil(t, err)
		path := f.Name()
		f.Close()
		os.Remove(path)

		// Write
		df := NewDelayedFile(path)
		_, err = df.Write([]byte("hello world"))
		assert.Nil(t, err, "Could not write to file: %v", err)

		// Setup failure
		testErr := errors.New("Copy test error")
		df.copier = func(io.Writer, io.Reader) (int64, error) {
			return 0, testErr
		}

		// Close
		err = df.Close()
		assert.Equal(t, testErr, err)
	}
}
