/*
Copyright The Kubernetes Authors.

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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// DownwardAPIVolumeFileApplyConfiguration represents an declarative configuration of the DownwardAPIVolumeFile type for use
// with apply.
type DownwardAPIVolumeFileApplyConfiguration struct {
	Path             *string                                  `json:"path,omitempty"`
	FieldRef         *ObjectFieldSelectorApplyConfiguration   `json:"fieldRef,omitempty"`
	ResourceFieldRef *ResourceFieldSelectorApplyConfiguration `json:"resourceFieldRef,omitempty"`
	Mode             *int32                                   `json:"mode,omitempty"`
}

// DownwardAPIVolumeFileApplyConfiguration constructs an declarative configuration of the DownwardAPIVolumeFile type for use with
// apply.
func DownwardAPIVolumeFile() *DownwardAPIVolumeFileApplyConfiguration {
	return &DownwardAPIVolumeFileApplyConfiguration{}
}

// WithPath sets the Path field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Path field is set to the value of the last call.
func (b *DownwardAPIVolumeFileApplyConfiguration) WithPath(value string) *DownwardAPIVolumeFileApplyConfiguration {
	b.Path = &value
	return b
}

// WithFieldRef sets the FieldRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FieldRef field is set to the value of the last call.
func (b *DownwardAPIVolumeFileApplyConfiguration) WithFieldRef(value *ObjectFieldSelectorApplyConfiguration) *DownwardAPIVolumeFileApplyConfiguration {
	b.FieldRef = value
	return b
}

// WithResourceFieldRef sets the ResourceFieldRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ResourceFieldRef field is set to the value of the last call.
func (b *DownwardAPIVolumeFileApplyConfiguration) WithResourceFieldRef(value *ResourceFieldSelectorApplyConfiguration) *DownwardAPIVolumeFileApplyConfiguration {
	b.ResourceFieldRef = value
	return b
}

// WithMode sets the Mode field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Mode field is set to the value of the last call.
func (b *DownwardAPIVolumeFileApplyConfiguration) WithMode(value int32) *DownwardAPIVolumeFileApplyConfiguration {
	b.Mode = &value
	return b
}