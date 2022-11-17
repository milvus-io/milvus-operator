package external

import (
	"net/http"
	"testing"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestCheckMinIO(t *testing.T) {
	// badendpoint
	err := CheckMinIO(CheckMinIOArgs{
		Type:     v1beta1.StorageTypeS3,
		AK:       "dummy",
		SK:       "dummy",
		Endpoint: "badendpoint.s3.amazonaws.com",
		Bucket:   "dummy",
		UseSSL:   true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Amazon S3 endpoint should be")

	err = CheckMinIO(CheckMinIOArgs{
		Type:     v1beta1.StorageTypeS3,
		AK:       "dummy",
		SK:       "dummy",
		Endpoint: "s3.amazonaws.com:443",
		Bucket:   "dummy",
		UseSSL:   true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AWS Access Key Id")

	err = CheckMinIO(CheckMinIOArgs{
		Type:     v1beta1.StorageTypeMinIO,
		AK:       "dummy",
		SK:       "dummy",
		Endpoint: "minio-endpoint.dummy",
		Bucket:   "dummy",
		UseSSL:   false,
	})
	assert.Error(t, err)

	err = CheckMinIO(CheckMinIOArgs{
		Type:     v1beta1.StorageTypeS3,
		AK:       "",
		SK:       "",
		Endpoint: "s3.amazonaws.com:443",
		Bucket:   "zilliz-infra-test",
		UseSSL:   true,
	})
	assert.NoError(t, err)

	err = CheckMinIO(CheckMinIOArgs{
		Type:     v1beta1.StorageTypeS3,
		AK:       "",
		SK:       "",
		Endpoint: "s3.ap-southeast-1.amazonaws.com:443",
		Bucket:   "zilliz-infra-test",
		UseSSL:   true,
		UseIAM:   true,
	})
	assert.NoError(t, err)

	err = CheckMinIO(CheckMinIOArgs{
		Type:          v1beta1.StorageTypeS3,
		AK:            "",
		SK:            "",
		Endpoint:      "storage.googleapis.com:443",
		Bucket:        "zilliz-infra-poc",
		UseSSL:        true,
		UseIAM:        true,
		CloudProvider: "gcp",
	})
	assert.NoError(t, err)
}

func TestIsHealthyByServerInfo(t *testing.T) {
	st := madmin.InfoMessage{
		Servers: []madmin.ServerProperties{
			{},
			{},
		},
	}
	err := isHealthyByServerInfo(st)
	assert.Error(t, err)
	st.Servers[0].State = "online"
	err = isHealthyByServerInfo(st)
	assert.NoError(t, err)
	st.Servers[0].State = "ok"
	err = isHealthyByServerInfo(st)
	assert.NoError(t, err)
	st.Servers[0].State = "offline"
	st.Servers[1].State = "online"
	err = isHealthyByServerInfo(st)
	assert.NoError(t, err)
}

func TestNewGCPMinioClient(t *testing.T) {
	t.Run("iam ok", func(t *testing.T) {
		minioCli, err := NewGCPMinioClient("", nil)
		assert.NoError(t, err)
		assert.Equal(t, GcsDefaultAddress, minioCli.EndpointURL().Host)
		assert.Equal(t, "https", minioCli.EndpointURL().Scheme)
	})

	t.Run("ak sk ok", func(t *testing.T) {
		minioCli, err := NewGCPMinioClient(GcsDefaultAddress+":443", &minio.Options{
			Creds:  credentials.NewStaticV2("ak", "sk", ""),
			Secure: true,
		})
		assert.NoError(t, err)
		assert.Equal(t, GcsDefaultAddress, minioCli.EndpointURL().Host)
		assert.Equal(t, "https", minioCli.EndpointURL().Scheme)
	})

	t.Run("create failed", func(t *testing.T) {
		defaultTransBak := minio.DefaultTransport
		defer func() {
			minio.DefaultTransport = defaultTransBak
		}()
		minio.DefaultTransport = func(secure bool) (*http.Transport, error) {
			return nil, errors.New("mock error")
		}
		_, err := NewGCPMinioClient("", nil)
		assert.Error(t, err)
	})
}

type mockTransport struct {
	err error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, m.err
}

type mockTokenSource struct {
	token string
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: m.token}, m.err
}

func TestGCPWrappedHTTPTransport_RoundTrip(t *testing.T) {
	ts, err := NewWrapHTTPTransport(true)
	assert.NoError(t, err)
	ts.backend = &mockTransport{}
	ts.tokenSrc = &mockTokenSource{token: "mocktoken"}

	t.Run("valid token ok", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://example.com", nil)
		assert.NoError(t, err)
		_, err = ts.RoundTrip(req)
		assert.NoError(t, err)
		assert.Equal(t, "Bearer mocktoken", req.Header.Get("Authorization"))
	})

	t.Run("invalid token, refresh failed", func(t *testing.T) {
		ts.currentToken = nil
		ts.tokenSrc = &mockTokenSource{err: errors.New("mock error")}
		req, err := http.NewRequest("GET", "http://example.com", nil)
		assert.NoError(t, err)
		_, err = ts.RoundTrip(req)
		assert.Error(t, err)
	})

	t.Run("invalid token, refresh ok", func(t *testing.T) {
		ts.currentToken = nil
		ts.tokenSrc = &mockTokenSource{err: nil}
		req, err := http.NewRequest("GET", "http://example.com", nil)
		assert.NoError(t, err)
		_, err = ts.RoundTrip(req)
		assert.NoError(t, err)
	})

	ts.currentToken = &oauth2.Token{}
	t.Run("valid token, call failed", func(t *testing.T) {
		ts.backend = &mockTransport{err: errors.New("mock error")}
		req, err := http.NewRequest("GET", "http://example.com", nil)
		assert.NoError(t, err)
		_, err = ts.RoundTrip(req)
		assert.Error(t, err)
	})

}
