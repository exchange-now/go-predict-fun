package x

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildProxyFuncWithAuth(t *testing.T) {
	fn, err := buildProxyFunc("127.0.0.1:8080", "user", "pass")
	require.NoError(t, err)

	u, err := fn(&http.Request{URL: &url.URL{Scheme: "https", Host: "api.predict.fun"}})
	require.NoError(t, err)
	require.Equal(t, "http", u.Scheme)
	require.Equal(t, "127.0.0.1:8080", u.Host)
	require.Equal(t, "user", u.User.Username())
	pw, _ := u.User.Password()
	require.Equal(t, "pass", pw)
}

func TestBuildProxyFuncEmptyUsesEnvironment(t *testing.T) {
	fn, err := buildProxyFunc("", "", "")
	require.NoError(t, err)
	require.NotNil(t, fn)
	u, err := fn(&http.Request{URL: &url.URL{Scheme: "https", Host: "api.predict.fun"}})
	require.NoError(t, err)
	require.Nil(t, u)
}
