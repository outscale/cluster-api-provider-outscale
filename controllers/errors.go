/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"errors"
	"fmt"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type fatalError struct {
	err error
}

func AsFatalError(err error) *fatalError {
	return &fatalError{
		err: err,
	}
}

func FatalErrorf(format string, a ...any) *fatalError {
	return &fatalError{
		err: fmt.Errorf(format, a...),
	}
}

func (e *fatalError) Error() string {
	return "FATAL: " + e.err.Error()
}

func HandleFatalError(r reconcile.Result, err error) (reconcile.Result, error) {
	_, fatal := errors.AsType[*fatalError](err)
	if fatal {
		return reconcile.Result{}, nil
	}
	return r, err
}

func MarkErrorCondition(to conditions.Setter, t clusterv1.ConditionType, reason string, err error) {
	severity := clusterv1.ConditionSeverityWarning
	_, fatal := errors.AsType[*fatalError](err)
	if fatal {
		severity = clusterv1.ConditionSeverityError
	}
	conditions.MarkFalse(to, t, reason, severity, "%s", err.Error())
}
