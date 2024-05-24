package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func Test_Mount(t *testing.T) {
	suite.Run(t, new(MountTest))
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type MountTest struct {
	suite.Suite
}

func (t *MountTest) TestIsMetadataPrefetchOnMountEnabled() {
	for _, input := range []struct {
		mode      string
		isEnabled bool
	}{
		{
			mode: "sync", isEnabled: true,
		},
		{
			mode: "async", isEnabled: true,
		},
		{
			mode: "disabled",
		},
		{
			mode: "",
		},
	} {
		t.T().Run("input="+input.mode, func(t2 *testing.T) {
			assert.Equal(t.T(), input.isEnabled, isMetadataPrefetchOnMountEnabled(input.mode))
		})
	}
}

func (t *MountTest) TestIsTypeCacheEnabled() {
	for _, input := range []struct {
		metadataCacheTTL   time.Duration
		typeCacheMaxSizeMB int
		isEnabled          bool
	}{
		{
			metadataCacheTTL:   0,
			typeCacheMaxSizeMB: 32,
		},
		{
			metadataCacheTTL:   time.Second,
			typeCacheMaxSizeMB: 0,
		},
		{
			metadataCacheTTL:   time.Second,
			typeCacheMaxSizeMB: -2,
		},
		{
			metadataCacheTTL:   time.Second,
			typeCacheMaxSizeMB: 32,
			isEnabled:          true,
		},
	} {
		t.T().Run(fmt.Sprintf("metadataCacheTTL=%v,typeCacheMaxSizeMB=%v", input.metadataCacheTTL, input.typeCacheMaxSizeMB), func(t2 *testing.T) {
			assert.Equal(t.T(), input.isEnabled, isTypeCacheEnabled(input.metadataCacheTTL, input.typeCacheMaxSizeMB))
		})
	}
}

func (t *MountTest) TestIsStatCacheEnabled() {
	for _, input := range []struct {
		metadataCacheTTL   time.Duration
		statCacheMaxSizeMB uint64
		isEnabled          bool
	}{
		{
			metadataCacheTTL:   0,
			statCacheMaxSizeMB: 32,
		},
		{
			metadataCacheTTL:   time.Second,
			statCacheMaxSizeMB: 0,
		},
		{
			metadataCacheTTL:   time.Second,
			statCacheMaxSizeMB: 32,
			isEnabled:          true,
		},
	} {
		t.T().Run(fmt.Sprintf("metadataCacheTTL=%v,typeCacheMaxSizeMB=%v", input.metadataCacheTTL, input.statCacheMaxSizeMB), func(t2 *testing.T) {
			assert.Equal(t.T(), input.isEnabled, isStatCacheEnabled(input.metadataCacheTTL, input.statCacheMaxSizeMB))
		})
	}
}
