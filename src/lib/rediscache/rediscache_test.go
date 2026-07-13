package rediscache_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RedisCacheSuite struct {
	suite.Suite
}

func TestRedisCache(t *testing.T) {
	suite.Run(t, &RedisCacheSuite{})
}
