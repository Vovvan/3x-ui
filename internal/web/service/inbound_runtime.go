package service

import (
	"errors"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
)

func localRuntime() (runtime.Runtime, error) {
	mgr := runtime.GetManager()
	if mgr == nil {
		return nil, errors.New("runtime manager is not initialized")
	}
	return mgr.Local(), nil
}

func (s *InboundService) runtimeFor(*model.Inbound) (runtime.Runtime, error) {
	return localRuntime()
}
