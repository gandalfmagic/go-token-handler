package zlogger

import "net/http"

type StatusRecorder struct {
	http.ResponseWriter
	Status  int
	Written int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Write(b []byte) (int, error) {
	var err error
	r.Written, err = r.ResponseWriter.Write(b)

	return r.Written, err
}
