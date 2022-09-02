/*
Copyright 2022 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

const (
	// MigrationModeTest is used to sign that the migration process should be started in "test" mode
	MigrationModeTest = "test"
	// MigrationModeTestWithPrivate is used to sign that the migration process should be started in "test-with-private" mode
	MigrationModeTestWithPrivate = "test-with-private"
	// MigrationModeProduction is used to sign that the migration process should be started in "production" mode
	MigrationModeProduction = "production"
)

// MigratedResource represents a single resource that has been migrated
type MigratedResource struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	MigratedAs []string `json:"migratedAs"`
	Warnings   []string `json:"warnings"`
}
