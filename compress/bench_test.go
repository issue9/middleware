// SPDX-License-Identifier: MIT

package compress

import (
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/issue9/assert/v2"
	"github.com/issue9/assert/v2/rest"
)

func f1(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("text\nhtml"))
}

func BenchmarkCompress_ServeHTTP_any(b *testing.B) {
	a := assert.New(b, false)
	c := newCompress(a, "*")
	a.NotNil(c)

	srv := rest.NewServer(a, c.MiddlewareFunc(f1), nil)

	for i := 0; i < b.N; i++ {
		srv.NewRequest(http.MethodGet, "/").
			Header("Accept-encoding", "gzip;q=0.8,deflate").
			Do(nil).
			Header("Content-Encoding", "deflate")
	}
}

func BenchmarkCompress_ServeHTTP(b *testing.B) {
	a := assert.New(b, false)
	c := newCompress(a, "text/*")
	a.NotNil(c)

	srv := rest.NewServer(a, c.MiddlewareFunc(f1), nil)

	for i := 0; i < b.N; i++ {
		srv.NewRequest(http.MethodGet, "/").
			Header("Accept-encoding", "gzip;q=0.8,deflate").
			Do(nil).
			Header("Content-Encoding", "deflate")
	}
}

func BenchmarkCompress_canCompress_any(b *testing.B) {
	a := assert.New(b, false)

	c := New(log.New(os.Stderr, "", log.LstdFlags), nil, "*")
	a.NotNil(c)

	for i := 0; i < b.N; i++ {
		c.canCompressed("text/html;charset=utf-8")
	}
}

func BenchmarkCompress_canCompress(b *testing.B) {
	a := assert.New(b, false)

	c := New(log.New(os.Stderr, "", log.LstdFlags), nil, "text/*", "application/json")
	a.NotNil(c)

	for i := 0; i < b.N; i++ {
		c.canCompressed("text/html;charset=utf-8")
	}
}
