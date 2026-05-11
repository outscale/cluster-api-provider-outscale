package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func MarkFalse(obj conditions.Setter, condition clusterv1.ConditionType, reason string) {
	conditions.Set(obj, metav1.Condition{
		Type:   string(condition),
		Status: conditions.BoolToStatus(false),
		Reason: reason,
	})
}
