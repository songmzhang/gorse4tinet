// Copyright 2020 gorse Project Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var redisDSN string

func init() {
	// get environment variables
	env := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}
	redisDSN = env("REDIS_URI", "redis://127.0.0.1:6379/")
}

type mockRedis struct {
	Database
}

func newMockRedis(t *testing.T) *mockRedis {
	var err error
	db := new(mockRedis)
	assert.NoError(t, err)
	db.Database, err = Open(redisDSN)
	assert.NoError(t, err)
	return db
}

func (db *mockRedis) Close(t *testing.T) {
	err := db.Database.Close()
	assert.NoError(t, err)
}

func TestRedis_Meta(t *testing.T) {
	db := newMockRedis(t)
	defer db.Close(t)
	testMeta(t, db.Database)
}

func TestRedis_Scores(t *testing.T) {
	db := newMockRedis(t)
	defer db.Close(t)
	testScores(t, db.Database)
}

func TestRedis_Sort(t *testing.T) {
	db := newMockRedis(t)
	defer db.Close(t)
	testSort(t, db.Database)
}

func TestRedis_Set(t *testing.T) {
	db := newMockRedis(t)
	defer db.Close(t)
	testSet(t, db.Database)
}
