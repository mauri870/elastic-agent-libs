// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nameSub1 = "sub.registry1"
	nameSub2 = "sub.registry2"
)

func TestRegistryEmpty(t *testing.T) {
	defer func(t *testing.T) {
		err := Clear()
		require.NoError(t, err)
	}(t)

	// get value
	v := Get("missing")
	if v != nil {
		t.Errorf("got %v, wanted nil", v)
	}

	// get value with recursive lookup
	v = Get("missing.value")
	if v != nil {
		t.Errorf("got %v, wanted nil", v)
	}

	// get missing registry
	reg := GetRegistry("missing")
	if reg != nil {
		t.Errorf("got %v, wanted nil", reg)
	}

	// get registry with recursive lookup
	reg = GetRegistry("missing.registry")
	if reg != nil {
		t.Errorf("got %v, wanted nil", reg)
	}
}

func TestRegistryGet(t *testing.T) {
	defer func(t *testing.T) {
		err := Clear()
		require.NoError(t, err)
	}(t)

	name1 := "v"
	name2 := nameSub1 + "." + name1
	name3 := nameSub2 + "." + name1

	// register top-level and recursive metric
	v1 := NewInt(Default, name1, Report)
	sub1 := Default.NewRegistry(nameSub1)
	sub2 := Default.NewRegistry(nameSub2)
	v2 := NewString(nil, name2, Report)
	v3 := NewFloat(sub2, name1, Report)

	// get values
	v := Get(name1)
	assert.Equal(t, v, v1)

	// get nested metric from top-level
	v = Get(name2)
	assert.Equal(t, v, v2)
	v = Get(name3)
	assert.Equal(t, v, v3)

	// get sub registry
	reg1 := GetRegistry(nameSub1)
	assert.Equal(t, sub1, reg1)
	reg2 := GetRegistry(nameSub2)
	assert.Equal(t, sub2, reg2)

	// get value from sub-registry
	v = reg1.Get(name1)
	assert.Equal(t, v, v2)

	v = reg2.Get(name1)
	assert.Equal(t, v, v3)
}

func TestRegistryRemove(t *testing.T) {
	defer func(t *testing.T) {
		err := Clear()
		require.NoError(t, err)
	}(t)

	name1 := "v"
	name2 := nameSub1 + "." + name1
	name3 := nameSub2 + "." + name1

	// register top-level and recursive metric
	NewInt(Default, name1, Report)
	sub1 := Default.NewRegistry(nameSub1)
	sub2 := Default.NewRegistry(nameSub2)
	NewInt(Default, name2, Report)
	NewInt(sub2, name1, Report)

	// remove metrics:
	Remove(name1)
	sub1.Remove(name1) // == Remove(name2)
	Remove(name3)      // remove name 3 recursively

	// check no variable is reachable
	assert.Nil(t, Get(name1))
	assert.Nil(t, Get(name2))
	assert.Nil(t, Get(name3))
}

func TestRegistryIter(t *testing.T) {
	defer func(t *testing.T) {
		err := Clear()
		require.NoError(t, err)
	}(t)

	vars := map[string]int64{
		"sub.registry.v1": 1,
		"sub.registry.v2": 2,
		"v3":              3,
	}

	for name, v := range vars {
		i := NewInt(Default, name, Report)
		i.Add(v)
	}

	collected := map[string]int64{}
	Do(Full, func(name string, v interface{}) {
		var ok bool
		collected[name], ok = v.(int64)
		require.True(t, ok)
	})

	assert.Equal(t, vars, collected)
}

func TestGetOrCreateRegistry(t *testing.T) {
	root := &Registry{
		name:    "root",
		entries: map[string]entry{},
		opts:    &defaultOptions,
	}

	require.Nil(t, root.GetRegistry("a.b.c"), "GetRegistry on empty registry always returns nil")

	c := root.GetOrCreateRegistry("a.b.c")
	require.NotNil(t, c, "GetOrCreateRegistry must be successful on an empty registry")
	assert.Equal(t, c, root.GetRegistry("a.b.c"), "GetRegistry after GetOrCreateRegistry should return the same value")
	assert.Equal(t, "root.a.b.c", c.name, "Registries created with GetOrCreateRegistry should contain the parent name followed by the path to the registry")

	y := c.GetOrCreateRegistry("z.y")
	require.NotNil(t, y, "GetOrCreateRegistry must be successful on an empty registry")
	assert.Equal(t, y, c.GetOrCreateRegistry("z.y"), "GetOrCreateRegistry on the same input should return the same result")
	assert.Equal(t, c.GetOrCreateRegistry("z.y"), root.GetOrCreateRegistry("a.b.c.z.y"), "GetOrCreateRegistry with equivalent paths from different starting points should return the same result")

	c.Add("scalar", &Int{}, Full)
	assert.Nil(t, root.GetOrCreateRegistry("a.b.c.scalar.w.x"), "GetOrCreateRegistry should return nil if part of the path is a non-registry type")
}
