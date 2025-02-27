package irq

import (
	"time"

	"github.com/kubewharf/katalyst-api/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type ContainerInfo struct {
	PodUID      string
	ContainerID string
	CgroupPath  string // 相对路径 /kubepods/burstable/podUID/containerID

	QosLevel    string
	Annotations map[string]string // enhancements

	RuntimeKind  string // pod spec runtime class
	StartedAt    time.Time
	ActualCPUSet map[int]machine.CPUSet // 实时绑定的cpuset，分numa
}

type IRQTuningInput struct {
	IRQForbiddenCores machine.CPUSet
	Containers        []ContainerInfo
}

type IrqStateAdapter interface {
	// 只返回running的容器，一般不会失败
	ListContainers() ([]IRQTuningInput, error)
	// 仅将结果同步到cpu plugin checkpoint中，实际中断绑核由irq tuning manager通过procfs完成
	SetIrqCpuset(cpuset machine.CPUSet) error
}

func example() {
	input := IRQTuningInput{
		IRQForbiddenCores: machine.NewCPUSet(0, 1),
		Containers: []ContainerInfo{
			{
				PodUID:      "uid1",
				ContainerID: "container-id1",
				CgroupPath:  "/kubepods/burstable/pod-uid1/container-id1",
				QosLevel:    string(consts.QoSLevelDedicatedCores),
				Annotations: map[string]string{
					consts.PodAnnotationMemoryEnhancementNumaBindingEnable: consts.PodAnnotationMemoryEnhancementNumaBindingEnable,
					consts.PodAnnotationMemoryEnhancementNumaExclusive:     consts.PodAnnotationMemoryEnhancementNumaExclusiveEnable,
					"bytedance.com/kata-bm":                                "true",
				},
				RuntimeKind: "kata-clh",
				StartedAt:   time.Now(),
				ActualCPUSet: map[int]machine.CPUSet{
					0: machine.NewCPUSet(2, 3, 4),
					1: machine.NewCPUSet(5, 6, 7),
				},
			},
		},
	}

	fmt.Println(input)
}
