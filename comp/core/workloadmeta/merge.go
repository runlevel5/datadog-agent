// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package workloadmeta

import (
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/imdario/mergo"
)

type (
	merger struct{}
)

var (
	timeType                       = reflect.TypeOf(time.Time{})
	portSliceType                  = reflect.TypeOf([]ContainerPort{})
	volumeSliceType                = reflect.TypeOf([]ContainerVolume{})
	networkSliceType               = reflect.TypeOf([]ContainerNetwork{})
	orchestratorContainerSliceType = reflect.TypeOf([]OrchestratorContainer{})
	ecsTaskTagsType                = reflect.TypeOf(ECSTaskTags{})
	containerInstanceTagsType      = reflect.TypeOf(ContainerInstanceTags{})
	mergerInstance                 = merger{}
)

func (merger) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	switch typ {
	case timeType:
		return timeMerge
	case portSliceType:
		return sliceMerge[ContainerPort](mergeContainerPort)
	case volumeSliceType:
		return sliceMerge[ContainerVolume](mergeContainerVolume)
	case networkSliceType:
		return sliceMerge[ContainerNetwork](mergeContainerNetwork)
	case orchestratorContainerSliceType:
		return sliceMerge[OrchestratorContainer](mergeOrchestratorContainer)
	case ecsTaskTagsType:
		return mapOverwrite[ECSTaskTags]
	case containerInstanceTagsType:
		return mapOverwrite[ContainerInstanceTags]
	}

	return nil
}

func timeMerge(dst, src reflect.Value) error {
	if !dst.CanSet() {
		return nil
	}

	isZero := src.MethodByName("IsZero")
	result := isZero.Call([]reflect.Value{})
	if !result[0].Bool() {
		dst.Set(src)
	}
	return nil
}

func sliceMerge[T any](mergeFunc func(mergeMap map[string]T, port T)) func(dst, src reflect.Value) error {
	return func(dst, src reflect.Value) error {
		if !dst.CanSet() {
			return nil
		}

		srcSlice := src.Interface().([]T)
		dstSlice := dst.Interface().([]T)

		// Not allocation the map if nothing to do
		if len(srcSlice) == 0 || len(dstSlice) == 0 {
			return nil
		}

		mergeMap := make(map[string]T, len(srcSlice)+len(dstSlice))
		for _, d := range dstSlice {
			mergeFunc(mergeMap, d)
		}

		for _, s := range srcSlice {
			mergeFunc(mergeMap, s)
		}

		dstSlice = make([]T, 0, len(mergeMap))
		for _, volume := range mergeMap {
			dstSlice = append(dstSlice, volume)
		}
		dst.Set(reflect.ValueOf(dstSlice))

		return nil
	}
}

func mergeContainerPort(mergeMap map[string]ContainerPort, port ContainerPort) {
	portKey := strconv.Itoa(port.Port) + port.Protocol
	existingPort, found := mergeMap[portKey]

	if found {
		if existingPort.Name == "" && port.Name != "" {
			mergeMap[portKey] = port
		}
	} else {
		mergeMap[portKey] = port
	}
}

func mergeContainerVolume(mergeMap map[string]ContainerVolume, volume ContainerVolume) {
	mergeMap[volume.Destination] = volume
}

func mergeContainerNetwork(mergeMap map[string]ContainerNetwork, network ContainerNetwork) {
	sort.Strings(network.IPv4Addresses)
	networkKey := ""
	for _, ip := range network.IPv4Addresses {
		networkKey += ip
	}

	mergeMap[networkKey] = network

}

func mergeOrchestratorContainer(mergeMap map[string]OrchestratorContainer, container OrchestratorContainer) {
	mergeMap[container.ID] = container
}

func mapOverwrite[T ECSTaskTags | ContainerInstanceTags](dst, src reflect.Value) error {
	if !dst.CanSet() {
		return nil
	}

	srcSlice := src.Interface().(T)
	dstSlice := dst.Interface().(T)

	// Not allocation the map if nothing to do
	if len(srcSlice) == 0 || len(dstSlice) == 0 {
		return nil
	}

	mergeMap := make(map[string]string, len(srcSlice))

	for k, v := range srcSlice {
		mergeMap[k] = v
	}

	dst.Set(reflect.ValueOf(mergeMap))

	return nil
}

func merge(dst, src interface{}) error {
	return mergo.Merge(dst, src, mergo.WithAppendSlice, mergo.WithTransformers(mergerInstance))
}
