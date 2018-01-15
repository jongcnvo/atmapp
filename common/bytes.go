package common

//CopyBytes returns a duplication of self byte array
//Input - copiedBytes: pointer to destination
//Output - copiedBytes: copy byte array to destination
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}
