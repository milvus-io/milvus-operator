package external

import (
	"testing"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	madmin "github.com/minio/madmin-go"
	"github.com/stretchr/testify/assert"
)

func TestCheckMinIOFailed(t *testing.T) {
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
		Endpoint: "s3.amazonaws.com",
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
		UseSSL:   true,
	})
	assert.Error(t, err)
}

func TestIsHealthyByServerInfo(t *testing.T) {
	st := madmin.InfoMessage{
		Servers: []madmin.ServerProperties{
			{},
		},
	}
	err := isHealthyByServerInfo(st)
	assert.Error(t, err)
	st.Servers[0].State = "ok"
	err = isHealthyByServerInfo(st)
	assert.NoError(t, err)
}
