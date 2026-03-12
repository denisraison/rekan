package handlers

import (
	"io"
	"mime/multipart"
	"net/http"
)

// formFileData reads the named form file field, returning its bytes, detected
// content type, and original header. The caller must not close the file.
func formFileData(r *http.Request, field string) ([]byte, string, *multipart.FileHeader, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return nil, "", nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", nil, err
	}

	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return data, ct, header, nil
}
