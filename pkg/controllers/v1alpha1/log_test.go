package controllers

import (
	"errors"
	"testing"

	pulsarlog "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/golang/mock/gomock"
)

func TestPulsarLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	logger := NewMockLogger(ctrl)
	logger.EXPECT().WithName("pulsar").Return(logger)
	var pl pulsarlog.Logger = newPulsarLog(logger)

	logger.EXPECT().WithValues("f1", "v1").Return(logger)
	logger.EXPECT().WithValues("f2", "v2").Return(logger)
	pl.SubLogger(pulsarlog.Fields{"f1": "v1", "f2": "v2"})

	logger.EXPECT().WithValues("f3", "v3").Return(logger)

	pl.WithFields(pulsarlog.Fields{"f3": "v3"})

	logger.EXPECT().WithValues("f4", "v4").Return(logger)
	pl.WithField("f4", "v4")

	errMock := errors.New("error")
	logger.EXPECT().WithValues("err", errMock).Return(logger)
	pl.WithError(errMock)

	logger.EXPECT().V(2).Return(logger)
	logger.EXPECT().Info("foo,bar")
	pl.Debugf("foo,%s", "bar")
	logger.EXPECT().Info("foo,bar")
	pl.Infof("foo,%s", "bar")
	logger.EXPECT().Info("foo,bar")
	pl.Warnf("foo,%s", "bar")
	logger.EXPECT().Info("foo,bar")
	pl.Errorf("foo,%s", "bar")

	logger.EXPECT().V(2).Return(logger)
	logger.EXPECT().Info("foo,bar")
	pl.Debug("foo", ",bar")
	logger.EXPECT().Info("foo,bar")
	pl.Info("foo", ",bar")
	logger.EXPECT().Info("foo,bar")
	pl.Warn("foo", ",bar")
	logger.EXPECT().Info("foo,bar")
	pl.Error("foo", ",bar")
}
