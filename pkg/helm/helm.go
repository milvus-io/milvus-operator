package helm

import (
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type ChartRequest struct {
	ReleaseName string
	Namespace   string
	Chart       string
	Values      map[string]interface{}
}

func ReleaseExist(cfg *action.Configuration, releaseName string) (bool, error) {
	histClient := action.NewHistory(cfg)
	histClient.Max = 1
	_, err := histClient.Run(releaseName)
	if err == driver.ErrReleaseNotFound {
		return false, nil
	}

	return err == nil, err
}

func Upgrade(cfg *action.Configuration, request ChartRequest) error {
	exist, err := ReleaseExist(cfg, request.ReleaseName)
	if err != nil {
		return err
	}
	if !exist {
		return Install(cfg, request)
	}

	return nil

	//return Update(cfg, request)
}

func Update(cfg *action.Configuration, request ChartRequest) error {
	client := action.NewUpgrade(cfg)
	client.Namespace = request.Namespace
	chartRequested, err := loader.Load(request.Chart)
	if err != nil {
		return err
	}

	_, err = client.Run(request.ReleaseName, chartRequested, request.Values)
	return err
}

func Install(cfg *action.Configuration, request ChartRequest) error {
	client := action.NewInstall(cfg)
	client.ReleaseName = request.ReleaseName
	client.Namespace = request.Namespace
	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	chartRequested, err := loader.Load(request.Chart)
	if err != nil {
		return err
	}

	_, err = client.Run(chartRequested, request.Values)
	return err
}
