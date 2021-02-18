package providers

import (
	"bufio"
	"encoding/hex"
	"hash"
	"io"
	"io/ioutil"
)

func flushResponse(resp io.ReadCloser) {
	io.Copy(ioutil.Discard, resp) // nolint: errcheck
	resp.Close()
}

func copyResponse(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, bufio.NewReader(src))

	return err
}

func hashedCopyResponse(hashFunc func() hash.Hash, dst io.Writer, src io.Reader) (string, error) {
	hasher := hashFunc()
	err := copyResponse(io.MultiWriter(hasher, dst), src)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
