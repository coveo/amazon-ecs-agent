// +build windows

// Copyright 2014-2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package task

import (
	"errors"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aws/amazon-ecs-agent/agent/config"
	"github.com/aws/amazon-ecs-agent/agent/taskresource"
	taskresourcevolume "github.com/aws/amazon-ecs-agent/agent/taskresource/volume"
	"github.com/cihub/seelog"
	dockercontainer "github.com/docker/docker/api/types/container"
)

const (
	// cpuSharesPerCore represents the cpu shares of a cpu core in docker
	cpuSharesPerCore  = 1024
	percentageFactor  = 100
	minimumCPUPercent = 1
)

// PlatformFields consists of fields specific to Windows for a task
type PlatformFields struct {
	// CpuUnbounded determines whether a mix of unbounded and bounded CPU tasks
	// are allowed to run in the instance
	CpuUnbounded bool `json:"cpuUnbounded"`
}

var cpuShareScaleFactor = runtime.NumCPU() * cpuSharesPerCore

// adjustForPlatform makes Windows-specific changes to the task after unmarshal
func (task *Task) adjustForPlatform(cfg *config.Config) {
	task.downcaseAllVolumePaths()
	platformFields := PlatformFields{
		CpuUnbounded: cfg.PlatformVariables.CPUUnbounded,
	}
	task.PlatformFields = platformFields
}

// downcaseAllVolumePaths forces all volume paths (host path and container path)
// to be lower-case.  This is to account for Windows case-insensitivity and the
// case-sensitive string comparison that takes place elsewhere in the code.
func (task *Task) downcaseAllVolumePaths() {
	for _, volume := range task.Volumes {
		if hostVol, ok := volume.Volume.(*taskresourcevolume.FSHostVolume); ok {
			hostVol.FSSourcePath = getCanonicalPath(hostVol.FSSourcePath)
		}
	}
	for _, container := range task.Containers {
		for i, mountPoint := range container.MountPoints {
			// container.MountPoints is a slice of values, not a slice of pointers so
			// we need to mutate the actual value instead of the copied value
			container.MountPoints[i].ContainerPath = getCanonicalPath(mountPoint.ContainerPath)
		}
	}
}

func getCanonicalPath(path string) string {
	lowercasedPath := strings.ToLower(path)
	// if the path is a bare drive like "d:", don't filepath.Clean it because it will add a '.'.
	// this is to fix the case where mounting from D:\ to D: is supported by docker but not ecs
	if isBareDrive(lowercasedPath) {
		return lowercasedPath
	}

	if isNamedPipesPath(lowercasedPath) {
		return lowercasedPath
	}

	return filepath.Clean(lowercasedPath)
}

func isBareDrive(path string) bool {
	if filepath.VolumeName(path) == path {
		return true
	}

	return false
}

func isNamedPipesPath(path string) bool {
	matched, err := regexp.MatchString(`\\{2}\.[\\]pipe[\\].+`, path)

	if err != nil {
		return false
	}

	return matched
}

// platformHostConfigOverride provides an entry point to set up default HostConfig options to be
// passed to Docker API.
func (task *Task) platformHostConfigOverride(hostConfig *dockercontainer.HostConfig) error {
	// Convert the CPUShares to CPUPercent
	hostConfig.CPUPercent = hostConfig.CPUShares * percentageFactor / int64(cpuShareScaleFactor)
	if hostConfig.CPUPercent == 0 && hostConfig.CPUShares != 0 {
		// if CPU percent is too low, we set it to the minimum(linux and some windows tasks).
		// if the CPU is explicitly set to zero or not set at all, and CPU unbounded
		// tasks are allowed for windows, let CPU percent be zero.
		// this is a workaround to allow CPU unbounded tasks(https://github.com/aws/amazon-ecs-agent/issues/1127)
		hostConfig.CPUPercent = minimumCPUPercent
		seelog.Warnf("CPUPercent has been limited to 1 percent since hostConfig.CPUPercent is %s and hostConfig.CPUShares is %s",
			hostConfig.CPUPercent, hostConfig.CPUShares)
	}
	hostConfig.CPUShares = 0

	// As of version  17.06.2-ee-6 of docker. MemoryReservation is not supported on windows. This ensures that
	// this parameter is not passed, allowing to launch a container without a hard limit.
	hostConfig.MemoryReservation = 0
	return nil
}

// dockerCPUShares converts containerCPU shares if needed as per the logic stated below:
// Docker silently converts 0 to 1024 CPU shares, which is probably not what we
// want.  Instead, we convert 0 to 2 to be closer to expected behavior. The
// reason for 2 over 1 is that 1 is an invalid value (Linux's choice, not Docker's).
func (task *Task) dockerCPUShares(containerCPU uint) int64 {
	if containerCPU <= 1 && !task.PlatformFields.CpuUnbounded {
		seelog.Debugf(
			"Converting CPU shares to allowed minimum of 2 for task arn: [%s] and cpu shares: %d",
			task.Arn, containerCPU)
		return 2
	}
	return int64(containerCPU)
}

func (task *Task) initializeCgroupResourceSpec(cgroupPath string, cGroupCPUPeriod time.Duration, resourceFields *taskresource.ResourceFields) error {
	return errors.New("unsupported platform")
}
