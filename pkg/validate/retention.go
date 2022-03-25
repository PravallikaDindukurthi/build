// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// BuildNameRef contains all required fields
// to validate retention parameters
type RetentionRef struct {
	Build *build.Build // build instance for analysis
}

func (r *RetentionRef) ValidatePath(_ context.Context) error {

	// Validate if retention limit is positive
	// if r.Build.Spec.Retention != nil {
	// 	if r.Build.Spec.Retention.FailedLimit != nil && *r.Build.Spec.Retention.FailedLimit <= 0 {
	// 		r.Build.Status.Reason = build.BuildReasonPtr(build.WrongRetentionParameterType)
	// 		r.Build.Status.Message = pointer.String(fmt.Sprintf("Build Failed Limit : %d, Positive values should be provided", r.Build.Spec.Retention.FailedLimit))

	// 	}

	// 	if r.Build.Spec.Retention.SucceededLimit != nil && *r.Build.Spec.Retention.SucceededLimit <= 0 {
	// 		r.Build.Status.Reason = build.BuildReasonPtr(build.WrongRetentionParameterType)
	// 		r.Build.Status.Message = pointer.String(fmt.Sprintf("Build Suceeded Limit : %d, Positive values should be provided", r.Build.Spec.Retention.SucceededLimit))
	// 	}
	// }
	return nil
}
