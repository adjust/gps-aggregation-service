// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package incrementaldpf calls the C++ incremental DPF functions from Go.
package incrementaldpf

// #cgo CFLAGS: -g -Wall
// #include <stdbool.h>
// #include <stdlib.h>
// #include "pipeline/distributed_point_function_c_bridge.h"
import (
	"C"
)

import (
	"errors"
	"fmt"
	"unsafe"

	"google.golang.org/protobuf/proto"

	pb "github.com/google/distributed_point_functions/dpf/distributed_point_function_go_proto"
)

func freeCBytes(cb C.struct_CBytes) {
	C.free(unsafe.Pointer(cb.c))
}

func createCParams(params []*pb.DpfParameters) (*C.struct_CBytes, []unsafe.Pointer, error) {
	paramsLen := len(params)
	cParamPointers := make([]unsafe.Pointer, paramsLen)
	cParams := (*C.struct_CBytes)(C.malloc(C.sizeof_struct_CBytes * C.uint64_t(paramsLen)))
	pSlice := (*[1 << 30]C.struct_CBytes)(unsafe.Pointer(cParams))[:paramsLen:paramsLen]
	for i, p := range params {
		bParam, err := proto.Marshal(p)
		if err != nil {
			return nil, nil, err
		}
		cParamPointers[i] = C.CBytes(bParam)
		pSlice[i] = C.struct_CBytes{c: (*C.char)(cParamPointers[i]), l: C.int(len(bParam))}
	}
	return cParams, cParamPointers, nil
}

func freeCParams(cParams *C.struct_CBytes, cParamPointers []unsafe.Pointer) {
	for _, p := range cParamPointers {
		C.free(p)
	}
	C.free(unsafe.Pointer(cParams))
}

// GenerateKeys generates a pair of DpfKeys for given parameters.
func GenerateKeys(params []*pb.DpfParameters, alpha uint64, betas []uint64) (*pb.DpfKey, *pb.DpfKey, error) {
	cParamsSize := C.int64_t(len(params))
	cParams, cParamPointers, err := createCParams(params)
	defer freeCParams(cParams, cParamPointers)
	if err != nil {
		return nil, nil, err
	}

	betasSize := len(betas)
	cBetasSize := C.int64_t(betasSize)
	var betasPointer *uint64
	if betasSize > 0 {
		betasPointer = &betas[0]
	}

	cKey1 := C.struct_CBytes{}
	cKey2 := C.struct_CBytes{}
	errStr := C.struct_CBytes{}
	status := C.CGenerateKeys(cParams, cParamsSize, C.uint64_t(alpha), (*C.uint64_t)(unsafe.Pointer(betasPointer)), cBetasSize, &cKey1, &cKey2, &errStr)
	defer freeCBytes(cKey1)
	defer freeCBytes(cKey2)
	defer freeCBytes(errStr)
	if status != 0 {
		return nil, nil, errors.New(C.GoStringN(errStr.c, errStr.l))
	}

	key1 := &pb.DpfKey{}
	if err := proto.Unmarshal(C.GoBytes(unsafe.Pointer(cKey1.c), cKey1.l), key1); err != nil {
		return nil, nil, err
	}

	key2 := &pb.DpfKey{}
	if err := proto.Unmarshal(C.GoBytes(unsafe.Pointer(cKey2.c), cKey2.l), key2); err != nil {
		return nil, nil, err
	}

	return key1, key2, nil
}

// CreateEvaluationContext creates the context for expanding the vectors.
func CreateEvaluationContext(params []*pb.DpfParameters, key *pb.DpfKey) (*pb.EvaluationContext, error) {
	cParamsSize := C.int64_t(len(params))
	cParams, cParamPointers, err := createCParams(params)
	defer freeCParams(cParams, cParamPointers)
	if err != nil {
		return nil, err
	}

	bKey, err := proto.Marshal(key)
	if err != nil {
		return nil, err
	}
	cKey := C.struct_CBytes{c: (*C.char)(C.CBytes(bKey)), l: C.int(len(bKey))}
	defer freeCBytes(cKey)

	cEvalCtx := C.struct_CBytes{}
	errStr := C.struct_CBytes{}
	status := C.CCreateEvaluationContext(cParams, cParamsSize, &cKey, &cEvalCtx, &errStr)
	defer freeCBytes(cEvalCtx)
	defer freeCBytes(errStr)
	if status != 0 {
		return nil, errors.New(C.GoStringN(errStr.c, errStr.l))
	}

	evalCtx := &pb.EvaluationContext{}
	if err := proto.Unmarshal(C.GoBytes(unsafe.Pointer(cEvalCtx.c), cEvalCtx.l), evalCtx); err != nil {
		return nil, err
	}
	return evalCtx, nil
}

