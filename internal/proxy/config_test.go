package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL_UnmarshalJSON(t *testing.T) {
	t.Run("should unmarshal urls", func(t *testing.T) {
		// given
		data := []byte(`"http://some-host:3000/some-path"`)

		// when
		var url URL
		err := url.UnmarshalJSON(data)

		// then
		assert.NoError(t, err)
		assert.Equal(t, URL{
			Scheme: "http",
			Host:   "some-host:3000",
			Path:   "/some-path",
		}, url)
	})
}
