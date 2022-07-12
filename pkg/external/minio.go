package external

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// CheckMinIOArgs is info for acquiring storage condition
type CheckMinIOArgs struct {
	// S3 / MinIO
	Type        string
	AK          string
	SK          string
	Bucket      string
	Endpoint    string
	UseSSL      bool
	UseIAM      bool
	IAMEndpoint string
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
		var creds *credentials.Credentials
		if args.UseIAM {
			creds = credentials.NewIAM(args.IAMEndpoint)
		} else {
			creds = credentials.NewStaticV4(args.AK, args.SK, "")
		}
		cli, err := minio.New(endpoint, &minio.Options{
			Creds:  creds,
			Secure: args.UseSSL,
		})
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
	ready := false
	for _, server := range st.Servers {
		log.Print(server)
		if server.State == "ok" || server.State == "online" {
			ready = true
			break
		}
	}
	if !ready {
		return errors.New("no server ready in server info")
	}
	return nil
}
