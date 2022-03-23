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

var _ = Describe("Integration tests Build and TaskRun", func() {
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

	Context("when a build with a short ttl set and buildrun succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTL)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTL)
		})

		It("Should not find the buildrun after few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			time.Sleep(5 * time.Second)
			br, err = tb.GetBR(br.Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when a build with a short ttl set and buildrun fails", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionTTLFail)
			buildRunSample = []byte(test.MinimalBuildRunRetentionTTLFail)
		})

		It("Should not find the buildrun after few seconds after it fails", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(br.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			time.Sleep(5 * time.Second)
			br, err = tb.GetBR(br.Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when a build with a small limit set and buildrun succeeds", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimit)
			buildRunSample = []byte(test.MinimalBuildRunRetentionLimit)
		})

		It("Should not find the buildrun after few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
			_, err = tb.GetBR(buildObject.Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when a build with a small limit set and buildrun succeeds - 2 buildruns", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimit)
			buildRunSample = []byte(test.MinimalBuildRunRetentionLimit)
		})

		It("Should not find the buildrun after few seconds after it succeeds", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			//Br 1
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			Expect(err).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(br1.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))
			br1Name := br1.Name
			// Br 2
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			Expect(err).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(br2.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionTrue))

			_, err = tb.GetBR(br1Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

	})

	Context("when a build with a small limit set and buildrun fails - 2 buildruns", func() {

		BeforeEach(func() {
			buildSample = []byte(test.MinimalBuildWithRetentionLimitFail)
			buildRunSample = []byte(test.MinimalBuildRunRetentionLimitFail)
		})

		It("Should not find the buildrun after few seconds after it fails", func() {

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			// Build run - 1
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-1"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			Expect(err).To(BeNil())
			br1, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(br1.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))
			br1Name := br1.Name
			// Build run - 2
			buildRunObject.Name = BUILDRUN + tb.Namespace + "-2"
			Expect(tb.CreateBR(buildRunObject)).To(BeNil())
			Expect(err).To(BeNil())
			br2, err := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(br2.Status.GetCondition(v1alpha1.Succeeded).Status).To(Equal(corev1.ConditionFalse))

			_, err = tb.GetBR(br1Name)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

	})

})
