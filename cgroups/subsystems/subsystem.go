package subsystems

// ResourceConfig 用于传递资源限制配置的结构体，包含内存限制，CPU时间片权重，CPU核心数
type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

type Subsystem interface {
	Name() string
	// 这里将cgroup抽象成了path，原因是cgroup在hierarchy的路径，就是虚拟文件系统中的虚拟路径
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int) error
	Remove(path string) error
}

// SubsystemsIns 通过不同的subsystem初始化实例创建资源限制处理链接数组
var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)
