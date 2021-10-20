package rulesmanager

import (
	"fmt"
	"github.com/seveirbian/edgeserverless/pkg/apis/edgeserverless/v1alpha1"
	"sync"
)

type RulesManager struct {
	// uri -> targets
	Rules sync.Map
}

func NewRulesManager() *RulesManager {
	return &RulesManager{
		Rules: sync.Map{},
	}
}

func (r *RulesManager) GetRule(uri string) (*v1alpha1.RouteSpec, error) {
	values, ok := r.Rules.Load(uri)
	if !ok {
		return nil, fmt.Errorf("[RulesManager] no value for uri %s\n", uri)
	}

	targets, ok := values.(v1alpha1.RouteSpec)
	if !ok {
		return nil, fmt.Errorf("[RulesManager] value is not valid targets\n")
	}

	return &targets, nil
}

func (r *RulesManager) AddRule(uri string, targets v1alpha1.RouteSpec) {
	r.Rules.Store(uri, targets)
}

func (r *RulesManager) DeleteRule(uri string) {
	r.Rules.Delete(uri)
}
