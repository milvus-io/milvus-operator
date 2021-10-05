package helm

import (
	"errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type ChartRequest struct {
	ReleaseName string
	Namespace   string
	Chart       string
	Values      map[string]interface{}
}

func NeedUpdate(status release.Status) bool {
	return status == release.StatusFailed ||
		status == release.StatusUnknown ||
		status == release.StatusUninstalled
}

func GetStatus(cfg *action.Configuration, releaseName string) (release.Status, error) {
	client := action.NewStatus(cfg)
	rel, err := client.Run(releaseName)
	if err != nil {
		return release.StatusUnknown, err
	}

	return rel.Info.Status, nil
}

func GetValues(cfg *action.Configuration, releaseName string) (map[string]interface{}, error) {
	client := action.NewGetValues(cfg)
	vals, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	if vals == nil {
		return map[string]interface{}{}, nil
	}

	return vals, nil
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

	return Update(cfg, request)
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

func Uninstall(cfg *action.Configuration, releaseName string) error {
	_, err := cfg.Releases.History(releaseName)
	if errors.Is(err, driver.ErrReleaseNotFound) {
		return nil
	}

	client := action.NewUninstall(cfg)
	client.DisableHooks = true
	_, err = client.Run(releaseName)
	if err != nil {
		return err
	}

	return nil
}
