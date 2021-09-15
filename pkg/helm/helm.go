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

func Upgrade(cfg *action.Configuration, request ChartRequest) error {
	histClient := action.NewHistory(cfg)
	histClient.Max = 1
	if _, err := histClient.Run(request.ReleaseName); err == driver.ErrReleaseNotFound {
		return Install(cfg, request)
	}

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
