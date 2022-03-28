// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/test"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("Integration tests for retention limits and ttls for succeeded buildruns.", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategyNoOp))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("When a buildrun related to a build with short ttl set succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTL)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the buildrun after few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			br, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Minute)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	// Test for editing buildrun after completion

	Context("Multiple successful buildruns related to build with limit 1.", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimit)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("The older successful buildrun should not exist.", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			//Br 1
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			br1Name := br1.Name
			// Br 2
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			_, err = tb.GetBR(br1Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

	})

	Context("Multiple buildruns with different build limits for failure and success", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitDiff)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		AfterEach(func() {
			err := tb.DeleteClusterBuildStrategy(cbsObject.Name + "-fail")
			Expect(err).To(BeNil())
		})

		It("Should not find the old buildrun if the limit has been triggered", func() {
			// var wg sync.WaitGroup
			var br1 *v1alpha1.BuildRun
			var br2 *v1alpha1.BuildRun
			var br3 *v1alpha1.BuildRun

			// Create build that will be successful using quick cbs
			Expect(tb.CreateBuild(buildObject)).To(BeNil())
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Create 2 successful buildruns
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-success-1"
			err = tb.CreateBR(buildRunObject)
			br1, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			buildRunObject.Name = BUILDRUN + tb.Namespace + "-success-2"
			err = tb.CreateBR(buildRunObject)
			br2, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			// Load the failing cbs
			cbsObjectFail, err := tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace+"-fail", []byte(test.ClusterBuildStrategySingleStepNoPush))
			Expect(err).To(BeNil())
			err = tb.CreateClusterBuildStrategy(cbsObjectFail)
			Expect(err).To(BeNil())

			// Load and create the buildobject with the relevant cbs
			buildSample = []byte(test.MinimalBuildWithRetentionLimitDiff)
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace+"-fail", STRATEGY+tb.Namespace+"-fail", buildSample)
			Expect(err).To(BeNil())
			buildObject.Spec.Source.ContextDir = nil
			Expect(tb.CreateBuild(buildObject)).To(BeNil())
			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Create 1 failed buildrun
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-fail-1"
			buildRunObject.Spec.BuildRef.Name = BUILD + tb.Namespace + "-fail"
			err = tb.CreateBR(buildRunObject)
			br3, err = tb.GetBRTillCompletion(BUILDRUN + tb.Namespace + "-fail-1")
			Expect(err).To(BeNil())
			Expect(br3.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			// Create 1 failed buildrun.
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-fail-2"
			buildRunObject.Spec.BuildRef.Name = BUILD + tb.Namespace + "-fail"
			err = tb.CreateBR(buildRunObject)
			br4, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br4.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			// Check that the older failed buildrun has been deleted while the successful buildruns exist
			_, err = tb.GetBR(BUILDRUN + tb.Namespace + "-fail-1")
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			_, err = tb.GetBR(BUILDRUN + tb.Namespace + "-success-2")
			Expect(err).To(BeNil())
			_, err = tb.GetBR(BUILDRUN + tb.Namespace + "-success-2")
			Expect(err).To(BeNil())
		})
	})

})

var _ = Describe("Integration tests for retention limits and ttls of buildRuns that fail.", func() {
	var (
		cbsObject      *v1alpha1.ClusterBuildStrategy
		buildObject    *v1alpha1.Build
		buildRunObject *v1alpha1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)
	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStepNoPush))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("When a buildrun related to a build with short ttl set succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTL)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("Should not find the buildrun after few seconds after it fails", func() {

			buildObject.Spec.Source.ContextDir = nil
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			br, err = tb.GetBRTillNotFound(buildRunObject.Name, time.Second*1, time.Minute)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("Multiple failed buildruns related to build with limit 1.", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimit)
			buildRunSample = []byte(test.MinimalBuildRunRetention)
		})

		It("The older failed buildrun should not exist.", func() {
			buildObject.Spec.Source.ContextDir = nil
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Build run - 1
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br1.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			br1Name := br1.Name
			// Build run - 2
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br2.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			_, err = tb.GetBR(br1Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

	})

})