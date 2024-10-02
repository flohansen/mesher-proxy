package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/flohansen/sentinel/internal/proxy/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProxy_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockHttpClient(ctrl)

	t.Run("should use internal handlers when requesting /internal/reload", func(t *testing.T) {
		// given
		p := NewProxy()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/internal/reload", nil)

		// when
		p.ServeHTTP(w, r)

		// then
		res := w.Result()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("should return 404 NOT FOUND if there is no target", func(t *testing.T) {
		// given
		p := NewProxy()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		// when
		p.ServeHTTP(w, r)

		// then
		res := w.Result()
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("should return 500 INTERNAL SERVER ERROR if the proxy request fails", func(t *testing.T) {
		// given
		p := NewProxy(WithClient(client))
		p.services["/"] = Target{
			Path: "/",
			URL:  newUrl("http://localhost:3000/"),
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		client.EXPECT().
			Do(r).
			Return(nil, errors.New("some error")).
			Times(1)

		// when
		p.ServeHTTP(w, r)

		// then
		res := w.Result()
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	})

	t.Run("should inject script if response is text/html and update appropriate headers", func(t *testing.T) {
		// given
		p := NewProxy(WithClient(client))
		p.services["/"] = Target{
			Path: "/",
			URL:  newUrl("http://localhost:3000/"),
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		client.EXPECT().
			Do(r).
			Return(&http.Response{
				Body: io.NopCloser(bytes.NewReader([]byte("<body></body>"))),
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
			}, nil).
			Times(1)

		// when
		p.ServeHTTP(w, r)

		// then
		res := w.Result()
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Regexp(t, "<body><script>.*</script></body>", string(body))
		assert.Equal(t, fmt.Sprintf("%d", len(body)), res.Header.Get("Content-Length"))
	})

	t.Run("should return 200 OK and proxy the request to the registered service", func(t *testing.T) {
		// given
		p := NewProxy(WithClient(client))
		p.services["/"] = Target{
			Path: "/",
			URL:  newUrl("http://localhost:3000/"),
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		client.EXPECT().
			Do(r).
			Return(&http.Response{
				Body: io.NopCloser(bytes.NewReader([]byte("response from service"))),
			}, nil).
			Times(1)

		// when
		p.ServeHTTP(w, r)

		// then
		res := w.Result()
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "response from service", string(body))
		assert.Equal(t, "localhost:3000", r.Host)
		assert.Equal(t, "localhost:3000", r.URL.Host)
		assert.Equal(t, "http", r.URL.Scheme)
		assert.Equal(t, "/", r.URL.Path)
	})
}

func newUrl(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}
