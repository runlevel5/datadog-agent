// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build (windows && npm) || linux_bpf

package http

import "sync"

var transactionSlicePool = sync.Pool{
	New: func() any {
		t := make([]Transaction, 0, 512)
		return &t
	},
}

// this type constraint is only used below to generate a []Transaction
// from multiple concrete implementations
type transaction[T any] interface {
	Transaction
	*T
}

// GetTransactionSlice converts a slice of type []T into a slice of type
// []Transaction (assuming that *T implements the Transaction interface).  Given
// that this code called from the hot path of HTTP monitoring, we make use of an
// object pool to recycle the []Transaction slices. It's the caller
// responsibility to call the returned `func()`, once the contents of the slice
// are used/copied.
//
// Note: The generic stuff is definitely more gnarly that I wished, but I
// couldn't find a simpler way to do it. Given that we have at least 3 different
// places using this with 3 distinct types, I thought it was worth it.
//
// Here's an example how you can call it:
// Assuming you have an `events` variable of type `EbpfEvent`,
//
// transactions, done := GetTransactionSlice[EbpfEvent, *EbpfEvent](events)
// ... do stuff with transactions
// done()
//
// A similar example can be found in the Golang's type paramaters spec:
// https://go.googlesource.com/proposal/+/HEAD/design/43651-type-parameters.md#pointer-method-example
func GetTransactionSlice[T any, PT transaction[T]](elements []T) ([]Transaction, func()) {
	p := transactionSlicePool.Get().(*[]Transaction)
	result := (*p)[:0]
	for i := range elements {
		result = append(result, PT(&elements[i]))
	}

	reclaimCB := func() {
		transactionSlicePool.Put(p)
	}

	return result, reclaimCB
}
