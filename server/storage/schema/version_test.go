// Copyright 2021 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"testing"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	bbolt "github.com/dgraph-io/badger/v4"

	"go.etcd.io/etcd/server/v3/storage/backend"
	betesting "go.etcd.io/etcd/server/v3/storage/backend/testing"
)

// TestVersion ensures that UnsafeSetStorageVersion/UnsafeReadStorageVersion work well together.
func TestVersion(t *testing.T) {
	tcs := []struct {
		version       string
		expectVersion string
	}{
		{
			version:       "3.5.0",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-alpha",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-beta.0",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-rc.1",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.1",
			expectVersion: "3.5.0",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.version, func(t *testing.T) {
			lg := zaptest.NewLogger(t)
			be, tmpPath := betesting.NewTmpBackend(t, time.Microsecond, 10)
			tx := be.BatchTx()
			if tx == nil {
				t.Fatal("batch tx is nil")
			}
			tx.Lock()
			tx.UnsafeCreateBucket(Meta)
			UnsafeSetStorageVersion(tx, semver.New(tc.version))
			tx.Unlock()
			be.ForceCommit()
			be.Close()

			b := backend.NewDefaultBackend(lg, tmpPath)
			defer b.Close()
			v := UnsafeReadStorageVersion(b.BatchTx())

			assert.Equal(t, tc.expectVersion, v.String())
		})
	}
}

// TestVersionSnapshot ensures that UnsafeSetStorageVersion/unsafeReadStorageVersionFromSnapshot work well together.
func TestVersionSnapshot(t *testing.T) {
	tcs := []struct {
		version       string
		expectVersion string
	}{
		{
			version:       "3.5.0",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-alpha",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-beta.0",
			expectVersion: "3.5.0",
		},
		{
			version:       "3.5.0-rc.1",
			expectVersion: "3.5.0",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.version, func(t *testing.T) {
			be, tmpPath := betesting.NewTmpBackend(t, time.Microsecond, 10)
			tx := be.BatchTx()
			if tx == nil {
				t.Fatal("batch tx is nil")
			}
			tx.Lock()
			tx.UnsafeCreateBucket(Meta)
			UnsafeSetStorageVersion(tx, semver.New(tc.version))
			tx.Unlock()
			be.ForceCommit()
			be.Close()
			db, err := bbolt.Open(bbolt.DefaultOptions(tmpPath))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			var ver *semver.Version
			if err = db.View(func(tx *bbolt.Txn) error {
				ver = ReadStorageVersionFromSnapshot(tx)
				return nil
			}); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.expectVersion, ver.String())

		})
	}
}