// EvaluateNext64 evaluates the given DPF key in the evaluation context with the specified configuration.
func EvaluateNext64(prefixes []uint64, evalCtx *pb.EvaluationContext) ([]uint64, error) {
	pSize := len(prefixes)
	cPrefixesSize := C.int64_t(pSize)
	var prefixesPointer *uint64
	if pSize > 0 {
		prefixesPointer = &prefixes[0]
	}

	bEvalCtx, err := proto.Marshal(evalCtx)
	if err != nil {
		return nil, err
	}
	cEvalCtx := C.struct_CBytes{c: (*C.char)(C.CBytes(bEvalCtx)), l: C.int(len(bEvalCtx))}
	outExpanded := C.struct_CUInt64Vec{}
	errStr := C.struct_CBytes{}
	status := C.CEvaluateNext64((*C.uint64_t)(unsafe.Pointer(prefixesPointer)), cPrefixesSize, &cEvalCtx, &outExpanded, &errStr)
	defer freeCBytes(cEvalCtx)
	defer C.free(unsafe.Pointer(outExpanded.vec))
	defer freeCBytes(errStr)
	if status != 0 {
		return nil, errors.New(C.GoStringN(errStr.c, errStr.l))
	}

	if err := proto.Unmarshal(C.GoBytes(unsafe.Pointer(cEvalCtx.c), cEvalCtx.l), evalCtx); err != nil {
		return nil, err
	}

	const maxLen = 1 << 30
	vecLen := uint64(outExpanded.vec_size)
	if vecLen > maxLen {
		return nil, fmt.Errorf("vector length %d should not exceed %d", vecLen, maxLen)
	}
	es := (*[maxLen]C.uint64_t)(unsafe.Pointer(outExpanded.vec))[:vecLen:vecLen]
	expanded := make([]uint64, vecLen)
	for i := uint64(0); i < uint64(vecLen); i++ {
		expanded[i] = uint64(es[i])
	}
	return expanded, nil
}

// CalculateBucketID gets the bucket ID for values in the expanded vectors.
func CalculateBucketID(params []*pb.DpfParameters, prefixes [][]uint64) ([]uint64, error) {
	if err := CheckExpansionParameters(params, prefixes); err != nil {
		return nil, err
	}

	// For direct expansion, return empty slice to avoid generating extra data.
	// Because in this case, the bucket ID equals the vector index.
	if len(params) == 1 {
		return nil, nil
	}

	paramsLen := len(params)
	expansionBits := params[paramsLen-1].GetLogDomainSize() - params[paramsLen-2].GetLogDomainSize()
	expansionSize := uint64(1) << expansionBits
	ids := make([]uint64, uint64(len(prefixes[paramsLen-1]))*expansionSize)
	i := uint64(0)
	for _, p := range prefixes[paramsLen-1] {
		prefix := p << uint64(expansionBits)
		for j := uint64(0); j < expansionSize; j++ {
			ids[i] = prefix | j
			i++
		}
	}
	return ids, nil
}

// CheckExpansionParameters checks if the DPF parameters and prefixes are valid for the hierarchical expansion.
func CheckExpansionParameters(params []*pb.DpfParameters, prefixes [][]uint64) error {
	paramsLen := len(params)
	if paramsLen == 0 {
		return errors.New("empty dpf parameters")
	}

	prefixesLen := len(prefixes)
	if paramsLen != prefixesLen {
		return fmt.Errorf("dpf parameter size should equal prefixes size %d, got %d", prefixesLen, paramsLen)
	}

	if len(prefixes[0]) != 0 {
		return fmt.Errorf("prefixes should be empty for the first level expansion, got %v", prefixes[0])
	}

	for i, p := range prefixes {
		if i > 0 && len(p) == 0 {
			return fmt.Errorf("prefix cannot be empty except for the top level expansion")
		}
	}
	return nil
}
