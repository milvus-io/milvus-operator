package external

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// CheckMinIOArgs is info for acquiring storage condition
type CheckMinIOArgs struct {
	// S3 / MinIO
	Type          string
	AK            string
	SK            string
	Bucket        string
	Endpoint      string
	UseSSL        bool
	UseIAM        bool
	IAMEndpoint   string
	CloudProvider string
}

func CheckMinIO(args CheckMinIOArgs) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch args.Type {
	case v1beta1.StorageTypeS3:
		endpoint := args.Endpoint
		if args.UseSSL {
			// minio client cannot recognize aws endpoints with :443
			endpoint = strings.TrimSuffix(endpoint, ":443")
		}
		var cli *minio.Client
		var err error
		switch args.CloudProvider {
		case "gcp":
			cli, err = NewGCPMinioClient(endpoint, &minio.Options{
				Creds: credentials.NewStaticV2(args.AK, args.SK, ""),
			})
		default:
			var creds *credentials.Credentials
			if args.UseIAM {
				creds = credentials.NewIAM(args.IAMEndpoint)
			} else {
				creds = credentials.NewStaticV4(args.AK, args.SK, "")
			}
			cli, err = minio.New(endpoint, &minio.Options{
				Creds:  creds,
				Secure: args.UseSSL,
			})
		}

		if err != nil {
			return err
		}
		// see cli.HealthCheck()
		// there will be 43k requests per month (1 per minute)
		// will charge extra $0.02 per month by aws
		// according to https://aws.amazon.com/s3/pricing/
		_, err = cli.GetBucketLocation(ctx, args.Bucket)
		return err
	default:
		// default to minio
		mcli, err := madmin.New(args.Endpoint, args.AK, args.SK, args.UseSSL)
		if err != nil {
			return err
		}
		st, err := mcli.ServerInfo(ctx)
		if err != nil {
			return err
		}
		return isHealthyByServerInfo(st)
	}
}

func isHealthyByServerInfo(st madmin.InfoMessage) error {
	for _, server := range st.Servers {
		if server.State == "ok" || server.State == "online" {
			return nil
		}
	}
	return errors.New("no server ready in server info")
}

// WrapHTTPTransport wraps http.Transport, add an auth header to support GCP native auth
type WrapHTTPTransport struct {
	tokenSrc     oauth2.TokenSource
	backend      transport
	currentToken *oauth2.Token
}

// transport abstracts http.Transport to simplify test
type transport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
}

// NewWrapHTTPTransport constructs a new WrapHTTPTransport
func NewWrapHTTPTransport(secure bool) (*WrapHTTPTransport, error) {
	tokenSrc := google.ComputeTokenSource("")
	// in fact never return err
	backend, err := minio.DefaultTransport(secure)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create default transport")
	}
	return &WrapHTTPTransport{
		tokenSrc: tokenSrc,
		backend:  backend,
	}, nil
}

// RoundTrip wraps original http.RoundTripper by Adding a Bearer token acquired from tokenSrc
func (t *WrapHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// here Valid() means the token won't be expired in 10 sec
	// so the http client timeout shouldn't be longer, or we need to change the default `expiryDelta` time
	if !t.currentToken.Valid() {
		var err error
		t.currentToken, err = t.tokenSrc.Token()
		if err != nil {
			return nil, errors.Wrap(err, "failed to acquire token")
		}
	}
	req.Header.Set("Authorization", "Bearer "+t.currentToken.AccessToken)
	return t.backend.RoundTrip(req)
}

const GcsDefaultAddress = "storage.googleapis.com"

// NewGCPMinioClient returns a minio.Client which is compatible for GCS
func NewGCPMinioClient(address string, opts *minio.Options) (*minio.Client, error) {
	if opts == nil {
		opts = &minio.Options{}
	}
	if address == "" {
		address = GcsDefaultAddress
		opts.Secure = true
	}

	// adhoc to remove port of gcs address to let minio-go know it's gcs
	if strings.Contains(address, GcsDefaultAddress) {
		address = GcsDefaultAddress
	}

	if opts.Creds != nil {
		// if creds is set, use it directly
		return minio.New(address, opts)
	}
	// opts.Creds == nil, assume using IAM
	transport, err := NewWrapHTTPTransport(opts.Secure)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create default transport")
	}
	opts.Transport = transport
	opts.Creds = credentials.NewStaticV2("", "", "")
	return minio.New(address, opts)
}
