package controllers

import (
	"sort"
	"strings"
	"time"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/patrickmn/go-cache"
)

// endpointCheckCache singleton
var endpointCheckCache EndpointCheckCache = NewEndpointCheckCacheImpl()

// EndpointCheckCache coordinates endpoint check to avoid duplicated check for same endpoint
type EndpointCheckCache interface {
	Get(endpoint []string) (condition *v1beta1.MilvusCondition, found bool)
	Set(endpoints []string, condition *v1beta1.MilvusCondition)
}

// EndpointCheckCacheImpl implements EndpointCheckCache
type EndpointCheckCacheImpl struct {
	cache *cache.Cache
}

func NewEndpointCheckCacheImpl() EndpointCheckCache {
	return EndpointCheckCacheImpl{cache: cache.New(unhealthySyncInterval, time.Hour)}
}

func strSliceAsKey(input []string) string {
	// sort slice & combine with comma
	sortable := sort.StringSlice(input)
	sort.Sort(sortable)
	return strings.Join(sortable, ",")
}

func (e EndpointCheckCacheImpl) Get(endpoints []string) (condition *v1beta1.MilvusCondition, isUpToDate bool) {
	if len(endpoints) == 0 {
		return nil, false
	}
	item, found := e.cache.Get(strSliceAsKey(endpoints))
	if !found {
		return nil, false
	}
	return item.(*v1beta1.MilvusCondition), true
}

func (e EndpointCheckCacheImpl) Set(endpoints []string, condition *v1beta1.MilvusCondition) {
	if len(endpoints) == 0 {
		return
	}
	e.cache.Set(strSliceAsKey(endpoints), condition, 0)
}
