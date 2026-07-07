package clickhouse

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/ClickHouse/clickhouse-operator/api/v1alpha1"
)

var _ = Describe("ConfigGenerator", func() {
	ctx := clickhouseReconciler{
		Cluster: &v1.ClickHouseCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "test-namespace",
			},
			Spec: v1.ClickHouseClusterSpec{
				Replicas:            new(int32(3)),
				Shards:              new(int32(2)),
				DataVolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{},
				AdditionalVolumeClaimTemplates: []v1.PersistentVolumeClaimTemplate{
					{NamedTemplateMeta: v1.NamedTemplateMeta{Name: "test1"}},
					{NamedTemplateMeta: v1.NamedTemplateMeta{Name: "test2"}},
				},
				Settings: v1.ClickHouseSettings{
					ExtraConfig: runtime.RawExtension{
						Raw: []byte(`{"test": "value"}`),
					},
					ExtraUsersConfig: runtime.RawExtension{
						Raw: []byte(`{}`),
					},
				},
			},
			Status: v1.ClickHouseClusterStatus{
				Version: "25.12.1.1",
			},
		},
		keeper: v1.KeeperCluster{
			Spec: v1.KeeperClusterSpec{
				Replicas: new(int32(3)),
			},
		},
	}

	for _, generator := range generators {
		It("should generate config: "+generator.Filename(), func() {
			Expect(generator.Enabled(&ctx)).To(BeTrue())
			data, err := generator.Generate(&ctx, v1.ClickHouseReplicaID{})
			Expect(err).ToNot(HaveOccurred())

			obj := map[any]any{}
			Expect(yaml.Unmarshal([]byte(data), &obj)).To(Succeed())
		})
	}
})

var _ = Describe("networkConfigGenerator listen_host", func() {
	networkGenerator := func() configGenerator {
		for _, g := range generators {
			if g.Filename() == "00-network.yaml" {
				return g
			}
		}

		return nil
	}

	newReconciler := func(listenHost []string) *clickhouseReconciler {
		return &clickhouseReconciler{
			Cluster: &v1.ClickHouseCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
				Spec: v1.ClickHouseClusterSpec{
					Replicas:            new(int32(1)),
					Shards:              new(int32(1)),
					DataVolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{},
					Settings: v1.ClickHouseSettings{
						Network: v1.NetworkSettings{ListenHost: listenHost},
					},
				},
				Status: v1.ClickHouseClusterStatus{Version: "25.12.1.1"},
			},
			keeper: v1.KeeperCluster{Spec: v1.KeeperClusterSpec{Replicas: new(int32(1))}},
		}
	}

	generateListenHost := func(listenHost []string) []any {
		gen := networkGenerator()
		Expect(gen).ToNot(BeNil())

		data, err := gen.Generate(newReconciler(listenHost), v1.ClickHouseReplicaID{})
		Expect(err).ToNot(HaveOccurred())

		obj := map[string]any{}
		Expect(yaml.Unmarshal([]byte(data), &obj)).To(Succeed())
		Expect(obj).To(HaveKey("listen_host"))

		hosts, ok := obj["listen_host"].([]any)
		Expect(ok).To(BeTrue())

		return hosts
	}

	It("defaults to dual-stack when listenHost is empty", func() {
		Expect(generateListenHost(nil)).To(Equal([]any{"::", "0.0.0.0"}))
	})

	It("honors a custom IPv4-only listenHost override", func() {
		Expect(generateListenHost([]string{"0.0.0.0"})).To(Equal([]any{"0.0.0.0"}))
	})
})
