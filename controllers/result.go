/*
SPDX-FileCopyrightText: 2026 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"fmt"
	"time"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ResultStatus int

const (
	ResultStatusUnknown = iota
	ResultStatusNotOK
	ResultStatusOK
)

type Result struct {
	Status ResultStatus
	Error  error
}

func (r *Result) CheckResultAndConditions(o conditions.Getter, onNOK func(error), onOK func(), msg string) (stop bool, result reconcile.Result, err error) {
	// set conditions
	existing := conditions.Get(o, clusterv1.ReadyCondition)
	switch r.Status {
	case ResultStatusOK:
		if onOK != nil {
			onOK()
		}
	case ResultStatusNotOK:
		if onNOK != nil {
			onNOK(r.Error)
		}
	default:
		if existing == nil && onNOK != nil {
			onNOK(r.Error)
		}
	}
	switch {
	case r.Error != nil:
		return true, reconcile.Result{}, fmt.Errorf("reconcile %s: %w", msg, r.Error)
	case r.Status == ResultStatusNotOK:
		return true, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	default:
		return false, reconcile.Result{}, nil
	}
}
