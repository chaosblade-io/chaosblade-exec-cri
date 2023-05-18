package container

import (
	"context"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
)

var (
	Timeout time.Duration
)

func GetRuntimeService(ctx context.Context, endpoint string, timeout time.Duration) (res internalapi.RuntimeService, err error) {
	t := Timeout
	if timeout != 0 {
		t = timeout
	}

	if endpoint == "" {
		for _, endPoint := range DefaultRuntimeEndpoints {
			log.Debugf(ctx, "Connect runtime service using endpoint %q with %q timeout", endPoint, t)

			res, err = remote.NewRemoteRuntimeService(endPoint, t, nil)
			if err != nil {
				log.Warnf(ctx, "Connect runtime service by `%s` failed, err : %v", endpoint, err)
				continue
			}

			log.Debugf(ctx, "Connected runtime service successfully using endpoint: %s", endPoint)
			break
		}
		return res, err
	}

	return remote.NewRemoteRuntimeService(endpoint, t, nil)
}

func GetImageService(ctx context.Context, endpoint string, timeout time.Duration) (res internalapi.ImageManagerService, err error) {
	t := Timeout
	if timeout != 0 {
		t = timeout
	}

	if endpoint == "" {
		for _, endPoint := range DefaultRuntimeEndpoints {
			log.Debugf(ctx, "Connect image service using endpoint %q with %q timeout", endPoint, t)

			res, err = remote.NewRemoteImageService(endPoint, t, nil)
			if err != nil {
				log.Warnf(ctx, "Connect image service by `%s` failed, err : %v", endpoint, err)
				continue
			}

			log.Debugf(ctx, "Connected image service successfully using endpoint: %s", endPoint)
			break
		}
		return res, err
	}

	return remote.NewRemoteImageService(endpoint, t, nil)
}
